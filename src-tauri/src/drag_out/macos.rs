//! macOS drag-out 实现（vendor 自 `drag` v2.1.1 的 macOS 路径，做两点改造）：
//!
//! 1. **预览图 DPI 解耦**：固定调 `NSImage::setSize(POINT_SIZE × POINT_SIZE)`，
//!    把高分辨率 PNG（如 256px）当作 Retina @2x 渲染到 [`POINT_SIZE`] pt 的显示框里——
//!    跟随光标的视觉大小由 `POINT_SIZE` 决定，清晰度由源 PNG 像素决定，互相独立。
//!    `drag` crate 用 `img.size()` 原样作为显示尺寸（pixels == points），
//!    256px PNG 飞起来就有 256pt 那么大，无法兼顾「清晰 + 不过大」。
//! 2. **直接用 `WebviewWindow::ns_view`**，不再依赖 `raw-window-handle`，减少一个外部 crate。
//!
//! 其余（NSDraggingItem / NSPasteboardItem / DragRsSource 等）一比一照搬，
//! 仅做 use 路径整理。

use std::ffi::c_void;
use std::path::PathBuf;

use objc2::rc::Retained;
use objc2::runtime::{NSObject, NSObjectProtocol, ProtocolObject};
use objc2::{define_class, msg_send, AnyThread, DefinedClass, MainThreadMarker, MainThreadOnly};
use objc2_app_kit::{
    NSApp, NSAttributedStringNSStringDrawing, NSBezierPath, NSColor, NSDraggingContext,
    NSDraggingItem, NSDraggingSession, NSDraggingSource, NSEvent, NSEventModifierFlags,
    NSEventType, NSFont, NSFontAttributeName, NSForegroundColorAttributeName, NSImage,
    NSMutableParagraphStyle, NSParagraphStyleAttributeName, NSPasteboardItem, NSView,
};
use objc2_foundation::{
    NSAttributedString, NSDictionary, NSMutableArray, NSPoint, NSRect, NSSize, NSString, NSURL,
};
use tauri::WebviewWindow;

use crate::core::{AppError, Result};

/// 拖拽预览图的显示尺寸（pt）。给到的源 PNG 像素 = `POINT_SIZE * scale`（Retina 屏 scale=2）
/// 才能保证不模糊；目前上层固定传 256px PNG，对应 @2x 下 128pt 显示框。
const POINT_SIZE: f64 = 128.0;

/// `public.utf8-plain-text` UTI；NSPasteboard 自带的 UTF-8 文本类型。
const UTI_UTF8_PLAIN_TEXT: &str = "public.utf8-plain-text";
/// `public.html` UTI；NSPasteboard 自带的 HTML 类型。
const UTI_HTML: &str = "public.html";
/// `public.rtf` UTI；NSPasteboard 自带的 RTF 类型。
const UTI_RTF: &str = "public.rtf";

type OnDropCallback = Box<dyn Fn(DragResult) + Send>;

#[derive(Debug, Clone, Copy)]
pub enum DragResult {
    Dropped,
    Cancel,
}

define_class!(
    #[unsafe(super(NSObject))]
    #[thread_kind = MainThreadOnly]
    #[name = "EcoPasteDragSource"]
    #[ivars = DragSourceIvars]
    struct DragSource;

    unsafe impl NSObjectProtocol for DragSource {}

    unsafe impl NSDraggingSource for DragSource {
        #[unsafe(method(draggingSession:sourceOperationMaskForDraggingContext:))]
        unsafe fn dragging_session(
            &self,
            _session: &NSDraggingSession,
            _context: NSDraggingContext,
        ) -> objc2_app_kit::NSDragOperation {
            objc2_app_kit::NSDragOperation::Copy
        }

        #[unsafe(method(draggingSession:endedAtPoint:operation:))]
        unsafe fn dragging_session_end(
            &self,
            _session: &NSDraggingSession,
            _ended_at_point: NSPoint,
            operation: objc2_app_kit::NSDragOperation,
        ) {
            let callback = &self.ivars().on_drop;
            if operation == objc2_app_kit::NSDragOperation::None {
                (callback)(DragResult::Cancel);
            } else {
                (callback)(DragResult::Dropped);
            }
        }
    }
);

struct DragSourceIvars {
    on_drop: OnDropCallback,
}

impl DragSource {
    fn new<F: Fn(DragResult) + Send + 'static>(
        on_drop: F,
        mtm: MainThreadMarker,
    ) -> Retained<Self> {
        let this = Self::alloc(mtm).set_ivars(DragSourceIvars {
            on_drop: Box::new(on_drop),
        });
        unsafe { msg_send![super(this), init] }
    }
}

/// 在 `window` 上启动一次文件 drag-out。
///
/// `preview_png` 是高分辨率源图字节（推荐 256px），会被 [`POINT_SIZE`] 归一为显示框；
/// 为 `None` 时退回用首个路径让 `NSImage` 自己 referencingFile（图片 / PDF 能出 QuickLook 缩略图，
/// 其余文件类型可能为空白）。
///
/// **必须在主线程调用**。调用方（命令层）用 `app.run_on_main_thread` 派发。
pub fn start_drag_files<F: Fn(DragResult) + Send + 'static>(
    window: &WebviewWindow,
    paths: Vec<PathBuf>,
    preview_png: Option<Vec<u8>>,
    on_drop: F,
) -> Result<()> {
    if paths.is_empty() {
        return Err(AppError::Clipboard("drag-out: empty path list".to_string()));
    }

    unsafe {
        let img = build_preview_image(preview_png, paths.first())?;

        let dragging_items = NSMutableArray::new();
        for path in &paths {
            let nsurl = NSURL::fileURLWithPath_isDirectory(
                &NSString::from_str(&path.display().to_string()),
                false,
            );
            let item = NSDraggingItem::initWithPasteboardWriter(
                NSDraggingItem::alloc(),
                &ProtocolObject::from_retained(nsurl),
            );
            dragging_items.addObject(&*item);
        }

        begin_drag_session(window, &dragging_items, &img, on_drop)
    }
}

/// 在 `window` 上启动一次文本 drag-out（plain + 可选 html / rtf）。
///
/// 同一个 `NSPasteboardItem` 多次 `setString:forType:`，AppKit 会让接收方按偏好选格式：
/// Word / Pages 优先 RTF，浏览器 / 富文本编辑器优先 HTML，纯文本 app 退回 plain。
///
/// **必须在主线程调用**。
pub fn start_drag_text<F: Fn(DragResult) + Send + 'static>(
    window: &WebviewWindow,
    plain: String,
    html: Option<String>,
    rtf: Option<String>,
    preview_png: Option<Vec<u8>>,
    on_drop: F,
) -> Result<()> {
    if plain.is_empty() {
        return Err(AppError::Clipboard("drag-out: empty text".to_string()));
    }

    unsafe {
        // 优先用上层传入的 PNG；缺失或解码失败时用 plain 现场渲染一张文本卡，
        // 比通用 app 图标更有辨识度（参考 Safari 拖文字的预览）。
        let img = build_preview_image(preview_png, None)
            .unwrap_or_else(|_| render_text_preview_image(&plain));

        let pb_item = NSPasteboardItem::new();
        let _ = pb_item.setString_forType(
            &NSString::from_str(&plain),
            &NSString::from_str(UTI_UTF8_PLAIN_TEXT),
        );
        if let Some(html) = html.as_deref() {
            let _ =
                pb_item.setString_forType(&NSString::from_str(html), &NSString::from_str(UTI_HTML));
        }
        if let Some(rtf) = rtf.as_deref() {
            let _ =
                pb_item.setString_forType(&NSString::from_str(rtf), &NSString::from_str(UTI_RTF));
        }

        let dragging_items = NSMutableArray::new();
        let item = NSDraggingItem::initWithPasteboardWriter(
            NSDraggingItem::alloc(),
            &ProtocolObject::from_retained(pb_item),
        );
        dragging_items.addObject(&*item);

        begin_drag_session(window, &dragging_items, &img, on_drop)
    }
}

/// 统一的 dragging session 启动：取 ns_view / contentView，构造 mouseDragged NSEvent，
/// 给所有 dragging item 设同一张预览图（居中于光标），调 `beginDraggingSession`。
unsafe fn begin_drag_session<F: Fn(DragResult) + Send + 'static>(
    window: &WebviewWindow,
    dragging_items: &NSMutableArray<NSDraggingItem>,
    img: &NSImage,
    on_drop: F,
) -> Result<()> {
    let mtm = MainThreadMarker::new()
        .ok_or_else(|| AppError::Clipboard("start_drag must run on main thread".to_string()))?;

    let ns_view_ptr = window
        .ns_view()
        .map_err(|err| AppError::Clipboard(format!("get ns_view failed: {err}")))?;
    let ns_view = &*(ns_view_ptr as *const c_void as *const NSView);
    let ns_window = ns_view
        .window()
        .ok_or_else(|| AppError::Clipboard("ns_view has no window".to_string()))?;
    let content_view = ns_window
        .contentView()
        .ok_or_else(|| AppError::Clipboard("ns_window has no contentView".to_string()))?;

    let cursor: NSPoint = ns_window.mouseLocationOutsideOfEventStream();

    // 关键：把 NSImage 的逻辑 size 归一化——长边固定为 POINT_SIZE pt，短边按原 PNG 比例缩放，
    // 与源 PNG 像素解耦。对正方形源图（如 OS 文件图标）就是 POINT_SIZE×POINT_SIZE；
    // 对长矩形（如卡片截图）保持原比例不压扁。
    let raw = img.size();
    let (disp_w, disp_h) = if raw.width > 0.0 && raw.height > 0.0 {
        let longest = raw.width.max(raw.height);
        let scale = POINT_SIZE / longest;
        (raw.width * scale, raw.height * scale)
    } else {
        (POINT_SIZE, POINT_SIZE)
    };
    img.setSize(NSSize::new(disp_w, disp_h));

    let image_rect = NSRect::new(
        NSPoint::new(cursor.x - disp_w / 2.0, cursor.y - disp_h / 2.0),
        NSSize::new(disp_w, disp_h),
    );
    for i in 0..dragging_items.count() {
        let item = dragging_items.objectAtIndex(i);
        item.setDraggingFrame_contents(image_rect, Some(img));
    }

    let current_event = NSApp(mtm).currentEvent();
    let timestamp = current_event.map(|e| e.timestamp()).unwrap_or(0.0);
    let window_number = ns_window.windowNumber();

    let drag_event = NSEvent::mouseEventWithType_location_modifierFlags_timestamp_windowNumber_context_eventNumber_clickCount_pressure(
        NSEventType::LeftMouseDragged,
        cursor,
        NSEventModifierFlags::empty(),
        timestamp,
        window_number,
        None,
        0,
        1,
        1.0,
    ).ok_or_else(|| AppError::Clipboard("create NSEvent failed".to_string()))?;

    let source = DragSource::new(on_drop, mtm);

    let _ = content_view.beginDraggingSessionWithItems_event_source(
        dragging_items,
        &drag_event,
        &ProtocolObject::<dyn NSDraggingSource>::from_retained(source),
    );

    Ok(())
}

/// 优先用传入的 PNG 字节；为空或解码失败时退回用首个路径 `initByReferencingFile`。
unsafe fn build_preview_image(
    preview_png: Option<Vec<u8>>,
    fallback_path: Option<&PathBuf>,
) -> Result<Retained<NSImage>> {
    if let Some(bytes) = preview_png {
        let data = objc2_foundation::NSData::from_vec(bytes);
        if let Some(img) = NSImage::initWithData(NSImage::alloc(), &data) {
            return Ok(img);
        }
    }

    let path =
        fallback_path.ok_or_else(|| AppError::Clipboard("no preview image source".to_string()))?;
    NSImage::initByReferencingFile(
        NSImage::alloc(),
        &NSString::from_str(&path.to_string_lossy()),
    )
    .ok_or_else(|| AppError::Clipboard("NSImage init failed".to_string()))
}

/// 文本预览图边长（pt，正方形）。外层 `begin_drag_session` 会按比例缩到 `POINT_SIZE`；
/// 这里画在 2× `POINT_SIZE` 的 NSImage 上，让光标跟随预览保持 Retina 清晰度。
const TEXT_PREVIEW_PT: f64 = POINT_SIZE * 2.0;
/// 文本预览中最多显示的字符数；超出截断 + 加省略号，避免极长内容拖慢 layout。
const TEXT_PREVIEW_MAX_CHARS: usize = 280;

/// 把 `text` 现场渲染成一张文本卡片 NSImage 用作 drag 预览，
/// 视觉模仿 Safari 拖选中文字（白底圆角 + 黑色正文 + 边距）。
///
/// 实现路径：`NSImage::lockFocus` → 填白底 → `NSAttributedString::drawInRect`。
/// 必须在主线程（`begin_drag_session` 同侧）调用。
#[allow(deprecated)] // lockFocus/unlockFocus 仍可用；block API（imageWithSize:flipped:drawingHandler:）在 Rust 侧写起来过重。
unsafe fn render_text_preview_image(text: &str) -> Retained<NSImage> {
    let size = NSSize::new(TEXT_PREVIEW_PT, TEXT_PREVIEW_PT);
    let img = NSImage::initWithSize(NSImage::alloc(), size);

    let snippet = clamp_text(text, TEXT_PREVIEW_MAX_CHARS);
    let ns_text = NSString::from_str(&snippet);

    // 字号按预览框估算：约 7 行可见，留点呼吸感。
    let font = NSFont::systemFontOfSize(20.0);
    let color = NSColor::labelColor();
    let para = NSMutableParagraphStyle::new();
    para.setLineBreakMode(objc2_app_kit::NSLineBreakMode::ByWordWrapping);

    let keys: [&objc2_foundation::NSAttributedStringKey; 3] = [
        NSFontAttributeName,
        NSForegroundColorAttributeName,
        NSParagraphStyleAttributeName,
    ];
    let values: [Retained<objc2::runtime::AnyObject>; 3] = [
        Retained::cast_unchecked(font),
        Retained::cast_unchecked(color),
        Retained::cast_unchecked(para),
    ];
    let attrs = NSDictionary::from_retained_objects(&keys, &values);

    let attr_str = NSAttributedString::initWithString_attributes(
        NSAttributedString::alloc(),
        &ns_text,
        Some(&attrs),
    );

    img.lockFocus();

    // 圆角白卡：先按圆角路径裁剪，再填背景色，文字绘制自然落在圆角内。
    // 颜色用 NSColor 的语义色（textBackgroundColor / labelColor），
    // 跟随系统外观自动深浅，与 app 内主题切换的视觉一致。
    let full_rect = NSRect::new(NSPoint::ZERO, size);
    let corner_radius = 20.0;
    let clip_path = NSBezierPath::bezierPathWithRoundedRect_xRadius_yRadius(
        full_rect,
        corner_radius,
        corner_radius,
    );
    clip_path.addClip();

    let bg = NSColor::textBackgroundColor();
    bg.setFill();
    clip_path.fill();

    // 文本绘制区域：四周留 16pt 边距
    let padding = 16.0;
    let text_rect = NSRect::new(
        NSPoint::new(padding, padding),
        NSSize::new(size.width - padding * 2.0, size.height - padding * 2.0),
    );
    attr_str.drawInRect(text_rect);

    img.unlockFocus();

    img
}

/// 截断 + 折行规整：超长的添省略号；连续空行压成单个换行，避免预览全是空白。
fn clamp_text(text: &str, max_chars: usize) -> String {
    let mut out = String::new();
    let mut last_was_newline = false;
    for ch in text.chars() {
        if out.chars().count() >= max_chars {
            out.push('…');
            break;
        }
        if ch == '\n' {
            if last_was_newline {
                continue;
            }
            last_was_newline = true;
            out.push('\n');
        } else if ch == '\r' {
            continue;
        } else {
            last_was_newline = false;
            out.push(ch);
        }
    }
    out
}

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
    NSApp, NSDraggingContext, NSDraggingItem, NSDraggingSession, NSDraggingSource, NSEvent,
    NSEventModifierFlags, NSEventType, NSImage, NSPasteboardItem, NSView,
};
use objc2_foundation::{NSMutableArray, NSPoint, NSRect, NSSize, NSString, NSURL};
use tauri::WebviewWindow;

use crate::core::{AppError, Result};

/// 拖拽预览图的显示尺寸（pt）。给到的源 PNG 像素 = `POINT_SIZE * scale`（Retina 屏 scale=2）
/// 才能保证不模糊；目前上层固定传 256px PNG，对应 @2x 下 128pt 显示框。
const POINT_SIZE: f64 = 128.0;

/// `public.utf8-plain-text` UTI；NSPasteboard 自带的 UTF-8 文本类型。
const UTI_UTF8_PLAIN_TEXT: &str = "public.utf8-plain-text";

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

/// 在 `window` 上启动一次纯文本 drag-out。
///
/// `preview_png` 同 [`start_drag_files`]，缺省时退回一张全透明 1×1 NSImage（AppKit 仍接受）。
///
/// **必须在主线程调用**。
pub fn start_drag_text<F: Fn(DragResult) + Send + 'static>(
    window: &WebviewWindow,
    text: String,
    preview_png: Option<Vec<u8>>,
    on_drop: F,
) -> Result<()> {
    if text.is_empty() {
        return Err(AppError::Clipboard("drag-out: empty text".to_string()));
    }

    unsafe {
        let img = build_preview_image(preview_png, None).unwrap_or_else(|_| NSImage::new());

        let pb_item = NSPasteboardItem::new();
        let _ = pb_item.setString_forType(
            &NSString::from_str(&text),
            &NSString::from_str(UTI_UTF8_PLAIN_TEXT),
        );

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

    // 关键：把 NSImage 的逻辑 size 固定为 POINT_SIZE pt（与源 PNG 像素解耦）。
    img.setSize(NSSize::new(POINT_SIZE, POINT_SIZE));

    let image_rect = NSRect::new(
        NSPoint::new(cursor.x - POINT_SIZE / 2.0, cursor.y - POINT_SIZE / 2.0),
        NSSize::new(POINT_SIZE, POINT_SIZE),
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

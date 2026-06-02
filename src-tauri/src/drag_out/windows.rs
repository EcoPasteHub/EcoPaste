//! Windows drag-out 实现。
//!
//! - **文件**（`start_drag_files`）：直接复用 `drag` crate v2.1.1
//!   （`IDataObject(CF_HDROP)` + `DoDragDrop`），稳定且自带 `IDragSourceHelper` 预览。
//! - **文本 / 富文本**（`start_drag_text`）：自实现 `IDataObject`，支持
//!   `CF_UNICODETEXT` + `CF_HTML`（注册名 "HTML Format"）+ `Rich Text Format`，
//!   接收方按偏好选格式（Word 优先 RTF，浏览器优先 HTML，纯文本退回 plain）。
//!
//! `CF_HTML` / `Rich Text Format` 不是系统常量，需 `RegisterClipboardFormatW` 注册
//! 拿 cfid；用 `OnceLock` 进程级缓存。CF_HTML 的 payload 必须带 Microsoft 定义的
//! header（`Version:0.9\r\nStartHTML:...`），否则 Word / Outlook 不识别。
//!
//! 通用约束：所有入口**必须在拥有窗口的线程上调用**（= Tauri 主线程），否则
//! `IDropSource::QueryContinueDrag` 会立刻返回 `DRAGDROP_S_CANCEL`，拖拽秒取消。
//!
//! `OleInitialize` 用进程级 `Once` 兜底，全程不 Uninitialize。

use std::iter::once;
use std::os::windows::ffi::OsStrExt;
use std::path::PathBuf;
use std::sync::{Once, OnceLock};

use tauri::WebviewWindow;
use windows::core::{implement, Error as WinError, HRESULT, HSTRING, PCWSTR};
use windows::Win32::Foundation::{
    BOOL, COLORREF, DRAGDROP_S_CANCEL, DRAGDROP_S_DROP, DRAGDROP_S_USEDEFAULTCURSORS,
    DV_E_FORMATETC, E_NOTIMPL, HANDLE, HWND, OLE_E_ADVISENOTSUPPORTED, POINT, RECT, SIZE, S_OK,
};
use windows::Win32::Graphics::Gdi::{
    CreateDIBSection, CreateFontW, CreateRoundRectRgn, CreateSolidBrush, DeleteDC, DeleteObject,
    DrawTextW, FillRgn, GetDC, GetSysColor, ReleaseDC, SelectClipRgn, SelectObject, SetBkMode,
    SetTextColor, BITMAPINFO, BITMAPINFOHEADER, BI_RGB, CLEARTYPE_QUALITY, CLIP_DEFAULT_PRECIS,
    COLOR_WINDOW, COLOR_WINDOWTEXT, DEFAULT_CHARSET, DIB_RGB_COLORS, DT_END_ELLIPSIS, DT_WORDBREAK,
    FF_SWISS, FW_NORMAL, HBITMAP, HGDIOBJ, OUT_DEFAULT_PRECIS, TRANSPARENT, VARIABLE_PITCH,
};
use windows::Win32::System::Com::{
    CoCreateInstance, IAdviseSink, IDataObject, IDataObject_Impl, IEnumFORMATETC, IEnumSTATDATA,
    CLSCTX_INPROC_SERVER, DVASPECT_CONTENT, FORMATETC, STGMEDIUM, STGMEDIUM_0, TYMED_HGLOBAL,
};
use windows::Win32::System::DataExchange::RegisterClipboardFormatW;
use windows::Win32::System::Memory::{GlobalAlloc, GlobalLock, GlobalUnlock, GMEM_FIXED};
use windows::Win32::System::Ole::{
    DoDragDrop, IDropSource, IDropSource_Impl, OleInitialize, CF_UNICODETEXT, DROPEFFECT,
    DROPEFFECT_COPY,
};
use windows::Win32::System::SystemServices::{MK_LBUTTON, MODIFIERKEYS_FLAGS};
use windows::Win32::UI::Shell::{CLSID_DragDropHelper, IDragSourceHelper, SHDRAGIMAGE};

use crate::core::{AppError, Result};

static OLE_INIT: Once = Once::new();
static CF_HTML: OnceLock<u16> = OnceLock::new();
static CF_RTF: OnceLock<u16> = OnceLock::new();

fn ensure_ole_init() {
    OLE_INIT.call_once(|| unsafe {
        // 与 drag-rs 一致：传 Some(null_mut)。windows v0.52 的 OleInitialize 签名为
        // Option<*mut c_void>；进程级一次性初始化，全程不 Uninitialize。
        let _ = OleInitialize(Some(std::ptr::null_mut()));
    });
}

/// 注册（或返回已缓存的）剪贴板自定义格式 id。同名格式在进程间共享 id。
fn register_format(name: &str, slot: &'static OnceLock<u16>) -> u16 {
    *slot.get_or_init(|| {
        let wide: Vec<u16> = std::ffi::OsStr::new(name)
            .encode_wide()
            .chain(once(0))
            .collect();
        let id = unsafe { RegisterClipboardFormatW(PCWSTR(wide.as_ptr())) };
        // 返回 0 表示注册失败；正常情况下不会发生。
        id as u16
    })
}

fn cf_html() -> u16 {
    register_format("HTML Format", &CF_HTML)
}

fn cf_rtf() -> u16 {
    register_format("Rich Text Format", &CF_RTF)
}

/// 按 Microsoft CF_HTML spec 构造 payload（UTF-8 字节流，含 header + 片段标记）。
///
/// header 字段是 ASCII 十进制偏移、固定 10 位宽，先写占位再回填实际偏移。
/// 包裹 `<!--StartFragment-->` / `<!--EndFragment-->` 注释，让接收方知道粘贴范围。
fn build_cf_html_payload(html: &str) -> Vec<u8> {
    const HEADER_TMPL: &str = "Version:0.9\r\n\
        StartHTML:0000000000\r\n\
        EndHTML:0000000000\r\n\
        StartFragment:0000000000\r\n\
        EndFragment:0000000000\r\n";
    const HTML_PREFIX: &str = "<html><body>\r\n<!--StartFragment-->";
    const HTML_SUFFIX: &str = "<!--EndFragment-->\r\n</body></html>";

    let mut buf = String::with_capacity(
        HEADER_TMPL.len() + HTML_PREFIX.len() + html.len() + HTML_SUFFIX.len(),
    );
    buf.push_str(HEADER_TMPL);
    let start_html = buf.len();
    buf.push_str(HTML_PREFIX);
    let start_fragment = buf.len();
    buf.push_str(html);
    let end_fragment = buf.len();
    buf.push_str(HTML_SUFFIX);
    let end_html = buf.len();

    let patch = |buf: &mut String, label: &str, value: usize| {
        let needle = format!("{label}:0000000000");
        let replacement = format!("{label}:{value:010}");
        if let Some(pos) = buf.find(&needle) {
            buf.replace_range(pos..pos + needle.len(), &replacement);
        }
    };
    patch(&mut buf, "StartHTML", start_html);
    patch(&mut buf, "EndHTML", end_html);
    patch(&mut buf, "StartFragment", start_fragment);
    patch(&mut buf, "EndFragment", end_fragment);

    buf.into_bytes()
}

/// 把字节缓冲拷到新的 HGLOBAL，包装成 STGMEDIUM 返回。
fn bytes_to_stgmedium(bytes: &[u8]) -> windows::core::Result<STGMEDIUM> {
    unsafe {
        let handle = GlobalAlloc(GMEM_FIXED, bytes.len())?;
        let ptr = GlobalLock(handle) as *mut u8;
        std::ptr::copy_nonoverlapping(bytes.as_ptr(), ptr, bytes.len());
        let _ = GlobalUnlock(handle);

        Ok(STGMEDIUM {
            tymed: TYMED_HGLOBAL.0 as u32,
            u: STGMEDIUM_0 { hGlobal: handle },
            pUnkForRelease: std::mem::ManuallyDrop::new(None),
        })
    }
}

#[implement(IDataObject)]
struct RichDataObject {
    /// UTF-16 with trailing NUL，CF_UNICODETEXT 用。
    text_utf16: Vec<u16>,
    /// 已带 Microsoft CF_HTML header 的 UTF-8 字节流，None 表示不提供 HTML。
    html_bytes: Option<Vec<u8>>,
    /// 原样的 RTF 字节流（ASCII），None 表示不提供 RTF。
    rtf_bytes: Option<Vec<u8>>,
}

impl RichDataObject {
    fn new(plain: &str, html: Option<&str>, rtf: Option<&str>) -> Self {
        let text_utf16: Vec<u16> = std::ffi::OsStr::new(plain)
            .encode_wide()
            .chain(once(0))
            .collect();
        Self {
            text_utf16,
            html_bytes: html.map(build_cf_html_payload),
            rtf_bytes: rtf.map(|s| s.as_bytes().to_vec()),
        }
    }

    /// 判定 FORMATETC 是否落在我们支持的某个格式上。
    fn supported_format(&self, format: *const FORMATETC) -> Option<u16> {
        let fmt = unsafe { format.as_ref()? };
        if fmt.tymed as i32 != TYMED_HGLOBAL.0 || fmt.dwAspect != DVASPECT_CONTENT.0 {
            return None;
        }
        if fmt.cfFormat == CF_UNICODETEXT.0 {
            return Some(CF_UNICODETEXT.0);
        }
        if self.html_bytes.is_some() && fmt.cfFormat == cf_html() {
            return Some(fmt.cfFormat);
        }
        if self.rtf_bytes.is_some() && fmt.cfFormat == cf_rtf() {
            return Some(fmt.cfFormat);
        }
        None
    }

    fn alloc_for(&self, cf: u16) -> windows::core::Result<STGMEDIUM> {
        if cf == CF_UNICODETEXT.0 {
            let bytes = unsafe {
                std::slice::from_raw_parts(
                    self.text_utf16.as_ptr() as *const u8,
                    self.text_utf16.len() * std::mem::size_of::<u16>(),
                )
            };
            return bytes_to_stgmedium(bytes);
        }
        if cf == cf_html() {
            if let Some(b) = &self.html_bytes {
                return bytes_to_stgmedium(b);
            }
        }
        if cf == cf_rtf() {
            if let Some(b) = &self.rtf_bytes {
                return bytes_to_stgmedium(b);
            }
        }
        Err(WinError::new(DV_E_FORMATETC, HSTRING::new()))
    }
}

#[allow(non_snake_case)]
impl IDataObject_Impl for RichDataObject {
    fn GetData(&self, pformatetc: *const FORMATETC) -> windows::core::Result<STGMEDIUM> {
        match self.supported_format(pformatetc) {
            Some(cf) => self.alloc_for(cf),
            None => Err(WinError::new(DV_E_FORMATETC, HSTRING::new())),
        }
    }

    fn GetDataHere(
        &self,
        _pformatetc: *const FORMATETC,
        _pmedium: *mut STGMEDIUM,
    ) -> windows::core::Result<()> {
        Err(WinError::new(E_NOTIMPL, HSTRING::new()))
    }

    fn QueryGetData(&self, pformatetc: *const FORMATETC) -> HRESULT {
        if self.supported_format(pformatetc).is_some() {
            S_OK
        } else {
            DV_E_FORMATETC
        }
    }

    fn GetCanonicalFormatEtc(
        &self,
        _pformatectin: *const FORMATETC,
        pformatetcout: *mut FORMATETC,
    ) -> HRESULT {
        unsafe { (*pformatetcout).ptd = std::ptr::null_mut() };
        E_NOTIMPL
    }

    fn SetData(
        &self,
        _pformatetc: *const FORMATETC,
        _pmedium: *const STGMEDIUM,
        _frelease: BOOL,
    ) -> windows::core::Result<()> {
        Err(WinError::new(E_NOTIMPL, HSTRING::new()))
    }

    fn EnumFormatEtc(&self, _dwdirection: u32) -> windows::core::Result<IEnumFORMATETC> {
        // 不实现枚举：多数接收方先 QueryGetData 探测，不依赖 enumerator。
        Err(WinError::new(E_NOTIMPL, HSTRING::new()))
    }

    fn DAdvise(
        &self,
        _pformatetc: *const FORMATETC,
        _advf: u32,
        _padvsink: Option<&IAdviseSink>,
    ) -> windows::core::Result<u32> {
        Err(WinError::new(OLE_E_ADVISENOTSUPPORTED, HSTRING::new()))
    }

    fn DUnadvise(&self, _dwconnection: u32) -> windows::core::Result<()> {
        Err(WinError::new(OLE_E_ADVISENOTSUPPORTED, HSTRING::new()))
    }

    fn EnumDAdvise(&self) -> windows::core::Result<IEnumSTATDATA> {
        Err(WinError::new(OLE_E_ADVISENOTSUPPORTED, HSTRING::new()))
    }
}

#[implement(IDropSource)]
struct DropSource;

#[allow(non_snake_case)]
impl IDropSource_Impl for DropSource {
    fn QueryContinueDrag(&self, fescapepressed: BOOL, grfkeystate: MODIFIERKEYS_FLAGS) -> HRESULT {
        if fescapepressed.as_bool() {
            DRAGDROP_S_CANCEL
        } else if (grfkeystate & MK_LBUTTON) == MODIFIERKEYS_FLAGS(0) {
            DRAGDROP_S_DROP
        } else {
            S_OK
        }
    }

    fn GiveFeedback(&self, _dweffect: DROPEFFECT) -> HRESULT {
        DRAGDROP_S_USEDEFAULTCURSORS
    }
}

/// 启动一次文本 drag-out（plain + 可选 html / rtf）。阻塞至 drop 完成；
/// 调用方必须在 Tauri 主线程上跑。
///
/// 没有传入 `_preview_png` 时（当前 commands 层就是这样），用 GDI 现场把文本前几行
/// 渲染成圆角 DIB，通过 `IDragSourceHelper::InitializeFromBitmap` 关联到 data object，
/// 跟随光标显示——和 macOS 的视觉对齐。
pub fn start_drag_text(
    window: &WebviewWindow,
    plain: &str,
    html: Option<&str>,
    rtf: Option<&str>,
    _preview_png: Option<Vec<u8>>,
) -> Result<()> {
    if plain.is_empty() {
        return Err(AppError::Clipboard("drag-out: empty text".to_string()));
    }

    ensure_ole_init();

    let data_object: IDataObject = RichDataObject::new(plain, html, rtf).into();
    let drop_source: IDropSource = DropSource.into();

    // 关联预览图：用 GDI 把文本渲染成 DIB，再走 IDragSourceHelper。
    // 失败不致命（外部 app 拖入时只是没有 ghost），打 warn 继续 DoDragDrop。
    let scale = window.scale_factor().unwrap_or(1.0);
    let size_px = (128.0 * scale).round() as i32;
    if let Err(err) = attach_text_preview(&data_object, plain, size_px) {
        log::warn!("attach drag preview failed: {err}");
    }

    let mut out = DROPEFFECT::default();
    let hr = unsafe { DoDragDrop(&data_object, &drop_source, DROPEFFECT_COPY, &mut out) };

    if hr == DRAGDROP_S_DROP {
        log::debug!("drag-out text finished: Dropped");
    } else {
        log::debug!("drag-out text finished: {hr:?}");
    }

    Ok(())
}

/// 把 GDI 渲染的文本 bitmap 通过 `IDragSourceHelper` 关联到 `data_object`，
/// 由 Shell 接管 HBITMAP（成功后我们不再 `DeleteObject`，失败由本函数清理）。
fn attach_text_preview(data_object: &IDataObject, text: &str, size_px: i32) -> Result<()> {
    let hbmp = render_text_preview_bitmap(text, size_px)
        .ok_or_else(|| AppError::Clipboard("render preview bitmap failed".to_string()))?;

    unsafe {
        let helper_res: windows::core::Result<IDragSourceHelper> =
            CoCreateInstance(&CLSID_DragDropHelper, None, CLSCTX_INPROC_SERVER);
        let helper = match helper_res {
            Ok(h) => h,
            Err(err) => {
                let _ = DeleteObject(HGDIOBJ(hbmp.0));
                return Err(AppError::Clipboard(format!(
                    "create DragDropHelper failed: {err}"
                )));
            }
        };

        let img = SHDRAGIMAGE {
            sizeDragImage: SIZE {
                cx: size_px,
                cy: size_px,
            },
            // 光标落在 bitmap 中心，与 macOS 一致。
            ptOffset: POINT {
                x: size_px / 2,
                y: size_px / 2,
            },
            hbmpDragImage: hbmp,
            // 0 == 不使用 color key 透明（我们的 bitmap 整张不透明）。
            crColorKey: COLORREF(0),
        };

        if let Err(err) = helper.InitializeFromBitmap(&img, data_object) {
            let _ = DeleteObject(HGDIOBJ(hbmp.0));
            return Err(AppError::Clipboard(format!(
                "InitializeFromBitmap failed: {err}"
            )));
        }
    }

    Ok(())
}

/// 用 GDI 把 `text` 前几行渲染成 size_px × size_px 的 32-bit top-down DIB，圆角白卡风格。
///
/// 颜色用 `GetSysColor(COLOR_WINDOW / COLOR_WINDOWTEXT)`，跟随系统主题（浅色/高对比度）。
/// Windows 10/11 暗色模式下 `GetSysColor` 不会自动反转——这是 Win32 设计限制，
/// 后续要更精确的暗色支持需要查 `AppsUseLightTheme` 注册表或用 UWP `UISettings`。
fn render_text_preview_bitmap(text: &str, size_px: i32) -> Option<HBITMAP> {
    unsafe {
        let hdc_screen = GetDC(HWND(0));
        if hdc_screen.is_invalid() {
            return None;
        }
        let hdc_mem = windows::Win32::Graphics::Gdi::CreateCompatibleDC(hdc_screen);
        if hdc_mem.is_invalid() {
            ReleaseDC(HWND(0), hdc_screen);
            return None;
        }

        let mut bmi: BITMAPINFO = std::mem::zeroed();
        bmi.bmiHeader.biSize = std::mem::size_of::<BITMAPINFOHEADER>() as u32;
        bmi.bmiHeader.biWidth = size_px;
        // 负高度 = top-down DIB（左上为原点），DrawText 默认坐标也是 top-down。
        bmi.bmiHeader.biHeight = -size_px;
        bmi.bmiHeader.biPlanes = 1;
        bmi.bmiHeader.biBitCount = 32;
        bmi.bmiHeader.biCompression = BI_RGB.0;

        let mut bits: *mut std::ffi::c_void = std::ptr::null_mut();
        let hbmp = match CreateDIBSection(
            hdc_screen,
            &bmi,
            DIB_RGB_COLORS,
            &mut bits,
            HANDLE::default(),
            0,
        ) {
            Ok(h) if !h.is_invalid() => h,
            _ => {
                DeleteDC(hdc_mem);
                ReleaseDC(HWND(0), hdc_screen);
                return None;
            }
        };

        let old_bmp = SelectObject(hdc_mem, HGDIOBJ(hbmp.0));

        // 圆角白底：CreateRoundRectRgn 的右/下是开区间，+1 才覆盖整张图。
        let radius = (size_px / 6).max(8);
        let rgn = CreateRoundRectRgn(0, 0, size_px + 1, size_px + 1, radius, radius);
        let bg_color = COLORREF(GetSysColor(COLOR_WINDOW));
        let bg_brush = CreateSolidBrush(bg_color);
        FillRgn(hdc_mem, rgn, bg_brush);
        DeleteObject(HGDIOBJ(bg_brush.0));
        SelectClipRgn(hdc_mem, rgn);

        // 字体：高度按预览尺寸的 1/12 估算，约 7 行可见。
        let font_height = (size_px / 12).max(12);
        let font = CreateFontW(
            font_height,
            0,
            0,
            0,
            FW_NORMAL.0 as i32,
            0,
            0,
            0,
            DEFAULT_CHARSET.0 as u32,
            OUT_DEFAULT_PRECIS.0 as u32,
            CLIP_DEFAULT_PRECIS.0 as u32,
            CLEARTYPE_QUALITY.0 as u32,
            (FF_SWISS.0 | VARIABLE_PITCH.0) as u32,
            PCWSTR::null(),
        );
        let old_font = SelectObject(hdc_mem, HGDIOBJ(font.0));

        SetTextColor(hdc_mem, COLORREF(GetSysColor(COLOR_WINDOWTEXT)));
        SetBkMode(hdc_mem, TRANSPARENT);

        let padding = (size_px / 8).max(8);
        let mut text_rect = RECT {
            left: padding,
            top: padding,
            right: size_px - padding,
            bottom: size_px - padding,
        };
        let snippet = clamp_text(text, 280);
        let mut wide: Vec<u16> = snippet.encode_utf16().collect();
        DrawTextW(
            hdc_mem,
            &mut wide[..],
            &mut text_rect,
            DT_WORDBREAK | DT_END_ELLIPSIS,
        );

        // 还原 DC 状态，释放临时 GDI 对象。HBITMAP 留给调用方（IDragSourceHelper 接管）。
        SelectClipRgn(hdc_mem, windows::Win32::Graphics::Gdi::HRGN::default());
        DeleteObject(HGDIOBJ(rgn.0));
        SelectObject(hdc_mem, old_font);
        DeleteObject(HGDIOBJ(font.0));
        SelectObject(hdc_mem, old_bmp);
        DeleteDC(hdc_mem);
        ReleaseDC(HWND(0), hdc_screen);

        Some(hbmp)
    }
}

/// 截断 + 折行规整：超长的添省略号；连续空行压成单个换行。
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

/// 启动一次文件 drag-out。直接转发到 `drag` crate（其 `CF_HDROP` 实现稳定），
/// 仅做参数适配与错误包装。
pub fn start_drag_files(
    window: &WebviewWindow,
    paths: Vec<PathBuf>,
    preview_png: Option<Vec<u8>>,
) -> Result<()> {
    use drag::{DragItem, Image, Options};

    let image = match preview_png {
        Some(bytes) => Image::Raw(bytes),
        None => Image::File(paths[0].clone()),
    };

    drag::start_drag(
        window,
        DragItem::Files(paths),
        image,
        |result, _cursor| {
            log::debug!("drag-out files finished: {result:?}");
        },
        Options::default(),
    )
    .map_err(|err| AppError::Clipboard(format!("drag-out failed: {err}")))?;

    Ok(())
}

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
    BOOL, DRAGDROP_S_CANCEL, DRAGDROP_S_DROP, DRAGDROP_S_USEDEFAULTCURSORS, DV_E_FORMATETC,
    E_NOTIMPL, OLE_E_ADVISENOTSUPPORTED, S_OK,
};
use windows::Win32::System::Com::{
    IAdviseSink, IDataObject, IDataObject_Impl, IEnumFORMATETC, IEnumSTATDATA, DVASPECT_CONTENT,
    FORMATETC, STGMEDIUM, STGMEDIUM_0, TYMED_HGLOBAL,
};
use windows::Win32::System::DataExchange::RegisterClipboardFormatW;
use windows::Win32::System::Memory::{GlobalAlloc, GlobalLock, GlobalUnlock, GMEM_FIXED};
use windows::Win32::System::Ole::{
    DoDragDrop, IDropSource, IDropSource_Impl, OleInitialize, CF_UNICODETEXT, DROPEFFECT,
    DROPEFFECT_COPY,
};
use windows::Win32::System::SystemServices::{MK_LBUTTON, MODIFIERKEYS_FLAGS};

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
/// `_preview_png` 暂不接入 `IDragSourceHelper`（DragSourceHelper 在 WebView2 子窗口的
/// 拖拽 ghost 显示存在已知问题，详见上层注释；接好接收端再决定是否补预览图）。
pub fn start_drag_text(
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

    let mut out = DROPEFFECT::default();
    let hr = unsafe { DoDragDrop(&data_object, &drop_source, DROPEFFECT_COPY, &mut out) };

    if hr == DRAGDROP_S_DROP {
        log::debug!("drag-out text finished: Dropped");
    } else {
        log::debug!("drag-out text finished: {hr:?}");
    }

    Ok(())
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

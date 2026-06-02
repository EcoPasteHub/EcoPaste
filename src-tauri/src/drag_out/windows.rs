//! Windows drag-out 实现。
//!
//! - **文件**（`start_drag_files`）：直接复用 `drag` crate v2.1.1
//!   （`IDataObject(CF_HDROP)` + `DoDragDrop`），稳定且自带 `IDragSourceHelper` 预览。
//! - **纯文本**（`start_drag_text`）：自实现 `IDataObject(CF_UNICODETEXT)` + `IDropSource`，
//!   因为 `drag` crate 的 `DragItem::Data` 是 dummy（不管传什么，实际拖出的永远是 `./` 目录）。
//!
//! 通用约束：所有入口**必须在拥有窗口的线程上调用**（= Tauri 主线程），否则
//! `IDropSource::QueryContinueDrag` 会立刻返回 `DRAGDROP_S_CANCEL`，拖拽秒取消。
//!
//! `OleInitialize` 用进程级 `Once` 兜底，全程不 Uninitialize。

use std::iter::once;
use std::os::windows::ffi::OsStrExt;
use std::path::PathBuf;
use std::sync::Once;

use tauri::WebviewWindow;
use windows::core::{implement, Error as WinError, HRESULT, HSTRING};
use windows::Win32::Foundation::{BOOL, DV_E_FORMATETC, E_NOTIMPL, OLE_E_ADVISENOTSUPPORTED, S_OK};
use windows::Win32::System::Com::{
    IAdviseSink, IDataObject, IDataObject_Impl, IEnumFORMATETC, IEnumSTATDATA, DVASPECT_CONTENT,
    FORMATETC, STGMEDIUM, STGMEDIUM_0, TYMED_HGLOBAL,
};
use windows::Win32::System::DataExchange::CF_UNICODETEXT;
use windows::Win32::System::Memory::{GlobalAlloc, GlobalLock, GlobalUnlock, GMEM_FIXED};
use windows::Win32::System::Ole::{
    DoDragDrop, IDropSource, IDropSource_Impl, OleInitialize, DRAGDROP_S_CANCEL, DRAGDROP_S_DROP,
    DROPEFFECT, DROPEFFECT_COPY,
};
use windows::Win32::System::SystemServices::{MK_LBUTTON, MODIFIERKEYS_FLAGS};

use crate::core::{AppError, Result};

static OLE_INIT: Once = Once::new();

fn ensure_ole_init() {
    OLE_INIT.call_once(|| unsafe {
        // 与 drag-rs 一致：传 Some(null_mut)。windows v0.52 的 OleInitialize 签名为
        // Option<*mut c_void>；进程级一次性初始化，全程不 Uninitialize。
        let _ = OleInitialize(Some(std::ptr::null_mut()));
    });
}

#[implement(IDataObject)]
struct TextDataObject {
    /// UTF-16 with trailing NUL，写入 HGLOBAL 时一并复制。
    text_utf16: Vec<u16>,
}

impl TextDataObject {
    fn new(text: &str) -> Self {
        let text_utf16: Vec<u16> = std::ffi::OsStr::new(text)
            .encode_wide()
            .chain(once(0))
            .collect();
        Self { text_utf16 }
    }

    fn is_supported(format: *const FORMATETC) -> bool {
        let Some(fmt) = (unsafe { format.as_ref() }) else {
            return false;
        };
        fmt.cfFormat == CF_UNICODETEXT.0
            && fmt.tymed as i32 == TYMED_HGLOBAL.0
            && fmt.dwAspect == DVASPECT_CONTENT.0
    }

    /// 复制内部 UTF-16 到新的 HGLOBAL；调用方负责后续释放（STGMEDIUM 由接收方 ReleaseStgMedium）。
    fn alloc_hglobal(&self) -> windows::core::Result<STGMEDIUM> {
        let bytes = self.text_utf16.len() * std::mem::size_of::<u16>();
        unsafe {
            let handle = GlobalAlloc(GMEM_FIXED, bytes)?;
            let ptr = GlobalLock(handle) as *mut u16;
            std::ptr::copy_nonoverlapping(self.text_utf16.as_ptr(), ptr, self.text_utf16.len());
            let _ = GlobalUnlock(handle);

            Ok(STGMEDIUM {
                tymed: TYMED_HGLOBAL.0 as u32,
                u: STGMEDIUM_0 { hGlobal: handle },
                pUnkForRelease: std::mem::ManuallyDrop::new(None),
            })
        }
    }
}

#[allow(non_snake_case)]
impl IDataObject_Impl for TextDataObject {
    fn GetData(&self, pformatetc: *const FORMATETC) -> windows::core::Result<STGMEDIUM> {
        if Self::is_supported(pformatetc) {
            self.alloc_hglobal()
        } else {
            Err(WinError::new(DV_E_FORMATETC, HSTRING::new()))
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
        if Self::is_supported(pformatetc) {
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
        // 不实现枚举：多数接收方先 QueryGetData(CF_UNICODETEXT) 探测，不依赖 enumerator。
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
        windows::Win32::System::Ole::DRAGDROP_S_USEDEFAULTCURSORS
    }
}

/// 启动一次纯文本 drag-out。阻塞至 drop 完成；调用方必须在 Tauri 主线程上跑。
///
/// `_preview_png` 暂不接入 `IDragSourceHelper`（DragSourceHelper 在 WebView2 子窗口的
/// 拖拽 ghost 显示存在已知问题，详见上层注释；接好接收端再决定是否补预览图）。
pub fn start_drag_text(text: &str, _preview_png: Option<Vec<u8>>) -> Result<()> {
    if text.is_empty() {
        return Err(AppError::Clipboard("drag-out: empty text".to_string()));
    }

    ensure_ole_init();

    let data_object: IDataObject = TextDataObject::new(text).into();
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

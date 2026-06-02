//! 让 OLE 拖拽在「我们自己的窗口里」也能显示半透明 ghost 预览图。
//!
//! Win32 OLE 拖拽的 ghost 由**接收方**（光标当前所在窗口注册的 `IDropTarget`）
//! 在每个 `DragEnter` / `DragOver` 回调里转发给 `IDropTargetHelper` 才会绘制。
//! WebView2 自己的 `IDropTarget`（注册在 webview 子窗 HWND 上）**不转发**，
//! 导致从我们窗口拖出去的内容在窗口内没有预览图——必须移到 Explorer / Office 等
//! "正常"窗口才出现 ghost。
//!
//! 解法：把窗口树里所有注册过 `IDropTarget` 的 HWND 上的目标包一层
//! `ForwardingDropTarget`：每个回调先调 `IDropTargetHelper` 再转发回原目标，
//! 既补上 ghost、又不破坏 WebView2 自己的 HTML5 drop 行为。
//!
//! Win32 把原 `IDropTarget` 存在 HWND 的 `OleDropTargetInterface` window prop 上
//! （`RegisterDragDrop` 文档化的内部约定），用 `GetPropW` 即可取回包装。
//!
//! 必须在拥有窗口的线程（= Tauri 主线程）上调用——`RegisterDragDrop` /
//! `RevokeDragDrop` 都有线程亲和性。

use std::ffi::c_void;
use std::iter::once;
use std::sync::{Mutex, OnceLock};

use tauri::WebviewWindow;
use windows::core::{implement, ComInterface, IUnknown, Interface, PCWSTR};
use windows::Win32::Foundation::{BOOL, HWND, LPARAM, POINT, POINTL};
use windows::Win32::System::Com::{CoCreateInstance, IDataObject, CLSCTX_INPROC_SERVER};
use windows::Win32::System::Ole::{
    IDropTarget, IDropTarget_Impl, RegisterDragDrop, RevokeDragDrop, DROPEFFECT,
};
use windows::Win32::System::SystemServices::MODIFIERKEYS_FLAGS;
use windows::Win32::UI::Shell::{CLSID_DragDropHelper, IDropTargetHelper};
use windows::Win32::UI::WindowsAndMessaging::{EnumChildWindows, GetPropW};

/// Win32 OLE 用这个 prop 名把原始 `IDropTarget` 挂在 HWND 上。
/// 文档化于多个 MS 内部头文件 / Shell sample，世面上的 drop target wrapper 都依赖这个约定。
const PROP_DROP_TARGET: &str = "OleDropTargetInterface";

#[implement(IDropTarget)]
struct ForwardingDropTarget {
    inner: IDropTarget,
    helper: IDropTargetHelper,
    hwnd: HWND,
}

#[allow(non_snake_case)]
impl IDropTarget_Impl for ForwardingDropTarget {
    fn DragEnter(
        &self,
        pdataobj: Option<&IDataObject>,
        grfkeystate: MODIFIERKEYS_FLAGS,
        pt: &POINTL,
        pdweffect: *mut DROPEFFECT,
    ) -> windows::core::Result<()> {
        unsafe {
            let p = POINT { x: pt.x, y: pt.y };
            let effect = *pdweffect;
            let _ = self.helper.DragEnter(self.hwnd, pdataobj, &p, effect);

            self.inner.DragEnter(pdataobj, grfkeystate, *pt, pdweffect)
        }
    }

    fn DragOver(
        &self,
        grfkeystate: MODIFIERKEYS_FLAGS,
        pt: &POINTL,
        pdweffect: *mut DROPEFFECT,
    ) -> windows::core::Result<()> {
        unsafe {
            let p = POINT { x: pt.x, y: pt.y };
            let effect = *pdweffect;
            let _ = self.helper.DragOver(&p, effect);

            self.inner.DragOver(grfkeystate, *pt, pdweffect)
        }
    }

    fn DragLeave(&self) -> windows::core::Result<()> {
        unsafe {
            let _ = self.helper.DragLeave();

            self.inner.DragLeave()
        }
    }

    fn Drop(
        &self,
        pdataobj: Option<&IDataObject>,
        grfkeystate: MODIFIERKEYS_FLAGS,
        pt: &POINTL,
        pdweffect: *mut DROPEFFECT,
    ) -> windows::core::Result<()> {
        unsafe {
            let p = POINT { x: pt.x, y: pt.y };
            let effect = *pdweffect;
            let _ = self.helper.Drop(pdataobj, &p, effect);

            self.inner.Drop(pdataobj, grfkeystate, *pt, pdweffect)
        }
    }
}

/// 进程级一次性安装：包装主窗口及其所有子窗（含 WebView2 内部）注册的 IDropTarget。
///
/// 用 OnceLock 兜底——WebView2 的子窗注册是异步的，所以本函数应该在窗口已显示、
/// 用户首次触发拖拽时调用，那时 WebView2 已 ready；万一更早调用没有捕到，
/// 后续靠 [`WRAPPED_TARGETS`] 也能识别重复并补装。
static GHOST_INSTALL_RESULT: Mutex<bool> = Mutex::new(false);

/// 记录我们已经包装过的「原始」IDropTarget raw 指针（注意：不是我们的 wrapper 指针）。
/// 用于幂等：第二次进入 [`install_for_window`] 时若发现某 HWND 当前 prop 指向
/// 我们包装过的某个 inner，就跳过——避免对自己再包一层。
static WRAPPED_INNERS: OnceLock<Mutex<Vec<usize>>> = OnceLock::new();

fn wrapped_inners() -> &'static Mutex<Vec<usize>> {
    WRAPPED_INNERS.get_or_init(|| Mutex::new(Vec::new()))
}

/// 给指定 Tauri 窗口的整棵 HWND 树安装 ghost drop target 包装。
/// 幂等、可重复调用；线程要求：拥有窗口的线程（Tauri 主线程）。
pub fn install_for_window(window: &WebviewWindow) {
    let Ok(raw_hwnd) = window.hwnd() else {
        return;
    };
    // Tauri 暴露的 HWND 来自更新版 `windows` crate（内部是 *mut c_void），
    // 我们这里依赖的是 windows 0.52（HWND 内部是 isize）——按裸指针的整数值平转换即可。
    let hwnd = HWND(raw_hwnd.0 as isize);

    unsafe {
        try_wrap_hwnd(hwnd);
        // EnumChildWindows 默认会遍历整棵后代树（不仅是直接子窗）。
        let _ = EnumChildWindows(hwnd, Some(enum_child_proc), LPARAM(0));
    }

    let mut installed = GHOST_INSTALL_RESULT.lock().unwrap();
    if !*installed {
        *installed = true;
        log::debug!("drag ghost: installed on main window tree");
    }
}

extern "system" fn enum_child_proc(hwnd: HWND, _lparam: LPARAM) -> BOOL {
    unsafe {
        try_wrap_hwnd(hwnd);
    }
    BOOL(1)
}

/// 取 HWND 上现存的 IDropTarget，包一层 `ForwardingDropTarget` 重新注册。
/// 若该 HWND 没注册过 drop target，或 inner 已经是我们的（避免自嵌套），直接跳过。
unsafe fn try_wrap_hwnd(hwnd: HWND) {
    let prop_w: Vec<u16> = PROP_DROP_TARGET.encode_utf16().chain(once(0)).collect();
    let handle = GetPropW(hwnd, PCWSTR(prop_w.as_ptr()));
    if handle.is_invalid() || handle.0 == 0 {
        return;
    }

    let raw = handle.0 as *mut c_void;
    if raw.is_null() {
        return;
    }

    // 已经包装过的 inner 不要再二次包装（每个 inner 只允许出现一次）。
    {
        let inners = wrapped_inners().lock().unwrap();
        if inners.contains(&(raw as usize)) {
            return;
        }
    }

    // GetProp 给我们的是借用指针，没 AddRef；用 from_raw_borrowed 借出 IUnknown，
    // 再 cast 到 IDropTarget——cast 内部走 QueryInterface 会 AddRef，得到一份强引用。
    let raw_ref = &raw;
    let Some(unk) = IUnknown::from_raw_borrowed(raw_ref) else {
        return;
    };
    let Ok(inner) = unk.cast::<IDropTarget>() else {
        return;
    };

    let helper: IDropTargetHelper =
        match CoCreateInstance(&CLSID_DragDropHelper, None, CLSCTX_INPROC_SERVER) {
            Ok(h) => h,
            Err(err) => {
                log::warn!("drag ghost: CoCreateInstance(IDropTargetHelper) failed: {err}");
                return;
            }
        };

    let wrapper: IDropTarget = ForwardingDropTarget {
        inner,
        helper,
        hwnd,
    }
    .into();

    // 先 Revoke 才能 Register 新目标；Revoke 会 Release 原 inner 一次，但我们前面
    // 通过 cast 已经多 AddRef 了一份，余额仍为 +1，inner 不会被释放。
    let _ = RevokeDragDrop(hwnd);
    if let Err(err) = RegisterDragDrop(hwnd, &wrapper) {
        log::warn!(
            "drag ghost: RegisterDragDrop on hwnd {:?} failed: {err}",
            hwnd.0
        );
        return;
    }

    wrapped_inners().lock().unwrap().push(raw as usize);
    log::debug!("drag ghost: wrapped IDropTarget on hwnd {:?}", hwnd.0);
}

//! Windows UI Automation 辅助:记住剪贴板窗口抢走前台之前目标应用的聚焦元素,
//! 回切时用元素级 SetFocus 复活 Chromium 等 web 视图丢失的内部焦点。
//!
//! 光标类应用(如 Tauri/Electron/WebView2 聊天窗口)失焦后会丢掉 web 内部焦点,
//! 仅 `SetForegroundWindow` 恢复顶层前台救不回输入光标(实测);对抢焦点前记住的
//! 聚焦元素执行 `IUIAutomationElement::SetFocus` 可以原位唤醒光标,随后的模拟
//! Ctrl+V 才能落进目标输入框。

#![cfg(target_os = "windows")]

use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{Mutex, Once};

use windows::core::{ComInterface, IUnknown};
use windows::Win32::System::Com::{
    CoCreateInstance, CoInitializeEx, CLSCTX_INPROC_SERVER, COINIT_MULTITHREADED,
};
use windows::Win32::UI::Accessibility::{CUIAutomation, IUIAutomation, IUIAutomationElement};

static COM_INIT: Once = Once::new();
static COM_OK: AtomicBool = AtomicBool::new(false);

/// UIA 代理对象按 MTA 代理跨线程使用是可用的,windows-rs 接口本质是裸 COM 指针,
/// 这里手工声明 Send 以便在 tauri 命令线程间传递。
struct SendElement(IUIAutomationElement);
unsafe impl Send for SendElement {}

static REMEMBERED: Mutex<Option<SendElement>> = Mutex::new(None);

fn automation() -> Option<IUIAutomation> {
    COM_INIT.call_once(|| {
        // MTA 无套间亲和,可安全在任意 tauri 命令线程调用;失败(如他处已按 STA 初始化)
        // 时降级为整个模块不可用,调用点均静默回退。
        let hr = unsafe { CoInitializeEx(None, COINIT_MULTITHREADED) };
        if hr.is_ok() {
            COM_OK.store(true, Ordering::Release);
        } else {
            log::warn!("UIA CoInitializeEx failed: {hr:?}");
        }
    });

    if !COM_OK.load(Ordering::Acquire) {
        return None;
    }

    unsafe {
        let unknown: IUnknown =
            CoCreateInstance(&CUIAutomation, None, CLSCTX_INPROC_SERVER).ok()?;
        unknown.cast::<IUIAutomation>().ok()
    }
}

/// 在剪贴板窗口抢走前台之前调用:记住当前系统的聚焦 UIA 元素。
/// 记忆遵循「先到先得」,与 pre-edit 前台窗口记录保持同一语义。
pub fn remember_focused_element() {
    let Some(automation) = automation() else {
        return;
    };

    let Ok(element) = (unsafe { automation.GetFocusedElement() }) else {
        return;
    };

    let mut guard = REMEMBERED.lock().expect("remembered uia element poisoned");
    if guard.is_some() {
        return;
    }

    *guard = Some(SendElement(element));
}

/// 对记忆的聚焦元素执行元素级 SetFocus,复活目标输入框的光标。
/// 返回是否执行成功;元素不存在或不支持时返回 false,调用方降级为仅恢复前台。
pub fn focus_remembered_element() -> bool {
    let element = {
        let guard = REMEMBERED.lock().expect("remembered uia element poisoned");
        guard.as_ref().map(|SendElement(inner)| inner.clone())
    };

    let Some(element) = element else {
        return false;
    };

    unsafe { element.SetFocus() }.is_ok()
}

/// 取空并释放记忆的元素,防止跨窗口使用过期引用。
pub fn dismiss_remembered_element() {
    REMEMBERED
        .lock()
        .expect("remembered uia element poisoned")
        .take();
}

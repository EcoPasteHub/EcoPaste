use std::collections::HashSet;
use std::ptr::null_mut;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{Mutex, OnceLock};

use serde_json::json;
use tauri::{AppHandle, Emitter};
use winapi::shared::minwindef::{LPARAM, LRESULT, UINT, WPARAM};
use winapi::um::processthreadsapi::GetCurrentThreadId;
use winapi::um::winuser::{
    CallNextHookEx, GetAsyncKeyState, GetMessageW, PostThreadMessageW, SetWindowsHookExW,
    UnhookWindowsHookEx, KBDLLHOOKSTRUCT, MSG, VK_CONTROL, VK_DOWN, VK_ESCAPE, VK_LCONTROL,
    VK_RCONTROL, VK_RETURN, VK_SHIFT, VK_TAB, VK_UP, WH_KEYBOARD_LL, WM_KEYDOWN, WM_KEYUP, WM_QUIT,
    WM_SYSKEYDOWN, WM_SYSKEYUP,
};

use super::NAV_EVENT;

static NAV_ENABLED: AtomicBool = AtomicBool::new(false);
static HOOK_THREAD_ID: Mutex<Option<u32>> = Mutex::new(None);
static APP_HANDLE: OnceLock<AppHandle> = OnceLock::new();

fn consumed_keys() -> &'static Mutex<HashSet<u32>> {
    static SET: OnceLock<Mutex<HashSet<u32>>> = OnceLock::new();
    SET.get_or_init(|| Mutex::new(HashSet::new()))
}

/// 仅放行当前前端需要的 Ctrl 快捷键：F 与数字 1-9。
fn ctrl_shortcut_key(vk: u32) -> Option<String> {
    match vk {
        0x46 => Some("f".to_string()),
        0x31..=0x39 => Some(((vk as u8) as char).to_string()),
        _ => None,
    }
}

pub fn enable_navigation_keys(app: &AppHandle) {
    let _ = APP_HANDLE.set(app.clone());
    NAV_ENABLED.store(true, Ordering::Relaxed);

    // 已有钩子线程时不再起新线程；NAV_ENABLED 的恢复就够了。
    if HOOK_THREAD_ID
        .lock()
        .expect("hook thread id poisoned")
        .is_some()
    {
        return;
    }

    std::thread::spawn(|| unsafe {
        let hook = SetWindowsHookExW(WH_KEYBOARD_LL, Some(hook_proc), null_mut(), 0);
        if hook.is_null() {
            log::error!("SetWindowsHookExW failed");
            return;
        }

        *HOOK_THREAD_ID.lock().expect("hook thread id poisoned") = Some(GetCurrentThreadId());

        let mut msg: MSG = std::mem::zeroed();
        // GetMessageW 收到 WM_QUIT 返回 0 → 消息泵自然退出。
        while GetMessageW(&mut msg, null_mut(), 0, 0) > 0 {}

        UnhookWindowsHookEx(hook);
        *HOOK_THREAD_ID.lock().expect("hook thread id poisoned") = None;
        consumed_keys()
            .lock()
            .expect("consumed keys poisoned")
            .clear();
    });
}

pub fn disable_navigation_keys() {
    NAV_ENABLED.store(false, Ordering::Relaxed);

    let tid = HOOK_THREAD_ID
        .lock()
        .expect("hook thread id poisoned")
        .take();
    if let Some(tid) = tid {
        unsafe {
            PostThreadMessageW(tid, WM_QUIT, 0, 0);
        }
    }
}

unsafe extern "system" fn hook_proc(code: i32, wparam: WPARAM, lparam: LPARAM) -> LRESULT {
    if code < 0 || !NAV_ENABLED.load(Ordering::Relaxed) {
        return CallNextHookEx(null_mut(), code, wparam, lparam);
    }

    let kbd = &*(lparam as *const KBDLLHOOKSTRUCT);
    let vk = kbd.vkCode;
    let msg = wparam as UINT;
    let ctrl_down = (GetAsyncKeyState(VK_CONTROL) as u16) & 0x8000 != 0;

    let is_ctrl = matches!(vk as i32, VK_CONTROL | VK_LCONTROL | VK_RCONTROL);

    if is_ctrl {
        if let Some(app) = APP_HANDLE.get() {
            let action = if msg == WM_KEYDOWN || msg == WM_SYSKEYDOWN {
                Some("ctrlDown")
            } else if msg == WM_KEYUP || msg == WM_SYSKEYUP {
                Some("ctrlUp")
            } else {
                None
            };

            if let Some(action) = action {
                if let Err(err) = app.emit(NAV_EVENT, json!({ "action": action })) {
                    log::warn!("emit nav event failed: {err:?}");
                }
            }
        }

        // Ctrl 状态只用于前端展示与组合键识别，不在此处吞键，避免影响系统行为。
        return CallNextHookEx(null_mut(), code, wparam, lparam);
    }

    let action = match vk as i32 {
        VK_UP => Some("up"),
        VK_DOWN => Some("down"),
        VK_RETURN => Some("enter"),
        VK_ESCAPE => Some("escape"),
        // Tab / Shift+Tab：在前端 ClipboardTabs 里循环切换分组。
        // GetAsyncKeyState 高位为按住状态；shift 同时按下视为反向。
        VK_TAB => Some({
            let shift_down = (GetAsyncKeyState(VK_SHIFT) as u16) & 0x8000 != 0;
            if shift_down {
                "prevTab"
            } else {
                "nextTab"
            }
        }),
        _ => None,
    };
    let shortcut_key = if ctrl_down {
        ctrl_shortcut_key(vk)
    } else {
        None
    };

    if msg == WM_KEYDOWN || msg == WM_SYSKEYDOWN {
        if let Some(shortcut_key) = shortcut_key {
            if let Some(app) = APP_HANDLE.get() {
                if let Err(err) = app.emit(
                    NAV_EVENT,
                    json!({ "action": "shortcut", "key": shortcut_key }),
                ) {
                    log::warn!("emit nav event failed: {err:?}");
                }
            }

            consumed_keys()
                .lock()
                .expect("consumed keys poisoned")
                .insert(vk);
            return 1;
        }

        if let Some(action) = action {
            if let Some(app) = APP_HANDLE.get() {
                if let Err(err) = app.emit(NAV_EVENT, json!({ "action": action })) {
                    log::warn!("emit nav event failed: {err:?}");
                }
            }
            // 记下 KEYDOWN 的 VK，配对的 KEYUP 也要吞——
            // 否则背后被聚焦的应用会收到孤立 KEYUP，造成奇怪行为。
            consumed_keys()
                .lock()
                .expect("consumed keys poisoned")
                .insert(vk);
            return 1;
        }
    } else if msg == WM_KEYUP || msg == WM_SYSKEYUP {
        if consumed_keys()
            .lock()
            .expect("consumed keys poisoned")
            .remove(&vk)
        {
            return 1;
        }
    }

    CallNextHookEx(null_mut(), code, wparam, lparam)
}

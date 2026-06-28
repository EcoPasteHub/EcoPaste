//! Windows 专属：用低级键盘钩子接管 `Win+V`。
//!
//! `Win+V` 是 Windows 系统保留热键，`RegisterHotKey(MOD_WIN, V)`（全局快捷键插件底层）
//! 无法拦截、也无法阻止系统剪贴板历史面板弹出。因此这里装一颗 `WH_KEYBOARD_LL` 钩子，
//! 在 V 按下时若检测到 Win 键按住就吞掉该按键并 toggle 主窗口，使系统面板不再触发。
//!
//! 与 `keyboard/`（主窗可见期间捕获导航键）不同：本钩子的生命周期由设置开关驱动，
//! 开启即常驻，与主窗口可见性无关。

use std::ptr::null_mut;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{Mutex, OnceLock};

use tauri::AppHandle;
use winapi::shared::minwindef::{LPARAM, LRESULT, UINT, WPARAM};
use winapi::um::processthreadsapi::GetCurrentThreadId;
use winapi::um::winuser::{
    CallNextHookEx, GetAsyncKeyState, GetMessageW, PostThreadMessageW, SendInput,
    SetWindowsHookExW, UnhookWindowsHookEx, INPUT, INPUT_KEYBOARD, KBDLLHOOKSTRUCT, KEYBDINPUT,
    KEYEVENTF_KEYUP, LLKHF_INJECTED, MSG, VK_LWIN, VK_RWIN, WH_KEYBOARD_LL, WM_KEYDOWN, WM_KEYUP,
    WM_QUIT, WM_SYSKEYDOWN, WM_SYSKEYUP,
};

use crate::window::{self, MAIN_WINDOW_LABEL};

/// V 键的虚拟键码，winapi 未直接导出。
const VK_V: u32 = 0x56;
/// 注入给系统的占位键：`0xE8` 是未分配 VK（AutoHotkey 的 menu-mask key 同款），
/// 注入它无实际副作用，却能让 shell 认为 Win 按住期间有过其它输入，从而不弹开始菜单。
const VK_DUMMY: u16 = 0xE8;

static ENABLED: AtomicBool = AtomicBool::new(false);
static V_CONSUMED: AtomicBool = AtomicBool::new(false);
static HOOK_THREAD_ID: Mutex<Option<u32>> = Mutex::new(None);
static APP_HANDLE: OnceLock<AppHandle> = OnceLock::new();

/// 按设置开关启停 Win+V 接管钩子；幂等，可重复调用。
pub fn set_enabled(app: &AppHandle, enabled: bool) {
    if enabled {
        enable(app);
    } else {
        disable();
    }
}

fn enable(app: &AppHandle) {
    let _ = APP_HANDLE.set(app.clone());
    ENABLED.store(true, Ordering::Relaxed);

    // 已有钩子线程时不再起新线程；ENABLED 的恢复就够了。
    if HOOK_THREAD_ID
        .lock()
        .expect("win_v hook thread id poisoned")
        .is_some()
    {
        return;
    }

    std::thread::spawn(|| unsafe {
        let hook = SetWindowsHookExW(WH_KEYBOARD_LL, Some(hook_proc), null_mut(), 0);
        if hook.is_null() {
            log::error!("SetWindowsHookExW(WH_KEYBOARD_LL) for win+v failed");
            return;
        }

        *HOOK_THREAD_ID
            .lock()
            .expect("win_v hook thread id poisoned") = Some(GetCurrentThreadId());

        let mut msg: MSG = std::mem::zeroed();
        // GetMessageW 收到 WM_QUIT 返回 0 → 消息泵自然退出。
        while GetMessageW(&mut msg, null_mut(), 0, 0) > 0 {}

        UnhookWindowsHookEx(hook);
        *HOOK_THREAD_ID
            .lock()
            .expect("win_v hook thread id poisoned") = None;
        V_CONSUMED.store(false, Ordering::Relaxed);
    });
}

fn disable() {
    ENABLED.store(false, Ordering::Relaxed);

    let tid = HOOK_THREAD_ID
        .lock()
        .expect("win_v hook thread id poisoned")
        .take();
    if let Some(tid) = tid {
        unsafe {
            PostThreadMessageW(tid, WM_QUIT, 0, 0);
        }
    }
}

unsafe extern "system" fn hook_proc(code: i32, wparam: WPARAM, lparam: LPARAM) -> LRESULT {
    if code < 0 || !ENABLED.load(Ordering::Relaxed) {
        return CallNextHookEx(null_mut(), code, wparam, lparam);
    }

    let kbd = &*(lparam as *const KBDLLHOOKSTRUCT);

    // 自己注入的占位键放行，避免与真实按键混淆。
    if kbd.flags & LLKHF_INJECTED != 0 {
        return CallNextHookEx(null_mut(), code, wparam, lparam);
    }

    let vk = kbd.vkCode;
    if vk != VK_V {
        return CallNextHookEx(null_mut(), code, wparam, lparam);
    }

    let msg = wparam as UINT;

    if msg == WM_KEYDOWN || msg == WM_SYSKEYDOWN {
        let win_down = (GetAsyncKeyState(VK_LWIN) as u16) & 0x8000 != 0
            || (GetAsyncKeyState(VK_RWIN) as u16) & 0x8000 != 0;
        if !win_down {
            return CallNextHookEx(null_mut(), code, wparam, lparam);
        }

        // 按住 V 会持续触发 WM_KEYDOWN；swap 把首次按下与自动重复区分开，
        // 只在首次 toggle，避免连续开合。重复期间仍吞键。
        if V_CONSUMED.swap(true, Ordering::Relaxed) {
            return 1;
        }

        suppress_start_menu();
        schedule_toggle();

        return 1;
    }

    if (msg == WM_KEYUP || msg == WM_SYSKEYUP) && V_CONSUMED.swap(false, Ordering::Relaxed) {
        // 配对吞掉 V 抬起：按下已被拦截，放行抬起会让前台应用收到孤立 KEYUP。
        return 1;
    }

    CallNextHookEx(null_mut(), code, wparam, lparam)
}

/// 注入一次占位键的按下/抬起，打断「Win 单击」语义，阻止系统在 Win 抬起时弹开始菜单。
fn suppress_start_menu() {
    let mut inputs: [INPUT; 2] = unsafe { std::mem::zeroed() };

    inputs[0].type_ = INPUT_KEYBOARD;
    unsafe {
        *inputs[0].u.ki_mut() = KEYBDINPUT {
            wVk: VK_DUMMY,
            wScan: 0,
            dwFlags: 0,
            time: 0,
            dwExtraInfo: 0,
        };
    }
    inputs[1].type_ = INPUT_KEYBOARD;
    unsafe {
        *inputs[1].u.ki_mut() = KEYBDINPUT {
            wVk: VK_DUMMY,
            wScan: 0,
            dwFlags: KEYEVENTF_KEYUP,
            time: 0,
            dwExtraInfo: 0,
        };
    }

    let sent = unsafe {
        SendInput(
            inputs.len() as u32,
            inputs.as_mut_ptr(),
            std::mem::size_of::<INPUT>() as i32,
        )
    };
    if sent as usize != inputs.len() {
        log::warn!(
            "inject start-menu suppress key sent {sent}/{}",
            inputs.len()
        );
    }
}

/// 钩子线程不能直接操作窗口，回到主线程 toggle 主窗口。
fn schedule_toggle() {
    let Some(app) = APP_HANDLE.get() else {
        return;
    };

    let handle = app.clone();
    if let Err(err) = app.run_on_main_thread(move || {
        if let Err(err) = window::toggle_window(&handle, MAIN_WINDOW_LABEL) {
            log::warn!("toggle main window via win+v failed: {err}");
        }
    }) {
        log::warn!("schedule win+v toggle failed: {err}");
    }
}

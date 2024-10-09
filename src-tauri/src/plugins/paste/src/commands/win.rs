use super::wait;
use enigo::{
    Direction::{Click, Press, Release},
    Enigo, Key, Keyboard, Settings,
};
use std::ffi::OsString;
use std::os::windows::ffi::OsStringExt;
use std::ptr;
use std::sync::Mutex;
use tauri::command;
use tauri_plugin_eco_window::MAIN_WINDOW_TITLE;
use winapi::shared::minwindef::DWORD;
use winapi::shared::windef::{HWINEVENTHOOK, HWND};
use winapi::um::winuser::{
    GetWindowTextLengthW, GetWindowTextW, SetForegroundWindow, SetWinEventHook,
    EVENT_SYSTEM_FOREGROUND, WINEVENT_OUTOFCONTEXT,
};

static PREVIOUS_WINDOW: Mutex<Option<isize>> = Mutex::new(None);

// 获取窗口标题
unsafe fn get_window_title(hwnd: HWND) -> String {
    let length = GetWindowTextLengthW(hwnd);

    if length == 0 {
        return String::new();
    }

    let mut buffer: Vec<u16> = vec![0; (length + 1) as usize];

    GetWindowTextW(hwnd, buffer.as_mut_ptr(), length + 1);

    OsString::from_wide(&buffer[..length as usize])
        .to_string_lossy()
        .into_owned()
}

// 定义事件钩子回调函数
unsafe extern "system" fn event_hook_callback(
    _h_win_event_hook: HWINEVENTHOOK,
    event: DWORD,
    hwnd: HWND,
    _id_object: i32,
    _id_child: i32,
    _dw_event_thread: DWORD,
    _dwms_event_time: DWORD,
) {
    if event == EVENT_SYSTEM_FOREGROUND {
        let window_title = get_window_title(hwnd);

        if window_title == MAIN_WINDOW_TITLE {
            return;
        }

        let mut previous_window = PREVIOUS_WINDOW.lock().unwrap();
        let _ = previous_window.insert(hwnd as isize);
    }
}

// 监听窗口切换
pub fn observe_app() {
    unsafe {
        // 设置事件钩子
        let hook = SetWinEventHook(
            EVENT_SYSTEM_FOREGROUND,
            EVENT_SYSTEM_FOREGROUND,
            ptr::null_mut(),
            Some(event_hook_callback),
            0,
            0,
            WINEVENT_OUTOFCONTEXT,
        );

        if hook.is_null() {
            log::error!("设置事件钩子失败");
            return;
        }
    }
}

// 获取上一个窗口
pub fn get_previous_window() -> Option<isize> {
    return PREVIOUS_WINDOW.lock().unwrap().clone();
}

// 聚焦上一个窗口
fn focus_previous_window() {
    unsafe {
        let hwnd = match get_previous_window() {
            Some(hwnd) => hwnd as HWND,
            None => return,
        };

        if hwnd.is_null() {
            return;
        }

        SetForegroundWindow(hwnd);
    }
}

// 粘贴
#[command]
pub async fn paste() {
    let mut enigo = Enigo::new(&Settings::default()).unwrap();

    focus_previous_window();

    wait(100);

    enigo.key(Key::Shift, Press).unwrap();
    // insert 的微软虚拟键码：https://learn.microsoft.com/en-us/windows/win32/inputdev/virtual-key-codes
    enigo.key(Key::Other(0x2D), Click).unwrap();
    enigo.key(Key::Shift, Release).unwrap();
}

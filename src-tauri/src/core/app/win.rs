use std::sync::Mutex;
use winapi::shared::minwindef::DWORD;
use winapi::shared::windef::{HWINEVENTHOOK, HWND};
use winapi::um::winuser::{SetWinEventHook, EVENT_SYSTEM_FOREGROUND, WINEVENT_OUTOFCONTEXT};

static FOREMOST_APPS: Mutex<Vec<isize>> = Mutex::new(Vec::new());

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
        let mut app = FOREMOST_APPS.lock().unwrap();

        if app.len() >= 2 {
            app.remove(0);
        }

        app.push(hwnd as isize);
    }
}

pub fn observe_app() {
    unsafe {
        // 设置事件钩子
        let hook = SetWinEventHook(
            EVENT_SYSTEM_FOREGROUND,
            EVENT_SYSTEM_FOREGROUND,
            std::ptr::null_mut(),
            Some(event_hook_callback),
            0,
            0,
            WINEVENT_OUTOFCONTEXT,
        );

        if hook.is_null() {
            println!("设置事件钩子失败");
            return;
        }
    }
}

pub fn get_foreground_apps() -> Vec<isize> {
    return FOREMOST_APPS.lock().unwrap().to_vec();
}

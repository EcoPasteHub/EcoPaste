use std::sync::Mutex;
use winapi::um::winuser::GetForegroundWindow;
use winapi::shared::windef::HWND;
static FOREMOST_APP: Mutex<Option<HWND>> = Mutex::new(None);

/// 记录之前激活的窗口
pub fn update_foremost_apps() {
    let mut app = FOREMOST_APP.lock().unwrap();
    unsafe {
        let hwnd = GetForegroundWindow();
        if hwnd.is_null() {
            println!("Could not get active window");
        }
        *app = Some(hwnd);
    }
}

/// 获取之前激活的窗口
pub fn get_previous_foremost_app() -> Option<HWND> {
    let app = FOREMOST_APP.lock().unwrap();
    match *app {
        Some(hwnd) => {
            Some(hwnd)
        }
        None => None,
    }
}
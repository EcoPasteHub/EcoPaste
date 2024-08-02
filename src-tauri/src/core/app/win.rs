use std::sync::Mutex;

use winapi::shared::minwindef::DWORD;
use winapi::shared::windef::HWND;
use winapi::um::winuser::{EVENT_SYSTEM_FOREGROUND, SetWinEventHook, WINEVENT_OUTOFCONTEXT};

// 定义一个全局变量来存储当前前台窗口的句柄
static CURRENT_FOREGROUND_WINDOW: Mutex<Option<HWND>> = Mutex::new(None);

// 定义事件钩子回调函数
unsafe extern "system" fn event_hook_callback(hWinEventHook: winapi::shared::windef::HWINEVENTHOOK,
                                              event: DWORD,
                                              hwnd: HWND,
                                              idObject: i32,
                                              idChild: i32,
                                              dwEventThread: DWORD,
                                              dwmsEventTime: DWORD) {
    if event == EVENT_SYSTEM_FOREGROUND {
        // todo 如果前台hwnd是我们自己的窗口，不做处理
        // 更新当前前台窗口句柄
        let mut app = CURRENT_FOREGROUND_WINDOW.lock().unwrap();
        *app = Some(hwnd);
        println!("前台窗口已改变: {:?}", hwnd);
    }
}

pub fn observe_app() {
    unsafe {
        // 设置事件钩子
        let hook = SetWinEventHook(EVENT_SYSTEM_FOREGROUND, EVENT_SYSTEM_FOREGROUND,
                                   std::ptr::null_mut(), Some(event_hook_callback),
                                   0, 0, WINEVENT_OUTOFCONTEXT);
        if hook.is_null() {
            println!("设置事件钩子失败");
            return;
        }
    }
}

/// 获取之前激活的窗口
pub fn get_previous_foremost_app() -> Option<HWND> {
    let app = CURRENT_FOREGROUND_WINDOW.lock().unwrap();
    match *app {
        Some(hwnd) => {
            Some(hwnd)
        }
        None => None,
    }
}
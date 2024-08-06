use crate::core::app::get_foreground_apps;
use rdev::{simulate, EventType, Key};
use std::{thread, time};
use tauri::{
    command, generate_handler,
    plugin::{Builder, TauriPlugin},
    Wry,
};

// 让上一个窗口聚焦（macos）
#[cfg(target_os = "macos")]
fn focus_previous_window() {
    use cocoa::{
        appkit::{NSApplicationActivationOptions, NSRunningApplication},
        base::nil,
    };

    let foreground_apps = get_foreground_apps();

    let process_id = foreground_apps[0].process_id;

    unsafe {
        let app = NSRunningApplication::runningApplicationWithProcessIdentifier(nil, process_id);

        app.activateWithOptions_(
            NSApplicationActivationOptions::NSApplicationActivateIgnoringOtherApps,
        );
    }
}

// 让上一个窗口聚焦（windows）
#[cfg(target_os = "windows")]
fn focus_previous_window() {
    use winapi::{shared::windef::HWND, um::winuser::SetForegroundWindow};
    unsafe {
        let foreground_apps = get_foreground_apps();

        let hwnd = foreground_apps[0] as HWND;

        if hwnd.is_null() {
            println!("Could not get active window");
            return;
        }

        if SetForegroundWindow(hwnd) == 0 {
            println!("Could not focus on window");
        }
    }
}

// 让上一个窗口聚焦（linux）
#[cfg(target_os = "linux")]
fn focus_previous_window() {
    use x11::xlib::{self, XCloseDisplay, XOpenDisplay, XSetInputFocus};

    unsafe {
        let display = XOpenDisplay(std::ptr::null_mut());
        if display.is_null() {
            eprintln!("Could not open display");
            return;
        }
        let window = get_foreground_apps();
        let window = match window.get(0) {
            Some(window) => *window,
            None => {
                eprintln!("Could not get active window");
                return;
            }
        };
        let result = XSetInputFocus(display, window, xlib::RevertToNone, xlib::CurrentTime);
        if result != xlib::Success as i32 {
            eprintln!("Could not focus on window")
        }
        XCloseDisplay(display);
    }
}

// 线程等待（毫秒）
fn sleep(millis: u64) {
    thread::sleep(time::Duration::from_millis(millis));
}

// 模拟键盘按键
fn dispatch(event_type: &EventType) {
    sleep(20);

    simulate(event_type).unwrap();
}

// 粘贴剪贴板内容
#[command]
async fn paste() {
    focus_previous_window();

    sleep(100);

    if cfg!(target_os = "macos") {
        dispatch(&EventType::KeyPress(Key::MetaLeft));
        dispatch(&EventType::KeyPress(Key::KeyV));
        dispatch(&EventType::KeyRelease(Key::KeyV));
        dispatch(&EventType::KeyRelease(Key::MetaLeft));
    } else {
        dispatch(&EventType::KeyPress(Key::ControlLeft));
        dispatch(&EventType::KeyPress(Key::KeyV));
        dispatch(&EventType::KeyRelease(Key::KeyV));
        dispatch(&EventType::KeyRelease(Key::ControlLeft));
    }
}

pub fn init() -> TauriPlugin<Wry> {
    Builder::new("paste")
        .invoke_handler(generate_handler![paste])
        .build()
}

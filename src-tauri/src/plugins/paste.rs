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
    use crate::core::app::get_frontmost_apps;
    use cocoa::{
        appkit::{NSApplicationActivationOptions, NSRunningApplication},
        base::nil,
    };

    let frontmost_apps = get_frontmost_apps();

    let process_id = frontmost_apps[0].process_id;

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
    use winapi::um::winuser::{GetForegroundWindow, SetForegroundWindow};

    unsafe {
        let hwnd = GetForegroundWindow();

        if hwnd.is_null() {
            println!("Could not get active window");
            return;
        }

        if SetForegroundWindow(hwnd) == 0 {
            println!("Could not focus on window");
        }
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

// 粘贴剪切板内容
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

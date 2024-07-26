use crate::core;
use rdev::{simulate, EventType, Key};
use std::{thread, time};
use tauri::{
    command, generate_handler,
    plugin::{Builder, TauriPlugin},
    AppHandle, Wry,
};

// 让上一个窗口聚焦（macos）
#[cfg(target_os = "macos")]
fn focus_previous_window(process_id: i32) {
    use cocoa::{
        appkit::{NSApplicationActivationOptions, NSRunningApplication},
        base::nil,
    };

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

// 粘贴剪切板内容（macos）
#[cfg(target_os = "macos")]
#[command]
async fn paste(app_handle: AppHandle) {
    let app_name = app_handle.package_info().name.clone();

    let frontmost_apps = core::app::get_frontmost_apps();

    if let Some(app) = frontmost_apps.iter().find(|app| app.name != app_name) {
        let process_id = app.process_id;

        focus_previous_window(process_id);
    } else {
        return;
    }

    dispatch(&EventType::KeyPress(Key::MetaLeft));
    dispatch(&EventType::KeyPress(Key::KeyV));
    dispatch(&EventType::KeyRelease(Key::KeyV));
    dispatch(&EventType::KeyRelease(Key::MetaLeft));
}

// 粘贴剪切板内容（windows）
#[cfg(target_os = "windows")]
#[command]
async fn paste() {
    focus_previous_window();

    sleep(100);

    dispatch(&EventType::KeyPress(Key::ControlLeft));
    dispatch(&EventType::KeyPress(Key::KeyV));
    dispatch(&EventType::KeyRelease(Key::KeyV));
    dispatch(&EventType::KeyRelease(Key::ControlLeft));
}

pub fn init() -> TauriPlugin<Wry> {
    Builder::new("paste")
        .invoke_handler(generate_handler![paste])
        .build()
}

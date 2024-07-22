use rdev::{simulate, EventType, Key};
use std::{thread, time};
use tauri::{
    command, generate_handler,
    plugin::{Builder, TauriPlugin},
    Wry,
};

#[cfg(target_os = "macos")]
use std::sync::Mutex;

// 上一个窗口的进程 id
#[cfg(target_os = "macos")]
static PREVIOUS_PROCESS_ID: Mutex<u64> = Mutex::new(0);

// 获取上一个窗口的进程 id
#[cfg(target_os = "macos")]
pub fn get_previous_process_id(window: tauri::Window) {
    use crate::plugins::window::MAIN_WINDOW_LABEL;

    let label = window.label();

    if label != MAIN_WINDOW_LABEL {
        return;
    }

    use active_win_pos_rs::get_active_window;

    let active_window = get_active_window().unwrap();

    let mut previous_process_id = PREVIOUS_PROCESS_ID.lock().unwrap();

    *previous_process_id = active_window.process_id;
}

// 让上一个窗口聚焦（macos）
#[cfg(target_os = "macos")]
fn focus_previous_window() {
    use cocoa::{
        appkit::{
            NSApplicationActivationOptions::NSApplicationActivateIgnoringOtherApps,
            NSRunningApplication,
        },
        base::nil,
    };

    let previous_process_id = *PREVIOUS_PROCESS_ID.lock().unwrap();

    if previous_process_id == 0 {
        return;
    }

    unsafe {
        let app = NSRunningApplication::runningApplicationWithProcessIdentifier(
            nil,
            previous_process_id as i32,
        );

        app.activateWithOptions_(NSApplicationActivateIgnoringOtherApps);
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

// 模拟键盘按键
fn dispatch(event_type: &EventType) {
    thread::sleep(time::Duration::from_millis(20));

    simulate(event_type).unwrap();
}

// 粘贴剪切板内容
#[command]
async fn paste() {
    focus_previous_window();

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

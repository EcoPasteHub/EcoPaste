use active_win_pos_rs::get_active_window;
use std::sync::Mutex;
use tauri::{
    command, generate_handler,
    plugin::{Builder, TauriPlugin},
    utils::config::WindowConfig,
    AppHandle, Manager, Window, WindowBuilder, Wry,
};

// 主窗口的label
pub static MAIN_WINDOW_LABEL: &str = "main";
// 偏好设置窗口的label
pub static PREFERENCE_WINDOW_LABEL: &str = "preference";

// 上一个窗口的进程 id
static PREVIOUS_PROCESS_ID: Mutex<u64> = Mutex::new(0);

// 创建窗口
#[command]
pub async fn create_window(app_handle: AppHandle, label: String, mut options: WindowConfig) {
    if let Some(window) = app_handle.get_window(&label) {
        show_window(window).await;
    } else {
        options.label = label.to_string();

        WindowBuilder::from_config(&app_handle, options.clone())
            .build()
            .unwrap();
    }
}

// 显示窗口
#[command]
pub async fn show_window(window: Window) {
    let label = window.label();

    if label == MAIN_WINDOW_LABEL {
        let active_window = get_active_window().unwrap();

        let mut previous_process_id = PREVIOUS_PROCESS_ID.lock().unwrap();

        *previous_process_id = active_window.process_id;
    }

    #[cfg(not(target_os = "linux"))]
    {
        window.show().unwrap();
        window.unminimize().unwrap();
        window.set_focus().unwrap();
    }

    #[cfg(target_os = "linux")]
    {
        let position = window.outer_position().unwrap();
        window.hide().unwrap();
        window
            .set_position(tauri::PhysicalPosition::new(position.x, position.y))
            .unwrap();
        window.show().unwrap();
    }
}

// 隐藏窗口
#[command]
pub async fn hide_window(window: Window) {
    window.hide().unwrap();
}

// 给窗口添加阴影
#[command]
pub async fn set_window_shadow(_window: Window) {
    #[cfg(not(target_os = "linux"))]
    window_shadows::set_shadow(&_window, true).unwrap();
}

// 让上一个窗口聚焦
pub fn focus_previous_window() {
    let process_id = *PREVIOUS_PROCESS_ID.lock().unwrap();

    if process_id == 0 {
        return;
    }

    #[cfg(target_os = "macos")]
    {
        use cocoa::{
            appkit::{
                NSApplicationActivationOptions::NSApplicationActivateIgnoringOtherApps,
                NSRunningApplication,
            },
            base::nil,
        };

        unsafe {
            let app = NSRunningApplication::runningApplicationWithProcessIdentifier(
                nil,
                process_id as i32,
            );

            app.activateWithOptions_(NSApplicationActivateIgnoringOtherApps);
        }
    }

    #[cfg(target_os = "windows")]
    {
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
}

pub fn init() -> TauriPlugin<Wry> {
    Builder::new("window")
        .invoke_handler(generate_handler![
            create_window,
            show_window,
            hide_window,
            set_window_shadow
        ])
        .build()
}

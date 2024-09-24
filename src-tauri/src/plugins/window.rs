use tauri::{
    async_runtime, command, generate_handler,
    plugin::{Builder, TauriPlugin},
    AppHandle, Manager, Window, Wry,
};

// 主窗口的label
pub static MAIN_WINDOW_LABEL: &str = "main";
// 偏好设置窗口的label
pub static PREFERENCE_WINDOW_LABEL: &str = "preference";
// 主窗口的title
pub static MAIN_WINDOW_TITLE: &str = "EcoPaste";

// 显示窗口（非linux）
#[cfg(not(target_os = "linux"))]
#[command]
pub async fn show_window(window: Window) {
    window.show().unwrap();
    window.unminimize().unwrap();
    window.set_focus().unwrap();
}

// 显示窗口（linux）
#[cfg(target_os = "linux")]
#[command]
pub async fn show_window(window: Window) {
    let position = window.outer_position().unwrap();
    let physical_position = tauri::PhysicalPosition::new(position.x, position.y);

    window.hide().unwrap();
    window.set_position(physical_position).unwrap();
    window.show().unwrap();
}

// 隐藏窗口
#[command]
pub async fn hide_window(window: Window) {
    window.hide().unwrap();
}

// 显示主窗口
pub fn show_main_window(app_handle: &AppHandle) {
    let window = app_handle.get_window(MAIN_WINDOW_LABEL).unwrap();

    async_runtime::spawn(async move {
        show_window(window).await;
    });
}

// 显示偏好设置窗口
pub fn show_preference_window(app_handle: &AppHandle) {
    let window = app_handle.get_window(PREFERENCE_WINDOW_LABEL).unwrap();

    async_runtime::spawn(async move {
        show_window(window).await;
    });
}

// 显示任务栏图标
#[command]
pub fn show_taskbar_icon(window: Window, show: bool) {
    #[cfg(not(target_os = "macos"))]
    window.set_skip_taskbar(!show).unwrap();

    #[cfg(target_os = "macos")]
    {
        use cocoa::appkit::{NSApp, NSApplication, NSApplicationActivationPolicy::*};

        unsafe {
            let app = NSApp();

            if show {
                app.setActivationPolicy_(NSApplicationActivationPolicyRegular);
            } else {
                app.setActivationPolicy_(NSApplicationActivationPolicyAccessory);
            }
        }
    }

    let _ = window;
}

pub fn init() -> TauriPlugin<Wry> {
    Builder::new("window")
        .invoke_handler(generate_handler![
            show_window,
            hide_window,
            show_taskbar_icon
        ])
        .build()
}

use tauri::{
    command, generate_handler,
    plugin::{Builder, TauriPlugin},
    utils::config::WindowConfig,
    AppHandle, Manager, Window, WindowBuilder, Wry,
};

// 主窗口的label
// pub static MAIN_WINDOW_LABEL: &str = "main";
// 偏好设置窗口的label
pub static PREFERENCE_WINDOW_LABEL: &str = "preference";

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
    window.hide().unwrap();
    window
        .set_position(tauri::PhysicalPosition::new(position.x, position.y))
        .unwrap();
    window.show().unwrap();
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

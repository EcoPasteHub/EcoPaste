use tauri::{
    command, generate_handler,
    plugin::{Builder, TauriPlugin},
    utils::config::WindowConfig,
    AppHandle, Manager, Window, WindowBuilder, Wry,
};

// 主窗口的名称
pub static MAIN_WINDOW_LABEL: &str = "main";

// 创建窗口
#[command]
pub async fn create_window(app_handle: AppHandle, label: String, mut options: WindowConfig) {
    if let Some(window) = app_handle.get_window(&label) {
        show_window(window).await;
    } else {
        options.label = label.to_string();

        let _window = WindowBuilder::from_config(&app_handle, options.clone())
            .build()
            .unwrap();

        #[cfg(not(target_os = "linux"))]
        {
            if !options.decorations {
                window_shadows::set_shadow(&_window, true).unwrap();
            }
        }
    }
}

// 显示窗口
#[command]
pub async fn show_window(window: Window) {
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

pub fn init() -> TauriPlugin<Wry> {
    Builder::new("window")
        .invoke_handler(generate_handler![create_window, show_window, hide_window,])
        .build()
}

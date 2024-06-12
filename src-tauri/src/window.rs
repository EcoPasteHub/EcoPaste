use tauri::{command, utils::config::WindowConfig, AppHandle, Manager, Window, WindowBuilder};

// 主窗口的名称
pub static MAIN_WINDOW_LABEL: &str = "main";

// 创建窗口
#[command]
pub async fn create_window(app_handle: AppHandle, label: String, mut options: WindowConfig) {
    if let Some(window) = app_handle.get_window(&label) {
        show_window(window);
    } else {
        options.label = label.to_string();

        WindowBuilder::from_config(&app_handle, options)
            .build()
            .unwrap();
    }
}

// 显示窗口
#[command]
pub fn show_window(window: Window) {
    window.show().unwrap();
    window.unminimize().unwrap();
    window.set_focus().unwrap();
}

// 隐藏窗口
#[command]
pub fn hide_window(window: Window) {
    window.hide().unwrap();
}

// 退出 app
#[command]
pub fn quit_app() {
    std::process::exit(0)
}

// 磨砂窗口：https://github.com/tauri-apps/window-vibrancy
#[command]
pub fn frosted_window(_window: Window) {
    #[cfg(target_os = "macos")]
    window_vibrancy::apply_vibrancy(
        &_window,
        window_vibrancy::NSVisualEffectMaterial::HeaderView,
        Some(window_vibrancy::NSVisualEffectState::Active),
        Some(10.0),
    )
    .unwrap();
}

// 窗口阴影：https://github.com/tauri-apps/window-shadows
#[command]
pub fn set_window_shadow(_window: Window) {
    #[cfg(target_os = "windows")]
    window_shadows::set_shadow(&window, true)
}

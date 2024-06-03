use tauri::{command, utils::config::WindowConfig, AppHandle, Manager, Window, WindowBuilder};
use window_vibrancy::*;

pub static MAIN_WINDOW_LABEL: &str = "main";

#[command]
pub fn create_window(app_handle: AppHandle, label: String, mut options: WindowConfig) {
    let window = app_handle.get_window(&label);

    if window.is_some() {
        return show_window(window.clone().unwrap());
    }

    options.label = label.clone();

    Some(
        WindowBuilder::from_config(&app_handle, options)
            .build()
            .expect("failed to create window"),
    )
    .unwrap();
}

#[command]
pub fn show_window(window: Window) {
    window.show().unwrap();
    window.unminimize().unwrap();
    window.set_focus().unwrap();
}

#[command]
pub fn hide_window(window: Window) {
    window.hide().unwrap();
}

#[command]
pub fn quit_app() {
    std::process::exit(0)
}

#[command]
// https://github.com/tauri-apps/window-vibrancy
pub fn frosted_window(window: Window) {
    #[cfg(target_os = "macos")]
    apply_vibrancy(&window, NSVisualEffectMaterial::HudWindow, None, None)
        .expect("Unsupported platform! 'apply_vibrancy' is only supported on macOS");

    #[cfg(target_os = "windows")]
    apply_blur(&window, Some((18, 18, 18, 125)))
        .expect("Unsupported platform! 'apply_blur' is only supported on Windows");
}

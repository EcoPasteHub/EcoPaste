use tauri::{App, WebviewWindow};

#[cfg(target_os = "macos")]
mod mac;

#[cfg(target_os = "windows")]
mod win;

#[cfg(target_os = "linux")]
mod linux;

#[cfg(target_os = "macos")]
pub use mac::*;

#[cfg(target_os = "windows")]
pub use win::*;

#[cfg(target_os = "linux")]
pub use linux::*;

pub fn default(app: &mut App, main_window: WebviewWindow, preference_window: WebviewWindow) {
    // 开发模式自动打开控制台：https://tauri.app/develop/debug
    #[cfg(any(dev, debug_assertions))]
    main_window.open_devtools();

    platform(app, main_window.clone(), preference_window.clone());
}

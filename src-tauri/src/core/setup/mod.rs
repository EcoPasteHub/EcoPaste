use crate::AUTO_LAUNCH_ARG;
use std::env;
use tauri::{App, Manager, WebviewWindow};

#[cfg(target_os = "macos")]
mod mac;

#[cfg(target_os = "windows")]
mod win;

#[cfg(target_os = "linux")]
mod linux;

#[cfg(target_os = "macos")]
pub use mac::*;

use tauri_plugin_eco_window::show_preference_window;
#[cfg(target_os = "windows")]
pub use win::*;

#[cfg(target_os = "linux")]
pub use linux::*;

pub fn default(app: &mut App, main_window: WebviewWindow, preference_window: WebviewWindow) {
    let app_handle = app.app_handle();

    // 判断是否为自动启动
    let args: Vec<String> = env::args().collect();
    if !args.contains(&AUTO_LAUNCH_ARG.to_string()) {
        show_preference_window(&app_handle);
    }

    // 自动打开控制台：https://tauri.app/develop/debug
    #[cfg(any(dev, debug_assertions))]
    main_window.open_devtools();

    // 给窗口添加阴影
    let _ = main_window.set_shadow(true);

    platform(app, main_window.clone(), preference_window.clone());
}

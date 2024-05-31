// Prevents additional console window on Windows in release, DO NOT REMOVE!!
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use tauri::{generate_context, generate_handler, Builder, Manager};
use window_vibrancy::*;

fn main() {
    Builder::default()
        .setup(|_app| {
            let window = _app.get_window("main").unwrap();

            // 在开发环境中启动时打开控制台：https://tauri.app/v1/guides/debugging/application/#opening-devtools-programmatically
            #[cfg(debug_assertions)]
            {
                window.open_devtools();
            }

            // 磨砂窗口：https://github.com/tauri-apps/window-vibrancy
            #[cfg(target_os = "macos")]
            apply_vibrancy(&window, NSVisualEffectMaterial::HudWindow, None, None)
                .expect("Unsupported platform! 'apply_vibrancy' is only supported on macOS");

            #[cfg(target_os = "windows")]
            apply_blur(&window, Some((18, 18, 18, 125)))
                .expect("Unsupported platform! 'apply_blur' is only supported on Windows");

            Ok(())
        })
        .plugin(tauri_plugin_clipboard::init())
        .invoke_handler(generate_handler![])
        .run(generate_context!())
        .expect("error while running tauri application");
}

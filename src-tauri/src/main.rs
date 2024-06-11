// Prevents additional console window on Windows in release, DO NOT REMOVE!!
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use tauri::{generate_context, generate_handler, Builder, Manager, WindowEvent};
use tauri_plugin_theme::ThemePlugin;
mod tray;
mod window;
use tauri_plugin_autostart::MacosLauncher;
use window::{
    create_window, frosted_window, hide_window, quit_app, show_window, MAIN_WINDOW_LABEL,
};

fn main() {
    let mut ctx = generate_context!();

    Builder::default()
        .setup(|_app| {
            // 在开发环境中启动时打开控制台：https://tauri.app/v1/guides/debugging/application/#opening-devtools-programmatically
            #[cfg(debug_assertions)]
            {
                let window = _app.get_window(MAIN_WINDOW_LABEL).unwrap();
                window.open_devtools();
            }

            // 隐藏 mac 下的任务栏图标：https://github.com/tauri-apps/tauri/issues/4852#issuecomment-1312716378
            #[cfg(target_os = "macos")]
            _app.set_activation_policy(tauri::ActivationPolicy::Accessory);

            Ok(())
        })
        // 剪切板插件：https://github.com/CrossCopy/tauri-plugin-clipboard
        .plugin(tauri_plugin_clipboard::init())
        // 主题插件：https://github.com/wyhaya/tauri-plugin-theme
        .plugin(ThemePlugin::init(ctx.config_mut()))
        // 确保在 windows 和 linux 上只有一个 app 实例在运行：https://github.com/tauri-apps/plugins-workspace/tree/v1/plugins/single-instance
        .plugin(tauri_plugin_single_instance::init(|app, _argv, _cwd| {
            let window = app.get_window(MAIN_WINDOW_LABEL).unwrap();

            show_window(window);
        }))
        // app 自启动：https://github.com/tauri-apps/tauri-plugin-autostart
        .plugin(tauri_plugin_autostart::init(
            MacosLauncher::LaunchAgent,
            Some(vec!["--flag1", "--flag2"]),
        ))
        // 数据库：https://github.com/tauri-apps/tauri-plugin-sql
        .plugin(tauri_plugin_sql::Builder::default().build())
        // fs拓展插件：https://github.com/tauri-apps/tauri-plugin-fs-extra
        .plugin(tauri_plugin_fs_extra::init())
        // 系统托盘：https://tauri.app/v1/guides/features/system-tray
        .system_tray(tray::menu())
        .on_system_tray_event(tray::handler)
        .invoke_handler(generate_handler![
            create_window,
            hide_window,
            show_window,
            quit_app,
            frosted_window
        ])
        // 让 app 保持在后台运行：https://tauri.app/v1/guides/features/system-tray/#preventing-the-app-from-closing
        .on_window_event(|event| match event.event() {
            WindowEvent::CloseRequested { api, .. } => {
                event.window().hide().unwrap();
                api.prevent_close();
            }
            _ => {}
        })
        .run(ctx)
        .expect("error while running tauri application");
}

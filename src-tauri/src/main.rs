// Prevents additional console window on Windows in release, DO NOT REMOVE!!
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod plugins;
mod tray;

use plugins::{
    backup, clipboard, fs_extra, mouse, ocr,
    window::{self, show_window, MAIN_WINDOW_LABEL},
};
use tauri::{async_runtime, generate_context, generate_handler, Builder, Manager, WindowEvent};
use tauri_plugin_autostart::MacosLauncher;
use tauri_plugin_theme::ThemePlugin;

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
        // 主题插件：https://github.com/wyhaya/tauri-plugin-theme
        .plugin(ThemePlugin::init(ctx.config_mut()))
        // 确保在 windows 和 linux 上只有一个 app 实例在运行：https://github.com/tauri-apps/plugins-workspace/tree/v1/plugins/single-instance
        .plugin(tauri_plugin_single_instance::init(|app, _argv, _cwd| {
            let window = app.get_window(MAIN_WINDOW_LABEL).unwrap();

            async_runtime::block_on(async move {
                show_window(window).await;
            });
        }))
        // app 自启动：https://github.com/tauri-apps/tauri-plugin-autostart
        .plugin(tauri_plugin_autostart::init(
            MacosLauncher::LaunchAgent,
            Some(vec!["--flag1", "--flag2"]),
        ))
        // 数据库：https://github.com/tauri-apps/tauri-plugin-sql
        .plugin(tauri_plugin_sql::Builder::default().build())
        // 自定义的窗口管理插件
        .plugin(window::init())
        // 自定义的 fs_extra 插件
        .plugin(fs_extra::init())
        // 自定义剪切板插件
        .plugin(clipboard::init())
        // 右键菜单插件：https://github.com/c2r0b/tauri-plugin-context-menu
        .plugin(tauri_plugin_context_menu::init())
        // 自定义鼠标相关的插件
        .plugin(mouse::init())
        // 自定义图片识别插件
        .plugin(ocr::init())
        // 自定义备份插件
        .plugin(backup::init())
        // 系统托盘：https://tauri.app/v1/guides/features/system-tray
        .system_tray(tray::menu())
        .on_system_tray_event(tray::handler)
        .invoke_handler(generate_handler![])
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

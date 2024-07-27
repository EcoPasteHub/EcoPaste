// Prevents additional console window on Windows in release, DO NOT REMOVE!!
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod core;
mod locales;
mod plugins;

use core::tray;
use plugins::{
    auto_launch, backup, clipboard, fs_extra, locale, mouse, ocr, paste,
    window::{self, show_window, PREFERENCE_WINDOW_LABEL},
};
use tauri::{
    async_runtime, generate_context, generate_handler, Builder, Manager, SystemTray, WindowEvent,
};
use tauri_plugin_autostart::MacosLauncher;
use tauri_plugin_theme::ThemePlugin;

pub const AUTO_LAUNCH_ARG: &str = "--auto-launch";

fn main() {
    let mut ctx = generate_context!();

    Builder::default()
        .setup(|app| {
            // 在开发环境中启动时打开控制台：https://tauri.app/v1/guides/debugging/application/#opening-devtools-programmatically
            #[cfg(any(debug_assertions, feature = "devtools"))]
            {
                let window = app.get_window(PREFERENCE_WINDOW_LABEL).unwrap();

                window.open_devtools();
            }

            #[cfg(target_os = "macos")]
            {
                // 隐藏 mac 下的任务栏图标：https://github.com/tauri-apps/tauri/issues/4852#issuecomment-1312716378
                // BUG: 打包之后主窗口没办法显示在全屏的屏幕上
                app.set_activation_policy(tauri::ActivationPolicy::Accessory);

                core::app::observe_app();
            }

            Ok(())
        })
        // 主题插件：https://github.com/wyhaya/tauri-plugin-theme
        .plugin(ThemePlugin::init(ctx.config_mut()))
        // 确保在 windows 和 linux 上只有一个 app 实例在运行：https://github.com/tauri-apps/plugins-workspace/tree/v1/plugins/single-instance
        .plugin(tauri_plugin_single_instance::init(|app, _argv, _cwd| {
            let window = app.get_window(PREFERENCE_WINDOW_LABEL).unwrap();

            async_runtime::block_on(async move {
                show_window(window).await;
            });
        }))
        // app 自启动：https://github.com/tauri-apps/tauri-plugin-autostart
        .plugin(tauri_plugin_autostart::init(
            MacosLauncher::LaunchAgent,
            Some(vec![AUTO_LAUNCH_ARG]),
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
        // 自定义语言相关的插件
        .plugin(locale::init()) // 系统托盘：https://tauri.app/v1/guides/features/system-tray
        // 自定义粘贴的插件
        .plugin(paste::init())
        // 自定义判断是否自动启动的插件
        .plugin(auto_launch::init())
        // 系统托盘：https://tauri.app/v1/guides/features/system-tray
        .system_tray(SystemTray::new())
        .on_system_tray_event(tray::Tray::handler)
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

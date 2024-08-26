// Prevents additional console window on Windows in release, DO NOT REMOVE!!
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod core;
mod locales;
mod plugins;

use core::{error::redirect_panic_to_log, setup, tray};
use plugins::{
    backup, clipboard, fs_extra, locale, macos_permissions, mouse, ocr, paste,
    window::{self, show_window, MAIN_WINDOW_LABEL, PREFERENCE_WINDOW_LABEL},
};
use std::env;
use tauri::{
    async_runtime, generate_context, generate_handler, Builder, Manager, SystemTray, WindowEvent,
};
use tauri_plugin_autostart::MacosLauncher;
use tauri_plugin_window_state::StateFlags;

pub const AUTO_LAUNCH_ARG: &str = "--auto-launch";

fn main() {
    redirect_panic_to_log();

    if cfg!(target_os = "linux") {
        std::env::set_var("WEBKIT_DISABLE_COMPOSITING_MODE", "1");
    }

    let mut ctx = generate_context!();

    let package_info = ctx.package_info();
    let app_name = &package_info.name;
    let app_version = &package_info.version;
    let tooltip = format!("{app_name} v{app_version}");

    let log_builder = {
        let builder =
            tauri_plugin_log::Builder::new().targets([tauri_plugin_log::LogTarget::LogDir]);

        if cfg!(debug_assertions) {
            builder.target(tauri_plugin_log::LogTarget::Stderr)
        } else {
            builder
        }
    };

    Builder::default()
        .setup(|app| {
            let main_window = app.get_window(MAIN_WINDOW_LABEL).unwrap();

            let preference_window = app.get_window(PREFERENCE_WINDOW_LABEL).unwrap();

            setup::default(app, main_window.clone(), preference_window.clone());

            setup::extra(app, main_window.clone(), preference_window.clone());

            core::app::observe_app();

            Ok(())
        })
        // 主题插件：https://github.com/wyhaya/tauri-plugin-theme
        .plugin(tauri_plugin_theme::init(ctx.config_mut()))
        // 确保在 windows 和 linux 上只有一个 app 实例在运行：https://github.com/tauri-apps/plugins-workspace/tree/v1/plugins/single-instance
        .plugin(tauri_plugin_single_instance::init(
            |app_handle, _argv, _cwd| {
                let window = app_handle.get_window(MAIN_WINDOW_LABEL).unwrap();

                async_runtime::block_on(async move {
                    show_window(window).await;
                });
            },
        ))
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
        // 自定义剪贴板插件
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
        // 日志插件：https://github.com/tauri-apps/tauri-plugin-log
        .plugin(log_builder.build())
        // 记住窗口状态的插件：https://github.com/tauri-apps/plugins-workspace/tree/v1/plugins/window-state
        .plugin(
            tauri_plugin_window_state::Builder::default()
                .with_state_flags(StateFlags::all() & !StateFlags::VISIBLE)
                .build(),
        )
        // macos 权限查询的插件
        .plugin(macos_permissions::init())
        // 系统托盘：https://tauri.app/v1/guides/features/system-tray
        .system_tray(SystemTray::new().with_tooltip(&tooltip))
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

// Prevents additional console window on Windows in release, DO NOT REMOVE!!
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod core;
mod locales;
mod plugins;

use core::setup;
use plugins::{
    backup, clipboard, fs_extra, locale, macos_permissions, mouse, ocr, paste, tray, updater,
    window::{self, show_main_window, MAIN_WINDOW_LABEL, PREFERENCE_WINDOW_LABEL},
};
use tauri::{generate_context, generate_handler, Builder, Manager, WindowEvent};
use tauri_plugin_autostart::MacosLauncher;
use tauri_plugin_log::LogTarget;
use tauri_plugin_window_state::{AppHandleExt, StateFlags};

pub const AUTO_LAUNCH_ARG: &str = "--auto-launch";

fn main() {
    let mut context = generate_context!();

    if cfg!(target_os = "linux") {
        std::env::set_var("WEBKIT_DISABLE_COMPOSITING_MODE", "1");
    }

    let app = Builder::default()
        .setup(|app| {
            let main_window = app.get_window(MAIN_WINDOW_LABEL).unwrap();

            let preference_window = app.get_window(PREFERENCE_WINDOW_LABEL).unwrap();

            setup::default(app, main_window.clone(), preference_window.clone());

            Ok(())
        })
        // 主题插件：https://github.com/wyhaya/tauri-plugin-theme
        .plugin(tauri_plugin_theme::init(context.config_mut()))
        // 确保在 windows 和 linux 上只有一个 app 实例在运行：https://github.com/tauri-apps/plugins-workspace/tree/v1/plugins/single-instance
        .plugin(tauri_plugin_single_instance::init(
            |app_handle, _argv, _cwd| {
                show_main_window(app_handle);
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
        .plugin(
            tauri_plugin_log::Builder::default()
                .targets([LogTarget::LogDir, LogTarget::Stdout, LogTarget::Webview])
                .build(),
        )
        // 记住窗口状态的插件：https://github.com/tauri-apps/plugins-workspace/tree/v1/plugins/window-state
        .plugin(
            tauri_plugin_window_state::Builder::default()
                .with_state_flags(StateFlags::all() & !StateFlags::VISIBLE)
                .build(),
        )
        // macos 权限查询的插件
        .plugin(macos_permissions::init())
        // 自定义托盘插件
        .plugin(tray::init())
        // 自定义更新插件
        .plugin(updater::init())
        .invoke_handler(generate_handler![])
        .on_window_event(|event| match event.event() {
            // 让 app 保持在后台运行：https://tauri.app/v1/guides/features/system-tray/#preventing-the-app-from-closing
            WindowEvent::CloseRequested { api, .. } => {
                event.window().hide().unwrap();

                api.prevent_close();
            }
            // 窗口失焦保存窗口的状态信息
            WindowEvent::Focused(focused) => {
                if *focused {
                    return;
                }

                let app_handle = event.window().app_handle();

                let _ = app_handle.save_window_state(StateFlags::all());
            }
            _ => {}
        })
        .build(context)
        .expect("error while running tauri application");

    app.run(|app_handle, event| match event {
        #[cfg(target_os = "macos")]
        tauri::RunEvent::Reopen {
            has_visible_windows,
            ..
        } => {
            if has_visible_windows {
                return;
            }

            plugins::window::show_preference_window(app_handle);
        }
        _ => {
            let _ = app_handle;
        }
    });
}

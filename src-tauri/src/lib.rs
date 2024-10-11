mod core;

use core::setup;
use tauri::{generate_context, Builder, Manager, WindowEvent};
use tauri_plugin_autostart::MacosLauncher;
use tauri_plugin_eco_window::{show_main_window, MAIN_WINDOW_LABEL, PREFERENCE_WINDOW_LABEL};
use tauri_plugin_log::{Target, TargetKind};
// use tauri_plugin_window_state::{AppHandleExt, StateFlags};

pub const AUTO_LAUNCH_ARG: &str = "--auto-launch";

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let app = Builder::default()
        .setup(|app| {
            let main_window = app.get_webview_window(MAIN_WINDOW_LABEL).unwrap();

            let preference_window = app.get_webview_window(PREFERENCE_WINDOW_LABEL).unwrap();

            setup::default(app, main_window.clone(), preference_window.clone());

            Ok(())
        })
        // 系统 shell 插件：https://github.com/tauri-apps/tauri-plugin-shell
        .plugin(tauri_plugin_shell::init())
        // 确保在 windows 和 linux 上只有一个 app 实例在运行：https://github.com/tauri-apps/plugins-workspace/tree/v2/plugins/single-instance
        .plugin(tauri_plugin_single_instance::init(
            |app_handle, _argv, _cwd| {
                show_main_window(app_handle);
            },
        ))
        // app 自启动：https://github.com/tauri-apps/tauri-plugin-autostart/tree/v2
        .plugin(tauri_plugin_autostart::init(
            MacosLauncher::LaunchAgent,
            Some(vec![AUTO_LAUNCH_ARG]),
        ))
        // 数据库：https://github.com/tauri-apps/tauri-plugin-sql/tree/v2
        .plugin(tauri_plugin_sql::Builder::default().build())
        // 日志插件：https://github.com/tauri-apps/tauri-plugin-log/tree/v2
        .plugin(
            tauri_plugin_log::Builder::new()
                .targets([
                    Target::new(TargetKind::Stdout),
                    Target::new(TargetKind::LogDir { file_name: None }),
                    Target::new(TargetKind::Webview),
                ])
                .build(),
        )
        // TODO: 窗口状态插件
        // 记住窗口状态的插件：https://github.com/tauri-apps/plugins-workspace/tree/v2/plugins/window-state
        // .plugin(
        //     tauri_plugin_window_state::Builder::default()
        //         .with_state_flags(StateFlags::all() & !StateFlags::VISIBLE)
        //         .build(),
        // )
        // 快捷键插件: https://github.com/tauri-apps/tauri-plugin-global-shortcut
        .plugin(tauri_plugin_global_shortcut::Builder::new().build())
        // 操作系统相关信息插件：https://github.com/tauri-apps/tauri-plugin-os
        .plugin(tauri_plugin_os::init())
        // 系统级别对话框插件：https://github.com/tauri-apps/tauri-plugin-dialog
        .plugin(tauri_plugin_dialog::init())
        // 访问文件系统插件：https://github.com/tauri-apps/tauri-plugin-fs
        .plugin(tauri_plugin_fs::init())
        // 更新插件：https://github.com/tauri-apps/tauri-plugin-updater
        .plugin(tauri_plugin_updater::Builder::new().build())
        // 进程相关插件：https://github.com/tauri-apps/tauri-plugin-process
        .plugin(tauri_plugin_process::init())
        // 自定义的窗口管理插件
        .plugin(tauri_plugin_eco_window::init())
        // 自定义的 fs_extra 插件
        .plugin(tauri_plugin_eco_fs_extra::init())
        // 自定义剪贴板插件
        .plugin(tauri_plugin_eco_clipboard::init())
        // 自定义鼠标相关的插件
        .plugin(tauri_plugin_eco_mouse::init())
        // 自定义图片识别插件
        .plugin(tauri_plugin_eco_ocr::init())
        // 自定义备份插件
        .plugin(tauri_plugin_eco_backup::init())
        // 自定义语言相关的插件
        .plugin(tauri_plugin_eco_locale::init())
        // 自定义粘贴的插件
        .plugin(tauri_plugin_eco_paste::init())
        // 自定义 macos 权限查询的插件
        .plugin(tauri_plugin_eco_macos_permissions::init())
        .on_window_event(|window, event| match event {
            // 让 app 保持在后台运行：https://tauri.app/v1/guides/features/system-tray/#preventing-the-app-from-closing
            WindowEvent::CloseRequested { api, .. } => {
                window.hide().unwrap();

                api.prevent_close();
            }
            // 窗口失焦保存窗口的状态信息
            // WindowEvent::Focused(focused) => {
            //     if *focused {
            //         return;
            //     }

            //     let app_handle = window.app_handle();

            //     let _ = app_handle.save_window_state(StateFlags::all());
            // }
            _ => {}
        })
        .invoke_handler(tauri::generate_handler![])
        .build(generate_context!())
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

            tauri_plugin_eco_window::show_preference_window(app_handle);
        }
        _ => {
            let _ = app_handle;
        }
    });
}

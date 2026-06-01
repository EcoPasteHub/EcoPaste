mod autostart;
mod clipboard;
mod commands;
mod core;
mod db;
#[cfg(target_os = "windows")]
mod keyboard;
mod keystroke;
mod settings;
mod shortcut;
mod tray;
mod window;

use tauri::{Manager, WindowEvent};

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    // Webview target 把日志回灌到前端 devtools console，只在 dev 启用；
    // 生产环境只落 LogDir 文件 + Stdout，避免向用户的 webview console 喷日志。
    let mut log_targets = vec![
        tauri_plugin_log::Target::new(tauri_plugin_log::TargetKind::Stdout),
        tauri_plugin_log::Target::new(tauri_plugin_log::TargetKind::LogDir { file_name: None }),
    ];
    if cfg!(debug_assertions) {
        log_targets.push(tauri_plugin_log::Target::new(
            tauri_plugin_log::TargetKind::Webview,
        ));
    }

    let log_plugin = tauri_plugin_log::Builder::new()
        .level(if cfg!(debug_assertions) {
            log::LevelFilter::Debug
        } else {
            log::LevelFilter::Info
        })
        .targets(log_targets)
        .build();

    tauri::Builder::default()
        .plugin(tauri_plugin_single_instance::init(
            |app_handle, _argv, _cwd| {
                if let Err(err) = window::show_window(app_handle, window::PREFERENCE_WINDOW_LABEL) {
                    log::error!("show preference window on second instance failed: {err:?}");
                }
            },
        ))
        .plugin(log_plugin)
        .plugin(tauri_plugin_os::init())
        .plugin(tauri_plugin_global_shortcut::Builder::new().build())
        .plugin(tauri_plugin_opener::init())
        .plugin(core::prevent_default::init())
        .invoke_handler(tauri::generate_handler![
            commands::read_clipboard,
            commands::list_clipboard_items,
            commands::list_clipboard_groups,
            commands::get_clipboard_item,
            commands::list_clipboard_apps,
            commands::list_all_apps,
            commands::refresh_apps,
            commands::get_clipboard_image_path,
            commands::get_clipboard_app_icon_path,
            commands::get_file_icon_path,
            commands::write_to_clipboard,
            commands::paste_clipboard_item,
            commands::toggle_clipboard_item_favorite,
            commands::delete_clipboard_item,
            commands::update_clipboard_item_note,
            commands::show_window,
            commands::hide_window,
            commands::toggle_window,
            commands::show_taskbar_icon,
            commands::position_window,
            commands::save_window_state,
            commands::restore_window_state,
            commands::get_settings,
            commands::update_settings,
            commands::get_autostart,
            commands::set_autostart,
            commands::is_launched_via_autostart,
        ])
        .setup(move |app| {
            let handle = app.handle().clone();

            // macOS：plugin 必须在 to_panel 前注册。
            #[cfg(target_os = "macos")]
            window::macos::register_plugin(&handle);

            let window_state_store = window::WindowStateStore::new(&handle).map_err(|err| {
                log::error!("window state store initialization failed: {err:?}");
                err
            })?;
            handle.manage(window_state_store);

            let settings = settings::init(&handle).map_err(|err| {
                log::error!("settings initialization failed: {err:?}");
                err
            })?;

            let handle_db = handle.clone();
            tauri::async_runtime::block_on(async move {
                let pool = db::init(&handle_db).await.map_err(|err| {
                    log::error!("database initialization failed: {err:?}");
                    err
                })?;
                handle_db.manage(pool.clone());
                clipboard::init(&handle_db, pool)?;
                Ok::<_, anyhow::Error>(())
            })?;

            shortcut::init(&handle, &settings.shortcuts).map_err(|err| {
                log::error!("global shortcut initialization failed: {err:?}");
                err
            })?;

            autostart::init(&handle).map_err(|err| {
                log::error!("autostart initialization failed: {err:?}");
                err
            })?;

            if let Err(err) = tray::init(&handle, &settings) {
                log::error!("tray initialization failed: {err:?}");
            }

            // 平台主窗口初始化：macOS 转 NSPanel；Windows 改 focusable=false。
            #[cfg(target_os = "macos")]
            if let Err(err) = window::macos::setup_main(&handle) {
                log::error!("setup main NSPanel failed: {err:?}");
            }
            #[cfg(target_os = "windows")]
            if let Err(err) = window::windows::setup_main(&handle) {
                log::error!("setup main (windows) failed: {err:?}");
            }

            Ok(())
        })
        .on_window_event(|window, event| {
            if let WindowEvent::CloseRequested { api, .. } = event {
                if window::hide_on_close(window) {
                    api.prevent_close();
                }
            }
        })
        .build(tauri::generate_context!())
        .expect("error while running tauri application")
        .run(|app_handle, event| {
            #[cfg(target_os = "macos")]
            if let tauri::RunEvent::Reopen {
                has_visible_windows,
                ..
            } = event
            {
                window::handle_reopen(app_handle, has_visible_windows);
            }

            let _ = (app_handle, event);
        });
}

mod autostart;
mod backup;
mod clipboard;
mod commands;
mod core;
mod db;
mod drag_out;
mod i18n;
#[cfg(target_os = "windows")]
mod keyboard;
mod keystroke;
mod menu;
#[cfg(target_os = "windows")]
mod mouse;
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
            |app_handle, argv, _cwd| {
                if let Some(path) = backup::backup_path_from_args(&argv) {
                    if let Err(err) = backup::emit_received_backup(
                        app_handle,
                        path,
                        backup::BackupReceiveSource::OpenFile,
                    ) {
                        log::error!("receive backup from second instance failed: {err:?}");
                    }
                    return;
                }

                if let Err(err) = window::show_window(app_handle, window::PREFERENCE_WINDOW_LABEL) {
                    log::error!("show preference window on second instance failed: {err:?}");
                }
            },
        ))
        .plugin(log_plugin)
        .plugin(tauri_plugin_os::init())
        .plugin(tauri_plugin_global_shortcut::Builder::new().build())
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_dialog::init())
        .plugin(core::prevent_default::init())
        .invoke_handler(tauri::generate_handler![
            commands::read_clipboard,
            commands::list_clipboard_items,
            commands::open_external_url,
            commands::list_clipboard_groups,
            commands::get_clipboard_item,
            commands::list_clipboard_apps,
            commands::list_all_apps,
            commands::add_clipboard_app_from_path,
            commands::delete_unreferenced_clipboard_apps,
            commands::get_clipboard_preview_payload,
            commands::play_copy_sound,
            commands::get_clipboard_image_path,
            commands::get_clipboard_app_icon_path,
            commands::get_file_icon_path,
            commands::write_to_clipboard,
            commands::paste_clipboard_item,
            commands::start_drag_clipboard_item,
            commands::toggle_clipboard_item_favorite,
            commands::toggle_clipboard_item_pinned,
            commands::delete_clipboard_item,
            commands::clear_clipboard_items,
            commands::update_clipboard_item_note,
            commands::open_clipboard_item_link,
            commands::reveal_clipboard_item,
            commands::show_window,
            commands::hide_window,
            commands::toggle_window,
            commands::show_taskbar_icon,
            commands::position_window,
            commands::set_main_window_pinned,
            commands::show_clipboard_preview,
            commands::close_clipboard_preview,
            commands::get_clipboard_preview_state,
            commands::get_settings,
            commands::suspend_global_shortcuts,
            commands::resume_global_shortcuts,
            commands::update_settings,
            commands::reset_settings,
            commands::export_history_backup,
            commands::inspect_history_backup,
            commands::import_history_backup,
            commands::get_storage_usage,
            commands::get_storage_location,
            commands::change_storage_location,
            commands::reset_storage_location,
            commands::clean_resource_cache,
            commands::open_preference_directory,
            commands::get_autostart,
            commands::set_autostart,
            commands::is_launched_via_autostart,
            menu::clipboard_item::popup_clipboard_item_menu,
        ])
        .on_menu_event(|app, event| {
            menu::clipboard_item::handle_menu_event(app, event.id().as_ref());
        })
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
                handle_db.manage(db::DatabaseState::new(pool));
                clipboard::init(&handle_db)?;
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

            // 平台主窗口初始化：macOS 转 NSPanel
            #[cfg(target_os = "macos")]
            if let Err(err) = window::macos::setup_main(&handle) {
                log::error!("setup main NSPanel failed: {err:?}");
            }

            menu::clipboard_item::init(&handle);

            #[cfg(target_os = "windows")]
            menu::context_window::init(&handle);

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

            #[cfg(target_os = "macos")]
            if let tauri::RunEvent::Opened { urls } = &event {
                for url in urls {
                    if url.scheme() != "file" {
                        continue;
                    }
                    let Ok(path) = url.to_file_path() else {
                        continue;
                    };
                    if !backup::is_backup_path(&path) {
                        continue;
                    }
                    if let Err(err) = backup::emit_received_backup(
                        app_handle,
                        path,
                        backup::BackupReceiveSource::OpenFile,
                    ) {
                        log::error!("receive backup from open file failed: {err:?}");
                    }
                    break;
                }
            }

            // 退出前保存所有窗口几何，兜住「调整大小后不关窗直接退出」的场景。
            if let tauri::RunEvent::ExitRequested { .. } = event {
                window::save_all_window_states(app_handle);
            }
        });
}

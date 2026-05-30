mod autostart;
mod clipboard;
mod commands;
mod core;
mod db;
mod keystroke;
mod settings;
mod shortcut;
mod window;

use tauri::{Manager, WindowEvent};
use tauri_awesome_rpc::AwesomeRpc;

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let log_plugin = tauri_plugin_log::Builder::new()
        .level(if cfg!(debug_assertions) {
            log::LevelFilter::Debug
        } else {
            log::LevelFilter::Info
        })
        .targets([
            tauri_plugin_log::Target::new(tauri_plugin_log::TargetKind::Stdout),
            tauri_plugin_log::Target::new(tauri_plugin_log::TargetKind::LogDir { file_name: None }),
            tauri_plugin_log::Target::new(tauri_plugin_log::TargetKind::Webview),
        ])
        .build();

    let allowed_origins = if cfg!(dev) {
        vec!["http://localhost:1420"]
    } else {
        vec!["tauri://localhost"]
    };

    let awesome_rpc = AwesomeRpc::new(allowed_origins)
        .max_payload(64 * 1024 * 1024)
        .max_in_buffer_capacity(64 * 1024 * 1024)
        .max_out_buffer_capacity(64 * 1024 * 1024);

    tauri::Builder::default()
        .plugin(tauri_plugin_single_instance::init(|app_handle, _argv, _cwd| {
            if let Err(err) =
                window::show_window(app_handle, window::PREFERENCE_WINDOW_LABEL)
            {
                log::error!("show preference window on second instance failed: {err:?}");
            }
        }))
        .plugin(log_plugin)
        .plugin(tauri_plugin_global_shortcut::Builder::new().build())
        .invoke_system(awesome_rpc.initialization_script())
        .invoke_handler(tauri::generate_handler![
            commands::read_clipboard,
            commands::get_clipboard_image_path,
            commands::write_to_clipboard,
            commands::paste_clipboard_item,
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
            awesome_rpc.start(app.handle().clone());

            let handle = app.handle().clone();

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

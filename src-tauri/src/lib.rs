mod clipboard;
mod commands;
mod core;
mod db;
mod paste;
mod settings;
mod window;

use tauri::Manager;
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
        .plugin(log_plugin)
        .invoke_system(awesome_rpc.initialization_script())
        .invoke_handler(tauri::generate_handler![
            commands::read_clipboard,
            commands::get_clipboard_image_path,
            commands::show_window,
            commands::hide_window,
            commands::toggle_window,
            commands::show_taskbar_icon,
            commands::position_window,
            commands::save_window_state,
            commands::restore_window_state,
        ])
        .setup(move |app| {
            awesome_rpc.start(app.handle().clone());

            let handle = app.handle().clone();

            let window_state_store = window::WindowStateStore::new(&handle).map_err(|err| {
                log::error!("window state store initialization failed: {err:?}");
                err
            })?;
            handle.manage(window_state_store);

            tauri::async_runtime::block_on(async move {
                let pool = db::init(&handle).await.map_err(|err| {
                    log::error!("database initialization failed: {err:?}");
                    err
                })?;
                handle.manage(pool.clone());
                clipboard::init(&handle, pool)?;
                Ok::<_, anyhow::Error>(())
            })?;

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}

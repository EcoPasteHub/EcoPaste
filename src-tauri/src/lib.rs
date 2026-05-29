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
        .setup(move |app| {
            awesome_rpc.start(app.handle().clone());

            let handle = app.handle().clone();
            tauri::async_runtime::block_on(async move {
                let pool = db::init(&handle).await.map_err(|err| {
                    log::error!("database initialization failed: {err:?}");
                    err
                })?;
                handle.manage(pool);
                Ok::<_, anyhow::Error>(())
            })?;

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}

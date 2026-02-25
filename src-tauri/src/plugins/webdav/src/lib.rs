use tauri::{
    generate_handler,
    plugin::{Builder, TauriPlugin},
    Runtime,
};

mod commands;

pub use commands::*;

pub fn init<R: Runtime>() -> TauriPlugin<R> {
    Builder::new("eco-webdav")
        .invoke_handler(generate_handler![
            commands::set_config,
            commands::get_config,
            commands::get_computer_name,
            commands::test_config,
            commands::list_backups,
            commands::upload_backup,
            commands::cancel_upload,
            commands::download_backup,
            commands::delete_backup,
            commands::create_slim_database
        ])
        .build()
}

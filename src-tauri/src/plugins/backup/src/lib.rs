use tauri::{
    generate_handler,
    plugin::{Builder, TauriPlugin},
    Runtime,
};

mod commands;

pub fn init<R: Runtime>() -> TauriPlugin<R> {
    Builder::new("eco-backup")
        .invoke_handler(generate_handler![
            commands::export_data,
            commands::import_data,
            commands::move_data
        ])
        .build()
}

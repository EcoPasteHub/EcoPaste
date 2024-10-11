use tauri::{
    generate_handler,
    plugin::{Builder, TauriPlugin},
    Runtime,
};

mod commands;

pub use commands::*;

pub fn init<R: Runtime>() -> TauriPlugin<R> {
    Builder::new("eco-fs-extra")
        .invoke_handler(generate_handler![
            commands::metadata,
            commands::preview_path
        ])
        .build()
}

use tauri::{
    generate_handler,
    plugin::{Builder, TauriPlugin},
    Runtime,
};

mod commands;

pub fn init<R: Runtime>() -> TauriPlugin<R> {
    Builder::new("autostart-x")
        .invoke_handler(generate_handler![commands::is_autostart])
        .build()
}

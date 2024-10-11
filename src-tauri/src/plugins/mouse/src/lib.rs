use tauri::{
    generate_handler,
    plugin::{Builder, TauriPlugin},
    Runtime,
};

mod commands;

pub fn init<R: Runtime>() -> TauriPlugin<R> {
    Builder::new("eco-mouse")
        .invoke_handler(generate_handler![commands::get_mouse_coords])
        .build()
}

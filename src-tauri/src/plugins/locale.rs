use tauri::{
    command, generate_handler,
    plugin::{Builder, TauriPlugin},
    Wry,
};

#[command]
pub fn get_locale() -> String {
    return current_locale::current_locale().unwrap();
}

pub fn init() -> TauriPlugin<Wry> {
    Builder::new("locale")
        .invoke_handler(generate_handler![get_locale])
        .build()
}

use crate::{core::tray::Tray, locales::ZH_CN};
use tauri::{
    command, generate_handler,
    plugin::{Builder, TauriPlugin},
    AppHandle, Wry,
};

#[command]
pub fn get_locale() -> String {
    let locale = current_locale::current_locale();

    if locale.is_ok() {
        return locale.ok().unwrap();
    }

    return ZH_CN.to_string();
}

#[command]
pub fn set_locale(app_handle: AppHandle, language: &str) {
    Tray::update_menu(&app_handle, language)
}

pub fn init() -> TauriPlugin<Wry> {
    Builder::new("locale")
        .invoke_handler(generate_handler![get_locale, set_locale])
        .build()
}

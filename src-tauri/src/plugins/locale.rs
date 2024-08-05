use crate::{core::tray::Tray, locales::ZH_CN};
use once_cell::sync::Lazy;
use std::sync::Mutex;
use tauri::{
    command, generate_handler,
    plugin::{Builder, TauriPlugin},
    AppHandle, Wry,
};

pub static LOCALE: Lazy<Mutex<String>> = Lazy::new(|| Mutex::new(ZH_CN.to_string()));

#[command]
pub fn get_locale() -> String {
    let locale = current_locale::current_locale();

    if locale.is_ok() {
        return locale.ok().unwrap();
    }

    return ZH_CN.to_string();
}

#[command]
pub fn set_locale(app_handle: AppHandle<Wry>, language: String) {
    let mut locale = LOCALE.lock().unwrap();

    *locale = language;

    drop(locale);

    Tray::update_menu(&app_handle)
}

pub fn init() -> TauriPlugin<Wry> {
    Builder::new("locale")
        .invoke_handler(generate_handler![get_locale, set_locale])
        .build()
}

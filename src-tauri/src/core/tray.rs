use crate::{
    locales::{get_locale, EN_US, JA_JP, LANGUAGES, ZH_CN, ZH_TW},
    plugins::window::{show_window, PREFERENCE_WINDOW_LABEL},
};
use tauri::{
    async_runtime, AppHandle, CustomMenuItem, Manager, SystemTrayEvent, SystemTrayMenu,
    SystemTrayMenuItem, SystemTraySubmenu,
};

pub struct Tray {}

impl Tray {
    pub fn menu(app_handle: &AppHandle, language: &str) -> SystemTrayMenu {
        let locale = get_locale(language);
        let app_version = app_handle.package_info().version.to_string();

        SystemTrayMenu::new()
            .add_item(CustomMenuItem::new("preference", locale.preference))
            .add_native_item(SystemTrayMenuItem::Separator)
            .add_submenu(SystemTraySubmenu::new(
                locale.language,
                SystemTrayMenu::new()
                    .add_item(CustomMenuItem::new(ZH_CN, "简体中文"))
                    .add_item(CustomMenuItem::new(ZH_TW, "繁體中文"))
                    .add_item(CustomMenuItem::new(EN_US, "English"))
                    .add_item(CustomMenuItem::new(JA_JP, "日本語")),
            ))
            .add_native_item(SystemTrayMenuItem::Separator)
            .add_item(CustomMenuItem::new("about", locale.about))
            .add_item(CustomMenuItem::new("update", locale.update))
            .add_item(CustomMenuItem::new("github", locale.github))
            .add_native_item(SystemTrayMenuItem::Separator)
            .add_item(
                CustomMenuItem::new("version", format!("{} {app_version}", locale.version))
                    .disabled(),
            )
            .add_item(CustomMenuItem::new("exit", locale.exit))
    }

    pub fn handler(app: &AppHandle, event: SystemTrayEvent) {
        async_runtime::block_on(async {
            let window = app.get_window(PREFERENCE_WINDOW_LABEL).unwrap();

            let about_event = || {
                window.emit("about", true).unwrap();
            };

            match event {
                SystemTrayEvent::LeftClick { .. } => window.emit("tray-click", true).unwrap(),
                SystemTrayEvent::MenuItemClick { id, .. } => match id.as_str() {
                    "preference" => show_window(window).await,
                    id if LANGUAGES.contains(&id) => window.emit("change-language", id).unwrap(),
                    "about" => about_event(),
                    "update" => {
                        about_event();
                        window.emit("update", true).unwrap();
                    }
                    "github" => {
                        window.emit("github", true).unwrap();
                    }
                    "exit" => app.exit(0),
                    _ => {}
                },
                _ => {}
            }
        })
    }

    pub fn update_menu(app_handle: &AppHandle, language: &str) {
        let tray = app_handle.tray_handle();

        tray.set_menu(Tray::menu(app_handle, language)).unwrap();

        Tray::update_item_selected(app_handle, language);
    }

    pub fn update_item_selected(app_handle: &AppHandle, language: &str) {
        let tray = app_handle.tray_handle();

        for &item in LANGUAGES.iter() {
            tray.get_item(item).set_selected(item == language).unwrap();
        }
    }
}

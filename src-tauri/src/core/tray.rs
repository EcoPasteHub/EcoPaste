use crate::{
    locales::{get_locale, EN_US, JA_JP, LANGUAGES, ZH_CN, ZH_TW},
    plugins::{
        clipboard::IS_LISTENING,
        locale::LOCALE,
        window::{show_window, PREFERENCE_WINDOW_LABEL},
    },
};
use tauri::{
    async_runtime, AppHandle, CustomMenuItem, Manager, SystemTrayEvent, SystemTrayMenu,
    SystemTrayMenuItem, SystemTraySubmenu,
};

pub struct Tray {}

impl Tray {
    pub fn menu(app_handle: &AppHandle, language: &str, is_listening: bool) -> SystemTrayMenu {
        let locale = get_locale(language);
        let app_version = app_handle.package_info().version.to_string();

        SystemTrayMenu::new()
            .add_item(CustomMenuItem::new("preference", locale.preference))
            .add_item(CustomMenuItem::new(
                "toggle-listening",
                if is_listening {
                    locale.stop_listening
                } else {
                    locale.start_listening
                },
            ))
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

    pub fn handler(app_handle: &AppHandle, event: SystemTrayEvent) {
        async_runtime::block_on(async {
            let window = app_handle.get_window(PREFERENCE_WINDOW_LABEL).unwrap();

            let about_event = || {
                window.emit("about", true).unwrap();
            };

            match event {
                SystemTrayEvent::LeftClick { .. } => {
                    #[cfg(target_os = "windows")]
                    {
                        use crate::plugins::window::MAIN_WINDOW_LABEL;

                        let window = app_handle.get_window(MAIN_WINDOW_LABEL).unwrap();

                        show_window(window).await;
                    }
                }
                SystemTrayEvent::MenuItemClick { id, .. } => match id.as_str() {
                    "preference" => show_window(window).await,
                    "toggle-listening" => {
                        let is_listening = *IS_LISTENING.lock().unwrap();

                        window.emit_all(&id, !is_listening).unwrap();
                    }
                    id if LANGUAGES.contains(&id) => window.emit("change-language", id).unwrap(),
                    "about" => about_event(),
                    "update" => {
                        about_event();
                        window.emit("update", true).unwrap();
                    }
                    "github" => {
                        window.emit("github", true).unwrap();
                    }
                    "exit" => app_handle.exit(0),
                    _ => {}
                },
                _ => {}
            }
        })
    }

    pub fn update_menu(app_handle: &AppHandle) {
        let language = {
            let locale = LOCALE.lock().unwrap();

            locale.clone().unwrap_or_else(|| ZH_CN.to_string())
        };

        let is_listening = *IS_LISTENING.lock().unwrap();

        let tray = app_handle.tray_handle();

        tray.set_menu(Tray::menu(app_handle, &language, is_listening))
            .unwrap();

        Tray::update_item_selected(app_handle, &language);
    }

    pub fn update_item_selected(app_handle: &AppHandle, language: &str) {
        let tray = app_handle.tray_handle();

        for &item in LANGUAGES.iter() {
            tray.get_item(item).set_selected(item == language).unwrap();
        }
    }
}

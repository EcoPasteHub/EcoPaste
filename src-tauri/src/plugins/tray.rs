use super::{
    clipboard::IS_LISTENING,
    locale::LOCALE,
    window::{show_preference_window, PREFERENCE_WINDOW_LABEL},
};
use crate::locales::{get_locale, EN_US, JA_JP, LANGUAGES, ZH_CN, ZH_TW};
use std::sync::Mutex;
use tauri::{
    command, generate_handler,
    plugin::{Builder, TauriPlugin},
    AppHandle, CustomMenuItem, Manager, SystemTray, SystemTrayEvent, SystemTrayMenu,
    SystemTrayMenuItem, SystemTraySubmenu, Wry,
};
use tauri_plugin_window_state::{AppHandleExt, StateFlags};

const TRAY_ID: &str = "ECO_PASTE_TRAY";

static IS_VISIBLE: Mutex<bool> = Mutex::new(false);

pub fn tray_menu(app_handle: &AppHandle) -> SystemTrayMenu {
    let language = LOCALE.lock().unwrap().clone().unwrap_or(ZH_CN.to_string());
    let locale = get_locale(&language);

    let is_listening = *IS_LISTENING.lock().unwrap();

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
            CustomMenuItem::new("version", format!("{} {app_version}", locale.version)).disabled(),
        )
        .add_item(CustomMenuItem::new("exit", locale.exit))
}

pub fn handle_tray_event(app_handle: AppHandle, system_tray: SystemTray) -> SystemTray {
    let window = app_handle.get_window(PREFERENCE_WINDOW_LABEL).unwrap();

    system_tray.on_event(move |e| match e {
        SystemTrayEvent::LeftClick { .. } => {
            #[cfg(target_os = "windows")]
            window.emit_all("toggle-main-window-visible", true).unwrap();
        }
        SystemTrayEvent::MenuItemClick { id, .. } => match id.as_str() {
            "preference" => {
                show_preference_window(&app_handle);
            }
            "toggle-listening" => {
                let is_listening = *IS_LISTENING.lock().unwrap();

                window.emit_all(&id, !is_listening).unwrap();
            }
            id if LANGUAGES.contains(&id) => window.emit("change-language", id).unwrap(),
            "about" => window.emit("about", true).unwrap(),
            "update" => {
                window.emit("about", true).unwrap();

                window.emit("update-app", true).unwrap();
            }
            "github" => {
                window.emit("github", true).unwrap();
            }
            "exit" => {
                app_handle.save_window_state(StateFlags::all()).unwrap();

                app_handle.exit(0)
            }
            _ => {}
        },
        _ => {}
    })
}

pub fn update_tray_menu(app_handle: &AppHandle) {
    let is_visible = IS_VISIBLE.lock().unwrap();

    if !*is_visible {
        return;
    }

    let language = LOCALE.lock().unwrap().clone().unwrap_or(ZH_CN.to_string());

    if let Some(tray) = app_handle.tray_handle_by_id(TRAY_ID) {
        tray.set_menu(tray_menu(app_handle)).unwrap();

        for &item in LANGUAGES.iter() {
            tray.get_item(item).set_selected(item == language).unwrap();
        }
    }
}

pub fn init_tray(app_handle: &AppHandle) {
    let tray_menu = tray_menu(app_handle);

    let package_info = app_handle.package_info();
    let app_name = &package_info.name;
    let app_version = &package_info.version;
    let tooltip = format!("{app_name} v{app_version}");

    let mut tray = SystemTray::new()
        .with_id(TRAY_ID)
        .with_menu(tray_menu)
        .with_tooltip(&tooltip);

    tray = handle_tray_event(app_handle.clone(), tray);

    tray.build(&app_handle.clone()).unwrap();

    update_tray_menu(app_handle);
}

pub fn destroy_tray(app_handle: &AppHandle) {
    if let Some(tray) = app_handle.tray_handle_by_id(TRAY_ID) {
        tray.destroy().unwrap();
    }
}

#[command]
pub async fn set_tray_visible(app_handle: AppHandle, visible: bool) {
    if visible {
        init_tray(&app_handle)
    } else {
        destroy_tray(&app_handle);
    }

    let mut is_visible = IS_VISIBLE.lock().unwrap();

    *is_visible = visible;
}

pub fn init() -> TauriPlugin<Wry> {
    Builder::new("tray")
        .invoke_handler(generate_handler![set_tray_visible])
        .build()
}

use tauri::{
    async_runtime, AppHandle, CustomMenuItem, Manager, SystemTray, SystemTrayEvent, SystemTrayMenu,
    SystemTrayMenuItem, SystemTraySubmenu,
};

use crate::plugins::window::{show_window, PREFERENCE_WINDOW_LABEL};

pub fn menu() -> SystemTray {
    let tray_menu = SystemTrayMenu::new()
        .add_item(CustomMenuItem::new("preference".to_string(), "偏好设置"))
        .add_native_item(SystemTrayMenuItem::Separator)
        .add_submenu(SystemTraySubmenu::new(
            "更多",
            SystemTrayMenu::new()
                .add_item(CustomMenuItem::new("about".to_string(), "关于"))
                .add_item(CustomMenuItem::new("update".to_string(), "检查更新"))
                .add_native_item(SystemTrayMenuItem::Separator)
                .add_item(CustomMenuItem::new("github".to_string(), "开源地址")),
        ))
        .add_native_item(SystemTrayMenuItem::Separator)
        .add_item(CustomMenuItem::new("exit".to_string(), "退出"));

    SystemTray::new().with_menu(tray_menu)
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

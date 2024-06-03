use crate::window::{quit_app, show_window, MAIN_WINDOW_LABEL};
use tauri::{
    AppHandle, CustomMenuItem, Manager, SystemTray, SystemTrayEvent, SystemTrayMenu,
    SystemTrayMenuItem, SystemTraySubmenu,
};

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
        .add_item(CustomMenuItem::new("quit".to_string(), "退出"));

    SystemTray::new().with_menu(tray_menu)
}

pub fn handler(app: &AppHandle, event: SystemTrayEvent) {
    let window = app.get_window(MAIN_WINDOW_LABEL).unwrap();

    match event {
        SystemTrayEvent::LeftClick { .. } => show_window(window),
        SystemTrayEvent::MenuItemClick { id, .. } => match id.as_str() {
            "preference" => show_window(window),
            "about" => {
                show_window(window.clone());
                window.emit("about", true).unwrap();
            }
            "update" => {
                println!("检查更新")
            }
            "github" => {
                println!("打开 Github 地址")
            }
            "quit" => quit_app(),
            _ => {}
        },
        _ => {}
    }
}

use tauri::{
    generate_handler,
    plugin::{Builder, TauriPlugin},
    Runtime,
};

mod commands;

pub fn init<R: Runtime>() -> TauriPlugin<R> {
    Builder::new("eco-macos-permissions")
        .invoke_handler(generate_handler![
            commands::check_accessibility_permissions,
            commands::request_accessibility_permissions,
            commands::request_full_disk_access_permissions
        ])
        .build()
}

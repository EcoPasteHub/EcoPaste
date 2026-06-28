use tauri::AppHandle;

use crate::autostart;
use crate::core::Result;

#[tauri::command]
pub async fn get_autostart(app: AppHandle) -> Result<bool> {
    autostart::is_enabled(&app)
}

#[tauri::command]
pub async fn set_autostart(app: AppHandle, enabled: bool) -> Result<()> {
    autostart::set_enabled(&app, enabled)
}

#[tauri::command]
pub async fn is_launched_via_autostart() -> bool {
    autostart::launched_via_autostart()
}

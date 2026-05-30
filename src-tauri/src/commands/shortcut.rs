use tauri::AppHandle;

use crate::core::Result;
use crate::shortcut::{self, ShortcutBindings};

#[tauri::command]
pub async fn get_shortcuts(app: AppHandle) -> Result<ShortcutBindings> {
    Ok(shortcut::current_bindings(&app))
}

#[tauri::command]
pub async fn update_shortcuts(app: AppHandle, bindings: ShortcutBindings) -> Result<()> {
    shortcut::apply(&app, bindings)
}

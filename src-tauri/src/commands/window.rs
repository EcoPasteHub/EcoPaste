use tauri::AppHandle;

use crate::core::Result;
use crate::window::{self, WindowPosition, WindowStyle};

#[tauri::command]
pub async fn show_window(app: AppHandle, label: String) -> Result<()> {
    window::show_window(&app, &label)
}

#[tauri::command]
pub async fn hide_window(app: AppHandle, label: String) -> Result<()> {
    window::hide_window(&app, &label)
}

#[tauri::command]
pub async fn toggle_window(app: AppHandle, label: String) -> Result<()> {
    window::toggle_window(&app, &label)
}

#[tauri::command]
pub async fn show_taskbar_icon(app: AppHandle, visible: bool) -> Result<()> {
    window::show_taskbar_icon(&app, visible)
}

#[tauri::command]
pub async fn position_window(
    app: AppHandle,
    label: String,
    style: WindowStyle,
    position: WindowPosition,
) -> Result<()> {
    window::position_window(&app, &label, style, position)
}

#[tauri::command]
pub async fn save_window_state(app: AppHandle, label: String) -> Result<()> {
    window::save_window_state(&app, &label)
}

#[tauri::command]
pub async fn restore_window_state(app: AppHandle, label: String) -> Result<bool> {
    window::restore_window_state(&app, &label)
}

mod position;
mod state;

pub use position::{WindowPosition, WindowStyle};
pub use state::WindowStateStore;

use tauri::{AppHandle, Manager, WebviewWindow};

use crate::core::Result;

pub const MAIN_WINDOW_LABEL: &str = "main";
pub const PREFERENCE_WINDOW_LABEL: &str = "preference";

fn get_window(app_handle: &AppHandle, label: &str) -> Result<WebviewWindow> {
    app_handle
        .get_webview_window(label)
        .ok_or_else(|| anyhow::anyhow!("window not found: {label}").into())
}

pub fn show_window(app_handle: &AppHandle, label: &str) -> Result<()> {
    let window = get_window(app_handle, label)?;
    window.show().map_err(|e| anyhow::anyhow!(e))?;
    window.unminimize().map_err(|e| anyhow::anyhow!(e))?;
    window.set_focus().map_err(|e| anyhow::anyhow!(e))?;
    Ok(())
}

pub fn hide_window(app_handle: &AppHandle, label: &str) -> Result<()> {
    let window = get_window(app_handle, label)?;
    window.hide().map_err(|e| anyhow::anyhow!(e))?;
    Ok(())
}

pub fn toggle_window(app_handle: &AppHandle, label: &str) -> Result<()> {
    let window = get_window(app_handle, label)?;
    if window.is_visible().unwrap_or(false) {
        hide_window(app_handle, label)
    } else {
        show_window(app_handle, label)
    }
}

#[cfg(target_os = "macos")]
pub fn show_taskbar_icon(app_handle: &AppHandle, visible: bool) -> Result<()> {
    app_handle
        .set_dock_visibility(visible)
        .map_err(|e| anyhow::anyhow!(e))?;
    Ok(())
}

#[cfg(target_os = "windows")]
pub fn show_taskbar_icon(app_handle: &AppHandle, visible: bool) -> Result<()> {
    let window = get_window(app_handle, MAIN_WINDOW_LABEL)?;
    window
        .set_skip_taskbar(!visible)
        .map_err(|e| anyhow::anyhow!(e))?;
    Ok(())
}

pub fn position_window(
    app_handle: &AppHandle,
    label: &str,
    style: WindowStyle,
    pos: WindowPosition,
) -> Result<()> {
    let window = get_window(app_handle, label)?;
    position::position_window(&window, style, pos)
}

pub fn save_window_state(app_handle: &AppHandle, label: &str) -> Result<()> {
    state::save_window_state(app_handle, label)
}

pub fn restore_window_state(app_handle: &AppHandle, label: &str) -> Result<bool> {
    state::restore_window_state(app_handle, label)
}

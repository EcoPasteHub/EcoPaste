mod position;
mod state;

#[cfg(target_os = "macos")]
pub mod macos;
#[cfg(target_os = "windows")]
pub mod windows;

#[cfg(target_os = "macos")]
pub use macos::handle_reopen;
pub use position::{WindowPosition, WindowStyle};
pub use state::WindowStateStore;

use tauri::{AppHandle, Manager, WebviewWindow, Window};

use crate::core::Result;

pub const MAIN_WINDOW_LABEL: &str = "main";
pub const PREFERENCE_WINDOW_LABEL: &str = "preference";

pub(super) fn get_window(app_handle: &AppHandle, label: &str) -> Result<WebviewWindow> {
    app_handle
        .get_webview_window(label)
        .ok_or_else(|| anyhow::anyhow!("window not found: {label}").into())
}

pub fn show_window(app_handle: &AppHandle, label: &str) -> Result<()> {
    #[cfg(target_os = "macos")]
    return macos::show_window(app_handle, label);
    #[cfg(target_os = "windows")]
    return windows::show_window(app_handle, label);
}

pub fn hide_window(app_handle: &AppHandle, label: &str) -> Result<()> {
    #[cfg(target_os = "macos")]
    return macos::hide_window(app_handle, label);
    #[cfg(target_os = "windows")]
    return windows::hide_window(app_handle, label);
}

pub fn toggle_window(app_handle: &AppHandle, label: &str) -> Result<()> {
    let visible = get_window(app_handle, label)?.is_visible().unwrap_or(false);
    if visible {
        hide_window(app_handle, label)
    } else {
        show_window(app_handle, label)
    }
}

pub fn show_taskbar_icon(app_handle: &AppHandle, visible: bool) -> Result<()> {
    #[cfg(target_os = "macos")]
    return macos::show_taskbar_icon(app_handle, visible);
    #[cfg(target_os = "windows")]
    return windows::show_taskbar_icon(app_handle, visible);
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

/// 关闭请求改为隐藏窗口，让应用常驻后台（系统托盘）。
/// 返回 `true` 表示已拦截关闭，调用方需 `api.prevent_close()`。
pub fn hide_on_close(window: &Window) -> bool {
    if let Err(err) = window.hide() {
        log::error!("hide window on close failed: {err:?}");
    }
    true
}

mod position;
mod state;

pub use position::{WindowPosition, WindowStyle};
pub use state::WindowStateStore;

use tauri::{AppHandle, Manager, WebviewWindow, Window};

use crate::core::Result;
use crate::keyboard;

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
    // main 不调 set_focus：Windows 上要避免抢走前台应用的焦点（粘贴目标）。
    // macOS 走 Web keydown 仍需要 webview 是 key window，由前端/平台默认行为承担。
    if label == MAIN_WINDOW_LABEL {
        keyboard::enable_navigation_keys(app_handle);
    } else {
        window.set_focus().map_err(|e| anyhow::anyhow!(e))?;
    }
    Ok(())
}

pub fn hide_window(app_handle: &AppHandle, label: &str) -> Result<()> {
    let window = get_window(app_handle, label)?;
    window.hide().map_err(|e| anyhow::anyhow!(e))?;
    if label == MAIN_WINDOW_LABEL {
        keyboard::disable_navigation_keys();
    }
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

/// 关闭请求改为隐藏窗口，让应用常驻后台（系统托盘）。
/// 返回 `true` 表示已拦截关闭，调用方需 `api.prevent_close()`。
pub fn hide_on_close(window: &Window) -> bool {
    if let Err(err) = window.hide() {
        log::error!("hide window on close failed: {err:?}");
    }
    true
}

/// macOS 点击 dock 图标 reopen 时，无可见窗口则唤起偏好窗口。
#[cfg(target_os = "macos")]
pub fn handle_reopen(app_handle: &AppHandle, has_visible_windows: bool) {
    if has_visible_windows {
        return;
    }
    if let Err(err) = show_window(app_handle, PREFERENCE_WINDOW_LABEL) {
        log::error!("show preference window on reopen failed: {err:?}");
    }
}

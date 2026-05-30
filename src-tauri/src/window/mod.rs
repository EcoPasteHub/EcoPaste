mod position;
mod state;

#[cfg(target_os = "macos")]
pub mod macos;
#[cfg(target_os = "windows")]
pub mod windows;

#[cfg(target_os = "macos")]
pub use macos::handle_reopen;
pub use state::WindowStateStore;

use tauri::{AppHandle, Emitter, Manager, WebviewWindow, Window};

use crate::core::Result;
use crate::settings::{SettingsStore, WindowPosition, WindowStyle};

pub const MAIN_WINDOW_LABEL: &str = "main";
pub const PREFERENCE_WINDOW_LABEL: &str = "preference";

/// 主窗口显隐变化事件。前端用以做默认聚焦 / 自动清空搜索等 UI 副作用。
/// 由 [`show_window`] / [`hide_window`] 在统一入口处发出，平台一致，
/// 不依赖 `tauri://focus` / `tauri://blur`（Windows 主窗口 `focusable: false` 不可靠）。
const WINDOW_VISIBILITY_EVENT: &str = "window://visibility";

#[derive(Clone, serde::Serialize)]
struct WindowVisibilityPayload<'a> {
    label: &'a str,
    visible: bool,
}

fn emit_visibility(app_handle: &AppHandle, label: &str, visible: bool) {
    if let Err(err) = app_handle.emit(
        WINDOW_VISIBILITY_EVENT,
        WindowVisibilityPayload { label, visible },
    ) {
        log::error!("emit window visibility failed: {err:?}");
    }
}

pub(super) fn get_window(app_handle: &AppHandle, label: &str) -> Result<WebviewWindow> {
    app_handle
        .get_webview_window(label)
        .ok_or_else(|| anyhow::anyhow!("window not found: {label}").into())
}

pub fn show_window(app_handle: &AppHandle, label: &str) -> Result<()> {
    if label == MAIN_WINDOW_LABEL {
        if let Err(err) = apply_main_layout(app_handle) {
            log::warn!("apply main window layout failed: {err}");
        }
    }

    #[cfg(target_os = "macos")]
    let result = macos::show_window(app_handle, label);
    #[cfg(target_os = "windows")]
    let result = windows::show_window(app_handle, label);
    if result.is_ok() {
        emit_visibility(app_handle, label, true);
    }
    result
}

pub fn hide_window(app_handle: &AppHandle, label: &str) -> Result<()> {
    if label == MAIN_WINDOW_LABEL {
        if let Some(store) = app_handle.try_state::<SettingsStore>() {
            let snap = store.snapshot();
            if matches!(snap.clipboard.window.style, WindowStyle::Standard)
                && matches!(snap.clipboard.window.position, WindowPosition::Remember)
            {
                if let Err(err) = state::save_window_state(app_handle, label) {
                    log::warn!("save main window state on hide failed: {err}");
                }
            }
        }
    }
    #[cfg(target_os = "macos")]
    let result = macos::hide_window(app_handle, label);
    #[cfg(target_os = "windows")]
    let result = windows::hide_window(app_handle, label);
    if result.is_ok() {
        emit_visibility(app_handle, label, false);
    }
    result
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

/// 主窗显示前按设置应用窗口样式与定位。
/// Standard + Remember 走 `restore_window_state`；其它走 `position_window`。
/// 平台 `show_window` 需要在主线程闭包里调用，避免 set_position 与 show 异步交错产生闪烁。
fn apply_main_layout(app_handle: &AppHandle) -> Result<()> {
    let Some(store) = app_handle.try_state::<SettingsStore>() else {
        return Ok(());
    };
    let snap = store.snapshot();
    let style = snap.clipboard.window.style;
    let position = snap.clipboard.window.position;

    if matches!(style, WindowStyle::Standard) && matches!(position, WindowPosition::Remember) {
        let _ = state::restore_window_state(app_handle, MAIN_WINDOW_LABEL)?;
        return Ok(());
    }

    let window = get_window(app_handle, MAIN_WINDOW_LABEL)?;
    position::position_window(&window, style, position)
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

pub(super) mod position;
pub mod preview;
mod state;

#[cfg(target_os = "macos")]
pub mod macos;
#[cfg(target_os = "windows")]
pub mod windows;

#[cfg(target_os = "macos")]
pub use macos::handle_reopen;
pub use state::WindowStateStore;

use std::sync::atomic::{AtomicBool, Ordering};

use tauri::{AppHandle, Emitter, Manager, WebviewWindow, Window};

use crate::core::Result;
use crate::settings::{SettingsStore, WindowPosition};

pub const MAIN_WINDOW_LABEL: &str = "main";
pub const PREFERENCE_WINDOW_LABEL: &str = "preference";
pub const CLIPBOARD_PREVIEW_WINDOW_LABEL: &str = "clipboard-preview";

/// 主窗口「固定」状态：true 时失焦不自动隐藏（点击窗外、切到其它 App 都不会隐藏），
/// 由前端 Pin 按钮 / 快捷键切换；macOS resign_key 与 Windows 外部点击钩子都尊重这个开关。
static MAIN_WINDOW_PINNED: AtomicBool = AtomicBool::new(false);

pub fn is_main_window_pinned() -> bool {
    MAIN_WINDOW_PINNED.load(Ordering::Relaxed)
}

pub fn set_main_window_pinned(pinned: bool) {
    MAIN_WINDOW_PINNED.store(pinned, Ordering::Relaxed);
}

/// 主窗口显隐变化事件。前端用以做默认聚焦 / 自动清空搜索等 UI 副作用。
/// 由 [`show_window`] / [`hide_window`] 在统一入口处发出，平台一致，
/// 不依赖 `tauri://focus` / `tauri://blur`（Windows 主窗口 `focusable: false` 不可靠）。
const WINDOW_VISIBILITY_EVENT: &str = "window://visibility";

#[derive(Clone, serde::Serialize)]
struct WindowVisibilityPayload<'a> {
    label: &'a str,
    visible: bool,
}

pub(super) fn emit_visibility(app_handle: &AppHandle, label: &str, visible: bool) {
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
    } else {
        let visible = get_window(app_handle, label)?.is_visible().unwrap_or(false);

        if !visible {
            // 次级窗口（如 preference）：只在从隐藏态打开时恢复位置 + 尺寸。
            // 已可见窗口可能刚被用户移动但尚未落盘，重复恢复会把窗口拉回旧位置。
            if let Err(err) = state::restore_window_state(app_handle, label) {
                log::warn!("restore window state failed for {label}: {err}");
            }
        }
    }

    #[cfg(target_os = "macos")]
    let result = macos::show_window(app_handle, label);
    #[cfg(target_os = "windows")]
    let result = windows::show_window(app_handle, label);
    if result.is_ok() && !delays_main_visibility_event(label) {
        if label == MAIN_WINDOW_LABEL {
            preview::resume_after_main_show();
        }
        emit_visibility(app_handle, label, true);
    }
    result
}

/// macOS 主窗口有延迟 show，visibility 需等 NSPanel 真的显示后再 emit。
fn delays_main_visibility_event(label: &str) -> bool {
    cfg!(target_os = "macos") && label == MAIN_WINDOW_LABEL
}

pub fn hide_window(app_handle: &AppHandle, label: &str) -> Result<()> {
    // 隐藏前保存任意窗口的实时几何：移动与缩放都在这里落盘，下次显示/启动可恢复。
    if let Err(err) = state::save_window_state(app_handle, label) {
        log::warn!("save window state on hide failed for {label}: {err}");
    }

    if label == MAIN_WINDOW_LABEL {
        preview::suppress_for_main_hide(app_handle);
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

pub fn position_window(app_handle: &AppHandle, label: &str, pos: WindowPosition) -> Result<()> {
    let window = get_window(app_handle, label)?;
    position::position_window(&window, pos)
}

/// 主窗显示前按设置应用窗口定位策略。
/// 始终先调用 `restore_window_state` 恢复尺寸与合法位置（含越界 fallback）；
/// 非 Remember 策略再由 `position_window` 覆盖位置。
/// 平台 `show_window` 需要在主线程闭包里调用，避免 set_position 与 show 异步交错产生闪烁。
fn apply_main_layout(app_handle: &AppHandle) -> Result<()> {
    let Some(store) = app_handle.try_state::<SettingsStore>() else {
        return Ok(());
    };
    let snap = store.snapshot();
    let position = snap.clipboard.window.position;

    let _ = state::restore_window_state(app_handle, MAIN_WINDOW_LABEL)?;

    if matches!(position, WindowPosition::Remember) {
        return Ok(());
    }

    let window = get_window(app_handle, MAIN_WINDOW_LABEL)?;
    position::position_window(&window, position)
}

/// 保存当前所有窗口的几何信息。供应用退出（`RunEvent::ExitRequested`）时调用，
/// 覆盖「调整大小后不关窗直接退出」这一隐藏/关闭都漏掉的场景。
pub fn save_all_window_states(app_handle: &AppHandle) {
    for label in app_handle.webview_windows().into_keys() {
        if let Err(err) = state::save_window_state(app_handle, &label) {
            log::warn!("save window state on exit failed for {label}: {err}");
        }
    }
}

/// 关闭请求改为隐藏窗口，让应用常驻后台（系统托盘）。
/// 返回 `true` 表示已拦截关闭，调用方需 `api.prevent_close()`。
pub fn hide_on_close(window: &Window) -> bool {
    // 关闭按钮不走 `hide_window`，需在此单独保存几何，否则 preference 的移动/缩放会丢失。
    if let Err(err) = state::save_window_state(window.app_handle(), window.label()) {
        log::warn!(
            "save window state on close failed for {}: {err}",
            window.label()
        );
    }

    if let Err(err) = window.hide() {
        log::error!("hide window on close failed: {err:?}");
    } else {
        emit_visibility(window.app_handle(), window.label(), false);
    }
    true
}

//! Windows 窗口管理：主窗口运行时改 focusable=false，show 不抢前台焦点；
//! 导航键由 `WH_KEYBOARD_LL` 钩子（`keyboard::enable_navigation_keys`）捕获后 emit。

use tauri::AppHandle;

use super::{get_window, MAIN_WINDOW_LABEL};
use crate::core::Result;
use crate::keyboard;

/// setup 末尾调用：主窗口运行时不可聚焦，show 不抢前台。
pub fn setup_main(app_handle: &AppHandle) -> Result<()> {
    let main = get_window(app_handle, MAIN_WINDOW_LABEL)?;
    main.set_focusable(false).map_err(|e| anyhow::anyhow!(e))?;
    Ok(())
}

pub fn show_window(app_handle: &AppHandle, label: &str) -> Result<()> {
    let window = get_window(app_handle, label)?;
    window.show().map_err(|e| anyhow::anyhow!(e))?;
    window.unminimize().map_err(|e| anyhow::anyhow!(e))?;
    // main 不调 set_focus：focusable=false 下也会失败/无效。导航键走 OS 钩子。
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

pub fn show_taskbar_icon(app_handle: &AppHandle, visible: bool) -> Result<()> {
    let window = get_window(app_handle, MAIN_WINDOW_LABEL)?;
    window
        .set_skip_taskbar(!visible)
        .map_err(|e| anyhow::anyhow!(e))?;
    Ok(())
}

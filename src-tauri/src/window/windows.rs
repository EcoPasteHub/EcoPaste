//! Windows 窗口管理：剪贴板窗口默认不可聚焦，输入控件编辑期间临时恢复可聚焦。

use std::sync::Mutex;

use tauri::AppHandle;
use windows::Win32::Foundation::HWND;
use windows::Win32::UI::WindowsAndMessaging::{GetForegroundWindow, IsWindow, SetForegroundWindow};

use super::{get_window, CLIPBOARD_WINDOW_LABEL};
use crate::core::Result;
use crate::{keyboard, mouse};

static PRE_EDIT_FOREGROUND_HWND: Mutex<Option<isize>> = Mutex::new(None);

pub fn show_window(app_handle: &AppHandle, label: &str) -> Result<()> {
    let window = get_window(app_handle, label)?;
    if label == CLIPBOARD_WINDOW_LABEL {
        window
            .set_focusable(false)
            .map_err(|e| anyhow::anyhow!(e))?;
        clear_pre_edit_foreground();
    }

    window.show().map_err(|e| anyhow::anyhow!(e))?;
    window.unminimize().map_err(|e| anyhow::anyhow!(e))?;

    if label == CLIPBOARD_WINDOW_LABEL {
        keyboard::enable_navigation_keys(app_handle);
        mouse::enable_outside_click_hide(app_handle);
    } else {
        window.set_focus().map_err(|e| anyhow::anyhow!(e))?;
    }

    Ok(())
}

pub fn set_clipboard_window_editing(app_handle: &AppHandle, editing: bool) -> Result<()> {
    let window = get_window(app_handle, CLIPBOARD_WINDOW_LABEL)?;
    let raw_hwnd = window.hwnd().map_err(|e| anyhow::anyhow!(e))?;
    let hwnd = HWND(raw_hwnd.0 as isize);

    if editing {
        remember_pre_edit_foreground(hwnd);
        keyboard::disable_navigation_keys();
        window.set_focusable(true).map_err(|e| anyhow::anyhow!(e))?;
        window.set_focus().map_err(|e| anyhow::anyhow!(e))?;

        return Ok(());
    }

    let should_restore_foreground = unsafe { GetForegroundWindow() == hwnd };
    window
        .set_focusable(false)
        .map_err(|e| anyhow::anyhow!(e))?;

    if window.is_visible().unwrap_or(false) {
        keyboard::enable_navigation_keys(app_handle);
        mouse::enable_outside_click_hide(app_handle);
    }

    if should_restore_foreground {
        restore_pre_edit_foreground(hwnd);
    } else {
        clear_pre_edit_foreground();
    }

    Ok(())
}

pub fn hide_window(app_handle: &AppHandle, label: &str) -> Result<()> {
    let window = get_window(app_handle, label)?;
    window.hide().map_err(|e| anyhow::anyhow!(e))?;
    if label == CLIPBOARD_WINDOW_LABEL {
        if let Err(err) = window.set_focusable(false) {
            log::warn!("reset clipboard window focusable on hide failed: {err:?}");
        }
        clear_pre_edit_foreground();
        keyboard::disable_navigation_keys();
        mouse::disable_outside_click_hide();
        crate::menu::context_window::hide(app_handle);
    }

    Ok(())
}

fn remember_pre_edit_foreground(clipboard_hwnd: HWND) {
    let mut guard = PRE_EDIT_FOREGROUND_HWND
        .lock()
        .expect("pre edit foreground hwnd poisoned");
    if guard.is_some() {
        return;
    }

    let foreground = unsafe { GetForegroundWindow() };
    if foreground.0 == 0 || foreground == clipboard_hwnd {
        return;
    }

    *guard = Some(foreground.0);
}

fn restore_pre_edit_foreground(clipboard_hwnd: HWND) {
    let previous = PRE_EDIT_FOREGROUND_HWND
        .lock()
        .expect("pre edit foreground hwnd poisoned")
        .take();
    let Some(previous) = previous else {
        return;
    };

    let previous_hwnd = HWND(previous);
    if previous_hwnd == clipboard_hwnd || !unsafe { IsWindow(previous_hwnd).as_bool() } {
        return;
    }

    if !unsafe { SetForegroundWindow(previous_hwnd).as_bool() } {
        log::debug!("restore pre-edit foreground window was rejected by Windows");
    }
}

fn clear_pre_edit_foreground() {
    PRE_EDIT_FOREGROUND_HWND
        .lock()
        .expect("pre edit foreground hwnd poisoned")
        .take();
}

pub fn show_taskbar_icon(app_handle: &AppHandle, visible: bool) -> Result<()> {
    let window = get_window(app_handle, CLIPBOARD_WINDOW_LABEL)?;
    window
        .set_skip_taskbar(!visible)
        .map_err(|e| anyhow::anyhow!(e))?;
    Ok(())
}

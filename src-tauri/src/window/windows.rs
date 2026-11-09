//! Windows 窗口管理:剪贴板窗口默认不可聚焦,输入控件编辑期间临时恢复可聚焦。

use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Mutex;

use tauri::AppHandle;
use windows::Win32::Foundation::HWND;
use windows::Win32::UI::WindowsAndMessaging::{GetForegroundWindow, IsWindow, SetForegroundWindow};

use super::{get_window, CLIPBOARD_WINDOW_LABEL};
use crate::core::Result;
use crate::{keyboard, mouse, uia};

static PRE_EDIT_FOREGROUND_HWND: Mutex<Option<isize>> = Mutex::new(None);
/// 粘贴回填期间为 true:恢复前台 + UIA 聚焦 + 清理由 paste_clipboard_item 串行掌控,
/// 前端 visibilitychange 触发的 editing(false) 异步路径不得抢跑。
static PASTE_IN_PROGRESS: AtomicBool = AtomicBool::new(false);

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

    // 粘贴流程接管期间,恢复与清理由 paste_clipboard_item 全权负责,这里只做中性复位。
    if PASTE_IN_PROGRESS.load(Ordering::Acquire) {
        window
            .set_focusable(false)
            .map_err(|e| anyhow::anyhow!(e))?;

        if window.is_visible().unwrap_or(false) {
            keyboard::enable_navigation_keys(app_handle);
            mouse::enable_outside_click_hide(app_handle);
        }

        return Ok(());
    }

    let should_restore_foreground = pre_edit_should_restore(hwnd);

    window
        .set_focusable(false)
        .map_err(|e| anyhow::anyhow!(e))?;

    if window.is_visible().unwrap_or(false) {
        keyboard::enable_navigation_keys(app_handle);
        mouse::enable_outside_click_hide(app_handle);
    }

    if should_restore_foreground {
        restore_pre_edit_foreground();
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
        // pre-edit 记录保留:普通隐藏(Esc / 外部点击)后 editing(false) 仍要走
        // 「恢复前台 + UIA 聚焦」路径;粘贴流程则由 paste_clipboard_item 显式消费。
        keyboard::disable_navigation_keys();
        mouse::disable_outside_click_hide();
        crate::menu::context_window::hide(app_handle);
    }

    Ok(())
}

pub fn begin_paste_restore() {
    PASTE_IN_PROGRESS.store(true, Ordering::Release);
}

pub fn end_paste_restore() {
    PASTE_IN_PROGRESS.store(false, Ordering::Release);
    clear_pre_edit_foreground();
}

/// 粘贴回填专用:显式恢复 pre-edit 前台并唤醒目标输入框焦点,再由调用方模拟粘贴。
pub fn restore_pre_edit_for_paste() {
    let Some(previous) = take_pre_edit_foreground() else {
        return;
    };

    // 目标应用失焦后 web 内部光标会丢失,SetForegroundWindow 只恢复顶层前台,
    // 这里必须在模拟 Ctrl+V 之前补上元素级 UIA 聚焦。
    revive_pre_edit_target(previous);
}

/// 是否应向前一个应用回切前台:剪贴板窗口仍在前台,或系统已自动切回记忆窗口
/// (窗口隐藏后后者是常态);用户已经切换到其它应用时不打扰。
fn pre_edit_should_restore(clipboard_hwnd: HWND) -> bool {
    let Some(previous) = peek_pre_edit_foreground() else {
        return false;
    };

    let foreground = unsafe { GetForegroundWindow() };

    foreground == clipboard_hwnd || foreground.0 == previous
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
    drop(guard);

    // 此刻前台尚未被剪贴板窗口夺走,捕获的聚焦元素即目标应用的输入框;
    // 抢完前台再捕获就只会拿到剪贴板自己的搜索框。
    uia::remember_focused_element();
}

fn restore_pre_edit_foreground() {
    let Some(previous) = take_pre_edit_foreground() else {
        return;
    };

    revive_pre_edit_target(previous);
}

fn revive_pre_edit_target(previous: isize) {
    let previous_hwnd = HWND(previous);
    if !unsafe { IsWindow(previous_hwnd).as_bool() } {
        uia::dismiss_remembered_element();
        return;
    }

    if !unsafe { SetForegroundWindow(previous_hwnd).as_bool() } {
        log::debug!("restore pre-edit foreground window was rejected by Windows");
    }

    // 仅恢复 OS 前台救不回 Chromium 等 web 应用的内部焦点(实测),必须对抢焦点前
    // 记住的聚焦元素做元素级 SetFocus;失败时降级为原有的「仅恢复前台」行为。
    if !uia::focus_remembered_element() {
        log::debug!("pre-edit UIA focus restore unavailable, degraded to foreground-only restore");
    }
}

/// 读取但不取走 pre-edit 前台记录。
fn peek_pre_edit_foreground() -> Option<isize> {
    PRE_EDIT_FOREGROUND_HWND
        .lock()
        .expect("pre edit foreground hwnd poisoned")
        .as_ref()
        .copied()
}

fn take_pre_edit_foreground() -> Option<isize> {
    PRE_EDIT_FOREGROUND_HWND
        .lock()
        .expect("pre edit foreground hwnd poisoned")
        .take()
}

fn clear_pre_edit_foreground() {
    PRE_EDIT_FOREGROUND_HWND
        .lock()
        .expect("pre edit foreground hwnd poisoned")
        .take();
    uia::dismiss_remembered_element();
}

pub fn show_taskbar_icon(app_handle: &AppHandle, visible: bool) -> Result<()> {
    let window = get_window(app_handle, CLIPBOARD_WINDOW_LABEL)?;
    window
        .set_skip_taskbar(!visible)
        .map_err(|e| anyhow::anyhow!(e))?;
    Ok(())
}

//! Windows 窗口管理：剪贴板窗口默认不可聚焦，输入控件编辑期间临时恢复可聚焦。

use std::sync::{LazyLock, Mutex};
use std::time::{Duration, Instant};

use tauri::AppHandle;
use windows::Win32::Foundation::HWND;
use windows::Win32::UI::WindowsAndMessaging::{
    GetForegroundWindow, GetWindowLongW, GetWindowThreadProcessId, IsWindow, IsWindowVisible,
    SetForegroundWindow, GWL_EXSTYLE, WS_EX_NOACTIVATE,
};

use super::{get_window, CLIPBOARD_WINDOW_LABEL};
use crate::core::Result;
use crate::{keyboard, mouse};

#[path = "windows_paste_target.rs"]
mod paste_target;
use paste_target::{PasteSession, WindowTarget};

static PRE_EDIT_FOREGROUND_HWND: Mutex<Option<WindowTarget>> = Mutex::new(None);
static PASTE_SESSION: LazyLock<Mutex<PasteSession>> =
    LazyLock::new(|| Mutex::new(PasteSession::default()));
const PASTE_HANDOFF_TIMEOUT: Duration = Duration::from_secs(1);
const PASTE_READINESS_POLL: Duration = Duration::from_millis(10);

pub fn show_window(app_handle: &AppHandle, label: &str) -> Result<()> {
    let window = get_window(app_handle, label)?;
    if label == CLIPBOARD_WINDOW_LABEL {
        let current = external_window(unsafe { GetForegroundWindow() });
        let visible = window.is_visible().map_err(|err| anyhow::anyhow!(err))?;
        PASTE_SESSION
            .lock()
            .unwrap_or_else(|err| err.into_inner())
            .show(current, visible);
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
        PASTE_SESSION
            .lock()
            .unwrap_or_else(|err| err.into_inner())
            .close();
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
    if foreground == clipboard_hwnd {
        return;
    }
    if let Some(target) = external_window(foreground) {
        *guard = Some(target);
        PASTE_SESSION
            .lock()
            .unwrap_or_else(|err| err.into_inner())
            .origin = Some(target);
    }
}

fn restore_pre_edit_foreground(clipboard_hwnd: HWND) {
    let previous = PRE_EDIT_FOREGROUND_HWND
        .lock()
        .expect("pre edit foreground hwnd poisoned")
        .take();
    let Some(previous) = previous else {
        return;
    };

    let previous_hwnd = HWND(previous.hwnd);
    if previous_hwnd == clipboard_hwnd || !is_live_target(previous) {
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

/// Query ownership every time: a nonzero HWND may have been destroyed and reused by another process.
fn window_owner(hwnd: HWND) -> Option<u32> {
    if hwnd.0 == 0 || !unsafe { IsWindow(hwnd).as_bool() } {
        return None;
    }
    let mut process_id = 0;
    let thread_id = unsafe { GetWindowThreadProcessId(hwnd, Some(&mut process_id)) };
    (thread_id != 0 && process_id != 0).then_some(process_id)
}

fn external_window(hwnd: HWND) -> Option<WindowTarget> {
    let target = WindowTarget {
        hwnd: hwnd.0,
        process_id: window_owner(hwnd)?,
    };
    is_live_target(target).then_some(target)
}

fn is_live_target(target: WindowTarget) -> bool {
    paste_target::is_valid_target(target, window_owner(HWND(target.hwnd)), std::process::id())
}

/// Await window-manager execution; successful dispatch alone does not acknowledge focus changes.
async fn on_main_thread<T: Send + 'static>(
    app_handle: &AppHandle,
    operation: impl FnOnce(AppHandle) -> Result<T> + Send + 'static,
) -> Result<T> {
    let (sender, receiver) = tokio::sync::oneshot::channel();
    let handle = app_handle.clone();
    app_handle
        .run_on_main_thread(move || {
            if !sender.is_closed() {
                let _ = sender.send(operation(handle));
            }
        })
        .map_err(|err| anyhow::anyhow!(err))?;
    tokio::time::timeout(PASTE_HANDOFF_TIMEOUT, receiver)
        .await
        .map_err(|_| anyhow::anyhow!("paste main-thread acknowledgment timed out"))?
        .map_err(|_| anyhow::anyhow!("paste main-thread acknowledgment was lost"))?
}

/// Preserve the external HWND and owner through editing/hiding, then verify foreground before Ctrl+V.
pub async fn paste_to_external_application(app_handle: &AppHandle, pinned: bool) -> Result<()> {
    let (target, generation) = on_main_thread(app_handle, move |handle| {
        let current = external_window(unsafe { GetForegroundWindow() });
        let origin = PASTE_SESSION
            .lock()
            .unwrap_or_else(|err| err.into_inner())
            .origin;
        let target = paste_target::select_target(current, origin, is_live_target)
            .ok_or_else(|| anyhow::anyhow!("no live external paste target"))?;

        set_clipboard_window_editing(&handle, false)?;
        if !pinned {
            super::hide_window(&handle, CLIPBOARD_WINDOW_LABEL)?;
        }
        if !is_live_target(target) {
            return Err(anyhow::anyhow!("paste target ownership changed during handoff").into());
        }
        let hwnd = HWND(target.hwnd);
        if unsafe { GetForegroundWindow() } != hwnd
            && !unsafe { SetForegroundWindow(hwnd).as_bool() }
        {
            return Err(anyhow::anyhow!(
                "Windows refused foreground activation for the paste target"
            )
            .into());
        }
        let generation = PASTE_SESSION
            .lock()
            .unwrap_or_else(|err| err.into_inner())
            .generation;
        Ok((target, generation))
    })
    .await?;

    let deadline = Instant::now() + PASTE_HANDOFF_TIMEOUT;
    loop {
        let posted = on_main_thread(app_handle, move |handle| {
            if PASTE_SESSION
                .lock()
                .unwrap_or_else(|err| err.into_inner())
                .generation
                != generation
            {
                return Err(
                    anyhow::anyhow!("paste handoff was superseded by a window change").into(),
                );
            }
            if !is_live_target(target) {
                return Err(
                    anyhow::anyhow!("paste target was closed or its HWND was reused").into(),
                );
            }
            let window = get_window(&handle, CLIPBOARD_WINDOW_LABEL)?;
            let raw_hwnd = window.hwnd().map_err(|err| anyhow::anyhow!(err))?;
            let clipboard_hwnd = HWND(raw_hwnd.0 as isize);
            let nonfocusable = unsafe { GetWindowLongW(clipboard_hwnd, GWL_EXSTYLE) } as u32
                & WS_EX_NOACTIVATE.0
                != 0;
            let foreground = external_window(unsafe { GetForegroundWindow() });
            if !paste_target::ready_to_paste(
                is_live_target(target),
                foreground == Some(target),
                nonfocusable,
                unsafe { IsWindowVisible(clipboard_hwnd).as_bool() },
                pinned,
            ) {
                return Ok(false);
            }
            // Existing SendInput validates all four events; a UIPI denial remains an error.
            crate::keystroke::simulate_paste()?;
            Ok(true)
        })
        .await?;
        if posted {
            return Ok(());
        }
        if Instant::now() >= deadline {
            return Err(
                anyhow::anyhow!("paste destination did not acquire foreground focus").into(),
            );
        }
        tokio::time::sleep(PASTE_READINESS_POLL).await;
    }
}

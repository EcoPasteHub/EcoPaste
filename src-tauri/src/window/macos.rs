//! macOS 窗口管理：剪贴板窗口转 NSPanel（show_and_make_key 拿键盘焦点但不激活 App），
//! 其它窗口走常规 show/hide。

#![allow(clippy::unused_unit)]

use std::sync::{LazyLock, Mutex};
use std::time::{Duration, Instant};

use objc2_app_kit::{
    NSApplication, NSApplicationActivationOptions, NSRunningApplication, NSWorkspace,
};

use tauri::{AppHandle, Manager};
use tauri_nspanel::{
    tauri_panel, CollectionBehavior, ManagerExt, PanelLevel, StyleMask, WebviewWindowExt,
};

use super::{get_window, CLIPBOARD_WINDOW_LABEL, ONBOARDING_WINDOW_LABEL, PREFERENCE_WINDOW_LABEL};
use crate::core::Result;
use crate::settings::SettingsStore;

#[path = "paste_target.rs"]
mod paste_target;

const CLIPBOARD_PANEL_SHOW_DELAY: Duration = Duration::from_millis(16);
const PASTE_HANDOFF_TIMEOUT: Duration = Duration::from_secs(1);
const PASTE_READINESS_POLL: Duration = Duration::from_millis(10);
static PASTE_SESSION: LazyLock<Mutex<paste_target::PasteSession<Retained<NSRunningApplication>>>> =
    LazyLock::new(|| Mutex::new(paste_target::PasteSession::default()));

tauri_panel! {
    panel!(MainPanel {
        config: {
            is_floating_panel: true,
            can_become_key_window: true,
            can_become_main_window: false
        }
    })

    panel_event!(MainPanelEventHandler {
        window_did_resign_key(notification: &NSNotification) -> (),
    })
}

/// setup 最早阶段调用：plugin 必须在 to_panel 前注册。
pub fn register_plugin(app_handle: &AppHandle) {
    let _ = app_handle.plugin(tauri_nspanel::init());
}

/// setup 末尾调用：转 NSPanel + 绑事件 emit。
pub fn setup_clipboard_panel(app_handle: &AppHandle) -> Result<()> {
    show_taskbar_icon(app_handle, false)?;

    let clipboard_window = get_window(app_handle, CLIPBOARD_WINDOW_LABEL)?;

    let panel = clipboard_window
        .to_panel::<MainPanel>()
        .map_err(|e| anyhow::anyhow!("to_panel failed: {e:?}"))?;

    panel.set_corner_radius(16.0);
    panel.set_level(PanelLevel::Dock.value());
    panel.set_style_mask(StyleMask::empty().resizable().nonactivating_panel().into());
    panel.set_collection_behavior(
        CollectionBehavior::new()
            .stationary()
            .move_to_active_space()
            .full_screen_auxiliary()
            .into(),
    );

    let handler = MainPanelEventHandler::new();

    let resign_handle = app_handle.clone();
    handler.window_did_resign_key(move |_| {
        if !super::should_auto_hide_clipboard_window() {
            return;
        }

        // 失焦即隐藏：Tauri 不主动隐藏 NSPanel，统一走 window::hide_window
        // 以触发 `window://visibility` 等下游副作用。
        if let Err(err) = super::hide_window(&resign_handle, CLIPBOARD_WINDOW_LABEL) {
            log::warn!("auto-hide clipboard window on resign-key failed: {err}");
        }
    });

    panel.set_event_handler(Some(handler.as_ref()));

    Ok(())
}

pub fn show_window(app_handle: &AppHandle, label: &str) -> Result<()> {
    if label == CLIPBOARD_WINDOW_LABEL {
        show_clipboard_panel(app_handle)
    } else {
        let window = get_window(app_handle, label)?;
        window.show().map_err(|e| anyhow::anyhow!(e))?;
        window.unminimize().map_err(|e| anyhow::anyhow!(e))?;
        window.set_focus().map_err(|e| anyhow::anyhow!(e))?;
        Ok(())
    }
}

pub fn hide_window(app_handle: &AppHandle, label: &str) -> Result<()> {
    if label == CLIPBOARD_WINDOW_LABEL {
        hide_clipboard_panel(app_handle)
    } else {
        get_window(app_handle, label)?
            .hide()
            .map_err(|e| anyhow::anyhow!(e))?;
        Ok(())
    }
}

pub fn show_taskbar_icon(app_handle: &AppHandle, visible: bool) -> Result<()> {
    app_handle
        .set_dock_visibility(visible)
        .map_err(|e| anyhow::anyhow!(e))?;
    Ok(())
}

/// 点击 dock 图标 reopen 时，无可见窗口则唤起偏好窗口。
pub fn handle_reopen(app_handle: &AppHandle, has_visible_windows: bool) {
    if has_visible_windows {
        return;
    }

    if let Some(settings_store) = app_handle.try_state::<SettingsStore>() {
        if !settings_store.snapshot().onboarding.completed {
            if let Err(err) = super::show_window(app_handle, ONBOARDING_WINDOW_LABEL) {
                log::error!("show onboarding window on reopen failed: {err:?}");
            }
            return;
        }
    }

    if let Err(err) = show_window(app_handle, PREFERENCE_WINDOW_LABEL) {
        log::error!("show preference window on reopen failed: {err:?}");
    }
}

/// 所有 panel 方法必须在主线程。
fn show_clipboard_panel(app_handle: &AppHandle) -> Result<()> {
    let external = NSWorkspace::sharedWorkspace()
        .frontmostApplication()
        .filter(|app| is_external_paste_target(app));
    let visible = get_window(app_handle, CLIPBOARD_WINDOW_LABEL)?
        .is_visible()
        .map_err(|err| anyhow::anyhow!(err))?;
    let generation = PASTE_SESSION
        .lock()
        .unwrap_or_else(|err| err.into_inner())
        .begin_show(external, visible);
    let handle = app_handle.clone();

    tauri::async_runtime::spawn(async move {
        tokio::time::sleep(CLIPBOARD_PANEL_SHOW_DELAY).await;

        let panel_handle = handle.clone();
        if let Err(err) = handle.run_on_main_thread(move || {
            {
                let mut session = PASTE_SESSION.lock().unwrap_or_else(|err| err.into_inner());
                if !session.is_current(generation) {
                    return;
                }
                session.mark_shown();
            }
            if let Ok(panel) = panel_handle.get_webview_panel(CLIPBOARD_WINDOW_LABEL) {
                panel.show_and_make_key();
                // show 时切到 can_join_all_spaces：跟随用户当前 space 出现。
                panel.set_collection_behavior(
                    CollectionBehavior::new()
                        .stationary()
                        .can_join_all_spaces()
                        .full_screen_auxiliary()
                        .into(),
                );
                super::preview::resume_after_clipboard_show();
                super::emit_visibility(&panel_handle, CLIPBOARD_WINDOW_LABEL, true);
                super::lifecycle::on_shown(&panel_handle, CLIPBOARD_WINDOW_LABEL);
            }
        }) {
            log::warn!("show clipboard panel on main thread failed: {err}");
        }
    });

    Ok(())
}

fn hide_clipboard_panel(app_handle: &AppHandle) -> Result<()> {
    let panel = app_handle
        .get_webview_panel(CLIPBOARD_WINDOW_LABEL)
        .map_err(|err| anyhow::anyhow!("clipboard panel is unavailable: {err:?}"))?;
    PASTE_SESSION
        .lock()
        .unwrap_or_else(|err| err.into_inner())
        .close();
    app_handle
        .run_on_main_thread(move || {
            panel.hide();
            // hide 后切回 move_to_active_space：下次 show 时按当前 space 重新落位。
            panel.set_collection_behavior(
                CollectionBehavior::new()
                    .stationary()
                    .move_to_active_space()
                    .full_screen_auxiliary()
                    .into(),
            );
        })
        .map_err(|e| anyhow::anyhow!(e))?;
    Ok(())
}

fn is_external_paste_target(app: &NSRunningApplication) -> bool {
    app.processIdentifier() != std::process::id() as i32 && !app.isTerminated()
}

/// Await the actual main-thread operation, not merely successful event-loop submission.
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

/// Retain the destination instance, hand off keyboard focus, then verify and post on the same main-thread turn.
pub async fn paste_to_external_application(app_handle: &AppHandle, pinned: bool) -> Result<()> {
    let (target, generation) = on_main_thread(app_handle, move |handle| {
        let current = NSWorkspace::sharedWorkspace().frontmostApplication();
        let origin = PASTE_SESSION
            .lock()
            .unwrap_or_else(|err| err.into_inner())
            .origin()
            .cloned();
        let target =
            paste_target::select_target(current, origin, |app| is_external_paste_target(app))
                .ok_or_else(|| anyhow::anyhow!("no live external paste target"))?;
        let panel = handle
            .get_webview_panel(CLIPBOARD_WINDOW_LABEL)
            .map_err(|err| anyhow::anyhow!("paste panel is unavailable: {err:?}"))?;

        if pinned {
            PASTE_SESSION
                .lock()
                .unwrap_or_else(|err| err.into_inner())
                .cancel_pending_show();
            panel.resign_key_window();
        } else {
            // Keep geometry, preview and visibility/lifecycle effects of normal hiding.
            super::hide_window(&handle, CLIPBOARD_WINDOW_LABEL)?;
        }

        let mtm = MainThreadMarker::new()
            .ok_or_else(|| anyhow::anyhow!("paste handoff is not on main thread"))?;
        let application = NSApplication::sharedApplication(mtm);
        if application.respondsToSelector(objc2::sel!(yieldActivationToApplication:)) {
            application.yieldActivationToApplication(&target);
        }
        if !target.activateWithOptions(NSApplicationActivationOptions::empty()) {
            return Err(anyhow::anyhow!("paste target activation was refused").into());
        }
        let generation = PASTE_SESSION
            .lock()
            .unwrap_or_else(|err| err.into_inner())
            .generation();
        Ok((target, generation))
    })
    .await?;

    let deadline = Instant::now() + PASTE_HANDOFF_TIMEOUT;
    loop {
        let target = target.clone();
        let posted = on_main_thread(app_handle, move |handle| {
            if !PASTE_SESSION
                .lock()
                .unwrap_or_else(|err| err.into_inner())
                .is_current(generation)
            {
                return Err(
                    anyhow::anyhow!("paste handoff was superseded by a window change").into(),
                );
            }
            if !is_external_paste_target(&target) {
                return Err(anyhow::anyhow!("paste target is no longer running").into());
            }
            let panel = handle
                .get_webview_panel(CLIPBOARD_WINDOW_LABEL)
                .map_err(|err| anyhow::anyhow!("paste panel is unavailable: {err:?}"))?;
            let frontmost = NSWorkspace::sharedWorkspace().frontmostApplication();
            if !paste_target::ready_to_paste(
                is_external_paste_target(&target),
                frontmost.as_deref() == Some(&*target) && target.isActive(),
                panel.as_panel().isKeyWindow(),
                panel.is_visible(),
                pinned,
            ) {
                return Ok(false);
            }
            crate::keystroke::simulate_paste()?;
            Ok(true)
        })
        .await?;
        if posted {
            break;
        }
        if Instant::now() >= deadline {
            return Err(anyhow::anyhow!("paste target did not acquire keyboard focus").into());
        }
        tokio::time::sleep(PASTE_READINESS_POLL).await;
    }

    if pinned {
        tokio::time::sleep(Duration::from_millis(50)).await;
        let restore_result = on_main_thread(app_handle, move |handle| {
            if super::is_clipboard_window_pinned()
                && PASTE_SESSION
                    .lock()
                    .unwrap_or_else(|err| err.into_inner())
                    .is_current(generation)
                && is_external_paste_target(&target)
                && NSWorkspace::sharedWorkspace()
                    .frontmostApplication()
                    .as_deref()
                    == Some(&*target)
            {
                let panel = handle
                    .get_webview_panel(CLIPBOARD_WINDOW_LABEL)
                    .map_err(|err| anyhow::anyhow!("paste panel is unavailable: {err:?}"))?;
                if panel.is_visible() {
                    panel.make_key_window();
                }
            }
            Ok(())
        })
        .await;
        if let Err(err) = restore_result {
            log::warn!("restore pinned panel after completed paste failed: {err:?}");
        }
    }
    Ok(())
}

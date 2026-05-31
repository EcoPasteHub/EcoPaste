//! macOS 窗口管理：主窗口转 NSPanel（show_and_make_key 拿键盘焦点但不激活 App），
//! 其它窗口走常规 show/hide。

use tauri::{AppHandle, Emitter, EventTarget, Manager};
use tauri_nspanel::{
    tauri_panel, CollectionBehavior, ManagerExt, PanelLevel, StyleMask, WebviewWindowExt,
};

use super::{get_window, MAIN_WINDOW_LABEL, PREFERENCE_WINDOW_LABEL};
use crate::core::Result;

/// 复用 Tauri 原生事件名，前端已有的 listen("tauri://focus" 等) 能直接接住——
/// NSPanel 化后 Tauri 自身不再触发这些事件，由我们手动 emit 补齐。
const WINDOW_FOCUS_EVENT: &str = "tauri://focus";
const WINDOW_BLUR_EVENT: &str = "tauri://blur";
const WINDOW_MOVED_EVENT: &str = "tauri://move";
const WINDOW_RESIZED_EVENT: &str = "tauri://resize";

tauri_panel! {
    panel!(MainPanel {
        config: {
            is_floating_panel: true,
            can_become_key_window: true,
            can_become_main_window: false
        }
    })

    panel_event!(MainPanelEventHandler {
        window_did_become_key(notification: &NSNotification) -> (),
        window_did_resign_key(notification: &NSNotification) -> (),
        window_did_resize(notification: &NSNotification) -> (),
        window_did_move(notification: &NSNotification) -> (),
    })
}

/// setup 最早阶段调用：plugin 必须在 to_panel 前注册。
pub fn register_plugin(app_handle: &AppHandle) {
    let _ = app_handle.plugin(tauri_nspanel::init());
}

/// setup 末尾调用：转 NSPanel + 绑事件 emit。
pub fn setup_main(app_handle: &AppHandle) -> Result<()> {
    show_taskbar_icon(app_handle, false)?;

    let main_window = get_window(app_handle, MAIN_WINDOW_LABEL)?;

    let panel = main_window
        .to_panel::<MainPanel>()
        .map_err(|e| anyhow::anyhow!("to_panel failed: {e:?}"))?;

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

    let window = main_window.clone();
    handler.window_did_become_key(move |_| {
        let _ = window.emit_to(
            EventTarget::labeled(MAIN_WINDOW_LABEL),
            WINDOW_FOCUS_EVENT,
            true,
        );
    });

    let window = main_window.clone();
    handler.window_did_resign_key(move |_| {
        let _ = window.emit_to(
            EventTarget::labeled(MAIN_WINDOW_LABEL),
            WINDOW_BLUR_EVENT,
            true,
        );
    });

    let window = main_window.clone();
    handler.window_did_resize(move |_| {
        let target = EventTarget::labeled(MAIN_WINDOW_LABEL);
        if let Ok(position) = window.outer_position() {
            let _ = window.emit_to(target.clone(), WINDOW_MOVED_EVENT, position);
        }
        if let Ok(size) = window.inner_size() {
            let _ = window.emit_to(target, WINDOW_RESIZED_EVENT, size);
        }
    });

    let window = main_window.clone();
    handler.window_did_move(move |_| {
        if let Ok(position) = window.outer_position() {
            let _ = window.emit_to(
                EventTarget::labeled(MAIN_WINDOW_LABEL),
                WINDOW_MOVED_EVENT,
                position,
            );
        }
    });

    panel.set_event_handler(Some(handler.as_ref()));

    Ok(())
}

pub fn show_window(app_handle: &AppHandle, label: &str) -> Result<()> {
    if label == MAIN_WINDOW_LABEL {
        show_main_panel(app_handle)
    } else {
        let window = get_window(app_handle, label)?;
        window.show().map_err(|e| anyhow::anyhow!(e))?;
        window.unminimize().map_err(|e| anyhow::anyhow!(e))?;
        window.set_focus().map_err(|e| anyhow::anyhow!(e))?;
        Ok(())
    }
}

pub fn hide_window(app_handle: &AppHandle, label: &str) -> Result<()> {
    if label == MAIN_WINDOW_LABEL {
        hide_main_panel(app_handle)
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
    if let Err(err) = show_window(app_handle, PREFERENCE_WINDOW_LABEL) {
        log::error!("show preference window on reopen failed: {err:?}");
    }
}

/// 所有 panel 方法必须在主线程。
fn show_main_panel(app_handle: &AppHandle) -> Result<()> {
    let handle = app_handle.clone();
    app_handle
        .run_on_main_thread(move || {
            if let Ok(panel) = handle.get_webview_panel(MAIN_WINDOW_LABEL) {
                panel.show_and_make_key();
                // show 时切到 can_join_all_spaces：跟随用户当前 space 出现。
                panel.set_collection_behavior(
                    CollectionBehavior::new()
                        .stationary()
                        .can_join_all_spaces()
                        .full_screen_auxiliary()
                        .into(),
                );
            }
        })
        .map_err(|e| anyhow::anyhow!(e))?;
    Ok(())
}

fn hide_main_panel(app_handle: &AppHandle) -> Result<()> {
    let handle = app_handle.clone();
    app_handle
        .run_on_main_thread(move || {
            if let Ok(panel) = handle.get_webview_panel(MAIN_WINDOW_LABEL) {
                panel.hide();
                // hide 后切回 move_to_active_space：下次 show 时按当前 space 重新落位。
                panel.set_collection_behavior(
                    CollectionBehavior::new()
                        .stationary()
                        .move_to_active_space()
                        .full_screen_auxiliary()
                        .into(),
                );
            }
        })
        .map_err(|e| anyhow::anyhow!(e))?;
    Ok(())
}

//! macOS 窗口管理：主窗口转 NSPanel（show_and_make_key 拿键盘焦点但不激活 App），
//! 其它窗口走常规 show/hide。

#![allow(clippy::unused_unit)]

use tauri::{AppHandle, Manager};
use tauri_nspanel::{
    tauri_panel, CollectionBehavior, ManagerExt, PanelLevel, StyleMask, WebviewWindowExt,
};

use super::{get_window, MAIN_WINDOW_LABEL, PREFERENCE_WINDOW_LABEL};
use crate::core::Result;

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

    let resign_handle = app_handle.clone();
    handler.window_did_resign_key(move |_| {
        if super::is_main_window_pinned() {
            return;
        }

        // 失焦即隐藏：Tauri 不主动隐藏 NSPanel，统一走 window::hide_window
        // 以触发 `window://visibility` 等下游副作用。
        if let Err(err) = super::hide_window(&resign_handle, MAIN_WINDOW_LABEL) {
            log::warn!("auto-hide main window on resign-key failed: {err}");
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

/// 让主 panel 放弃 key 状态，但保持可见——用于固定窗口下的粘贴：
/// panel 仍是 key window 时 CGEvent ⌘V 会被 panel 自身吞掉，resign 后键焦点回到前台 App 的窗口。
pub fn resign_main_panel_key(app_handle: &AppHandle) -> Result<()> {
    let handle = app_handle.clone();
    app_handle
        .run_on_main_thread(move || {
            if let Ok(panel) = handle.get_webview_panel(MAIN_WINDOW_LABEL) {
                panel.resign_key_window();
            }
        })
        .map_err(|e| anyhow::anyhow!(e))?;
    Ok(())
}

/// 粘贴完成后把 key 状态拿回来：固定窗口模式下用户还要继续用键盘 / 列表操作。
pub fn make_main_panel_key(app_handle: &AppHandle) -> Result<()> {
    let handle = app_handle.clone();
    app_handle
        .run_on_main_thread(move || {
            if let Ok(panel) = handle.get_webview_panel(MAIN_WINDOW_LABEL) {
                panel.make_key_window();
            }
        })
        .map_err(|e| anyhow::anyhow!(e))?;
    Ok(())
}

//! 全局快捷键：注册由 Rust 主导，前端在偏好设置里改完通过 `update_settings` 触发重注册。
//!
//! 配置来自 `settings::Shortcuts`。本模块只负责 OS 级注册——`paste_plain` / `quick_paste`
//! 是窗口内交互（前端 `useKeyPress`），不在这里处理。

use std::sync::Mutex;

use serde::Serialize;
use tauri::{AppHandle, Emitter, Manager};
use tauri_plugin_global_shortcut::{GlobalShortcutExt, Shortcut, ShortcutEvent, ShortcutState};

use crate::core::{AppError, Result};
use crate::settings::Shortcuts;
use crate::window::{self, MAIN_WINDOW_LABEL, PREFERENCE_WINDOW_LABEL};

pub const CONFLICT_EVENT: &str = "shortcut://conflict";

#[derive(Debug, Clone, Serialize)]
pub struct ShortcutConflict {
    pub action: &'static str,
    pub binding: String,
    pub reason: String,
}

#[derive(Default)]
pub struct ShortcutManager {
    active: Mutex<Vec<(&'static str, Shortcut)>>,
}

pub fn init(app: &AppHandle, shortcuts: &Shortcuts) -> Result<()> {
    app.manage(ShortcutManager::default());
    apply(app, shortcuts)
}

/// 全量替换：先取消上一轮注册，再逐个注册新项。注册失败仅 emit 不中断其它项。
pub fn apply(app: &AppHandle, shortcuts: &Shortcuts) -> Result<()> {
    let plugin = app.global_shortcut();
    let manager = app.state::<ShortcutManager>();

    let previous = {
        let mut guard = manager.active.lock().expect("shortcut state poisoned");
        std::mem::take(&mut *guard)
    };
    for (_, shortcut) in &previous {
        if let Err(err) = plugin.unregister(*shortcut) {
            log::warn!("unregister previous shortcut failed: {err:?}");
        }
    }

    let desired: [(&'static str, &str); 2] = [
        ("open_clipboard", &shortcuts.open_clipboard),
        ("open_preference", &shortcuts.open_preference),
    ];

    let mut active = Vec::new();
    for (action, binding) in desired {
        if binding.trim().is_empty() {
            continue;
        }
        match register_one(app, action, binding) {
            Ok(shortcut) => active.push((action, shortcut)),
            Err(err) => {
                log::warn!("register shortcut {action}={binding} failed: {err}");
                let _ = app.emit(
                    CONFLICT_EVENT,
                    ShortcutConflict {
                        action,
                        binding: binding.into(),
                        reason: err.to_string(),
                    },
                );
            }
        }
    }

    *manager.active.lock().expect("shortcut state poisoned") = active;
    Ok(())
}

fn register_one(app: &AppHandle, action: &'static str, binding: &str) -> Result<Shortcut> {
    let plugin = app.global_shortcut();
    let shortcut: Shortcut = binding
        .parse()
        .map_err(|err| AppError::Other(anyhow::anyhow!("parse shortcut {binding}: {err}")))?;

    if plugin.is_registered(shortcut) {
        plugin
            .unregister(shortcut)
            .map_err(|err| AppError::Other(anyhow::anyhow!(err)))?;
    }

    plugin
        .on_shortcut(shortcut, move |app, _scut, event| {
            handle_event(app, action, event);
        })
        .map_err(|err| AppError::Other(anyhow::anyhow!(err)))?;
    Ok(shortcut)
}

fn handle_event(app: &AppHandle, action: &'static str, event: ShortcutEvent) {
    // Pressed 触发一次即可（Released 是按键松开），避免 toggle 在按下/松开各执行一次回弹。
    if !matches!(event.state(), ShortcutState::Pressed) {
        return;
    }
    let label = match action {
        "open_clipboard" => MAIN_WINDOW_LABEL,
        "open_preference" => PREFERENCE_WINDOW_LABEL,
        _ => return,
    };
    if let Err(err) = window::toggle_window(app, label) {
        log::warn!("toggle window via shortcut {action} failed: {err}");
    }
}

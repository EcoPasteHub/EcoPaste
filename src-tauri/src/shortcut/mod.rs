//! 全局快捷键：注册由 Rust 主导，前端只在偏好设置里改绑定值后调命令重注册。
//!
//! 默认绑定对齐旧版（`stores/global.ts`）：
//! - 打开剪贴板窗：Alt+C
//! - 打开偏好窗：Alt+X
//! - 纯文本粘贴 / 快捷粘贴：窗口内局部按键（前端 `useKeyPress`），不在此注册。

use std::sync::Mutex;

use serde::{Deserialize, Serialize};
use tauri::{AppHandle, Emitter, Manager};
use tauri_plugin_global_shortcut::{GlobalShortcutExt, Shortcut, ShortcutEvent, ShortcutState};

use crate::core::{AppError, Result};
use crate::window::{self, MAIN_WINDOW_LABEL, PREFERENCE_WINDOW_LABEL};

pub const CONFLICT_EVENT: &str = "shortcut://conflict";

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ShortcutBindings {
    pub open_clipboard: String,
    pub open_preference: String,
}

impl Default for ShortcutBindings {
    fn default() -> Self {
        Self {
            open_clipboard: "Alt+C".into(),
            open_preference: "Alt+X".into(),
        }
    }
}

#[derive(Debug, Clone, Serialize)]
pub struct ShortcutConflict {
    pub action: &'static str,
    pub binding: String,
    pub reason: String,
}

#[derive(Default)]
pub struct ShortcutManager {
    state: Mutex<ShortcutManagerState>,
}

#[derive(Default)]
struct ShortcutManagerState {
    bindings: ShortcutBindings,
    active: Vec<(&'static str, Shortcut)>,
}

pub fn init(app: &AppHandle) -> Result<()> {
    app.manage(ShortcutManager::default());
    apply(app, ShortcutBindings::default())
}

/// 用新绑定全量替换：先取消已注册项，再逐个注册新项。注册失败仅记录 + emit，不中断其它项。
pub fn apply(app: &AppHandle, bindings: ShortcutBindings) -> Result<()> {
    let plugin = app.global_shortcut();
    let manager = app.state::<ShortcutManager>();

    let previous = {
        let mut guard = manager.state.lock().expect("shortcut state poisoned");
        std::mem::take(&mut guard.active)
    };
    for (_, shortcut) in &previous {
        if let Err(err) = plugin.unregister(*shortcut) {
            log::warn!("unregister previous shortcut failed: {err:?}");
        }
    }

    let desired: [(&'static str, &str); 2] = [
        ("open_clipboard", &bindings.open_clipboard),
        ("open_preference", &bindings.open_preference),
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

    let mut guard = manager.state.lock().expect("shortcut state poisoned");
    guard.bindings = bindings;
    guard.active = active;
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

pub fn current_bindings(app: &AppHandle) -> ShortcutBindings {
    let manager = app.state::<ShortcutManager>();
    let bindings = manager
        .state
        .lock()
        .expect("shortcut state poisoned")
        .bindings
        .clone();
    bindings
}

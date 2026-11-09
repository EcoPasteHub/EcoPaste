use std::collections::HashMap;
use std::fs;
use std::path::PathBuf;
use std::sync::{Mutex, RwLock};

use anyhow::Context;
use serde::{Deserialize, Serialize};
use tauri::{AppHandle, Manager, PhysicalPosition, PhysicalSize};

use crate::core::Result;

const STATE_FILENAME: &str = "window-state.json";
/// 可恢复窗口的最小边长（物理像素）。最小化等状态下 `inner_size` 会返回 0（或个位数），
/// 一旦落盘，下次 `set_size` 会把窗口恢复成无法操作的细条，因此两个方向都要守住下限。
const MIN_RESTORABLE_SIDE: u32 = 100;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WindowState {
    pub x: i32,
    pub y: i32,
    pub width: u32,
    pub height: u32,
}

impl WindowState {
    /// 尺寸是否为可恢复的真实窗口几何。
    fn is_restorable(&self) -> bool {
        self.width >= MIN_RESTORABLE_SIDE && self.height >= MIN_RESTORABLE_SIDE
    }
}

pub struct WindowStateStore {
    path: RwLock<PathBuf>,
    states: Mutex<HashMap<String, WindowState>>,
}

impl WindowStateStore {
    pub fn new(app: &AppHandle) -> Result<Self> {
        let dir = crate::core::paths::state_dir(app)?;

        fs::create_dir_all(&dir).with_context(|| format!("failed to create dir at {dir:?}"))?;

        let path = dir.join(STATE_FILENAME);

        let states = if path.exists() {
            match fs::read_to_string(&path) {
                Ok(content) => serde_json::from_str(&content).unwrap_or_else(|e| {
                    log::warn!("failed to parse window state at {path:?}, using defaults: {e}");
                    HashMap::new()
                }),
                Err(e) => {
                    log::warn!("failed to read window state at {path:?}, using defaults: {e}");
                    HashMap::new()
                }
            }
        } else {
            HashMap::new()
        };

        log::info!("window state store ready at {path:?}");
        Ok(Self {
            path: RwLock::new(path),
            states: Mutex::new(states),
        })
    }

    pub fn save(&self, label: &str, state: WindowState) -> Result<()> {
        let mut states = self.states.lock().unwrap_or_else(|poisoned| {
            log::error!("window state mutex poisoned on save, recovering");
            poisoned.into_inner()
        });
        states.insert(label.to_owned(), state);
        let json =
            serde_json::to_string_pretty(&*states).context("failed to serialize window states")?;
        let path = self.path();
        fs::write(&path, json)
            .with_context(|| format!("failed to write window state to {:?}", path))?;
        Ok(())
    }

    pub fn get(&self, label: &str) -> Option<WindowState> {
        let states = self.states.lock().unwrap_or_else(|poisoned| {
            log::error!("window state mutex poisoned on get, recovering");
            poisoned.into_inner()
        });
        states.get(label).cloned()
    }

    /// 数据目录热切换后重新绑定窗口状态文件，并重新读取新目录里的状态。
    pub fn rebase(&self, app: &AppHandle) -> Result<()> {
        let dir = crate::core::paths::state_dir(app)?;
        fs::create_dir_all(&dir).with_context(|| format!("failed to create dir at {dir:?}"))?;
        let path = dir.join(STATE_FILENAME);
        let next_states = load_states(&path);

        *self.path.write().expect("window state path poisoned") = path;
        *self.states.lock().unwrap_or_else(|poisoned| {
            log::error!("window state mutex poisoned on rebase, recovering");
            poisoned.into_inner()
        }) = next_states;
        Ok(())
    }

    fn path(&self) -> PathBuf {
        self.path
            .read()
            .expect("window state path poisoned")
            .clone()
    }
}

fn load_states(path: &PathBuf) -> HashMap<String, WindowState> {
    if !path.exists() {
        return HashMap::new();
    }

    match fs::read_to_string(path) {
        Ok(content) => serde_json::from_str(&content).unwrap_or_else(|e| {
            log::warn!("failed to parse window state at {path:?}, using defaults: {e}");
            HashMap::new()
        }),
        Err(e) => {
            log::warn!("failed to read window state at {path:?}, using defaults: {e}");
            HashMap::new()
        }
    }
}

/// 读取窗口当前的实时几何（`outer_position` + `inner_size`）并落盘。
/// 在隐藏 / 关闭 / 退出等可靠生命周期点调用即可捕获用户的移动与缩放。
pub fn save_window_state(app: &AppHandle, label: &str) -> Result<()> {
    let window = app
        .get_webview_window(label)
        .ok_or_else(|| anyhow::anyhow!("window not found: {label}"))?;

    let pos = window.outer_position().map_err(|e| anyhow::anyhow!(e))?;
    let size = window.inner_size().map_err(|e| anyhow::anyhow!(e))?;

    let state = WindowState {
        x: pos.x,
        y: pos.y,
        width: size.width,
        height: size.height,
    };
    if !state.is_restorable() {
        log::warn!(
            "skip saving degraded window state for {label}: {}x{} at ({}, {})",
            state.width,
            state.height,
            state.x,
            state.y
        );
        return Ok(());
    }

    let store = app.state::<WindowStateStore>();
    store.save(label, state)
}

/// 恢复窗口的尺寸 + 位置。无存档返回 `Ok(false)`。
///
/// 始终恢复存档尺寸；位置在恢复前校验是否仍位于可用显示器范围内：
/// 若上次所在显示器已被拔出，则 fallback 到当前光标所在屏幕的中心，
/// 避免窗口出现在不可见的虚拟坐标区域。
pub fn restore_window_state(app: &AppHandle, label: &str) -> Result<bool> {
    let store = app.state::<WindowStateStore>();
    let Some(state) = store.get(label) else {
        return Ok(false);
    };
    if !state.is_restorable() {
        // 旧版本可能把最小化窗口的 0×0 几何落盘；整条存档都不可信，
        // 交给 builder 的默认尺寸与居中位置，避免把窗口恢复成细条或放到屏幕外。
        log::warn!(
            "ignore degraded window state for {label}: {}x{} at ({}, {})",
            state.width,
            state.height,
            state.x,
            state.y
        );
        return Ok(false);
    }

    let window = app
        .get_webview_window(label)
        .ok_or_else(|| anyhow::anyhow!("window not found: {label}"))?;

    window
        .set_size(PhysicalSize::new(state.width, state.height))
        .map_err(|e| anyhow::anyhow!(e))?;

    let monitors = window
        .available_monitors()
        .map_err(|e| anyhow::anyhow!(e))?;
    let on_screen = monitors.iter().any(|m| {
        let mx = m.position().x;
        let my = m.position().y;
        let mw = m.size().width as i32;
        let mh = m.size().height as i32;
        state.x >= mx && state.x < mx + mw && state.y >= my && state.y < my + mh
    });

    if on_screen {
        window
            .set_position(PhysicalPosition::new(state.x, state.y))
            .map_err(|e| anyhow::anyhow!(e))?;
    } else {
        super::position::center_on_cursor_monitor(&window)?;
    }

    Ok(true)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn state(width: u32, height: u32) -> WindowState {
        WindowState {
            x: 0,
            y: 0,
            width,
            height,
        }
    }

    #[test]
    fn zero_sized_state_is_not_restorable() {
        assert!(!state(0, 0).is_restorable());
        assert!(!state(0, 600).is_restorable());
        assert!(!state(960, 0).is_restorable());
    }

    #[test]
    fn real_window_size_is_restorable() {
        assert!(state(MIN_RESTORABLE_SIDE, MIN_RESTORABLE_SIDE).is_restorable());
        assert!(state(960, 600).is_restorable());
    }
}

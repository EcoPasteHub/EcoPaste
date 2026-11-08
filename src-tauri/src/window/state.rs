use std::collections::HashMap;
use std::fs;
use std::path::PathBuf;
use std::sync::{Mutex, RwLock};

use anyhow::Context;
use serde::{Deserialize, Serialize};
use tauri::{AppHandle, Manager, Monitor, PhysicalPosition, PhysicalSize, Window};

use crate::core::Result;

const STATE_FILENAME: &str = "window-state.json";
const SCREEN_MARGIN_LOGICAL: f64 = 16.0;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WindowState {
    pub x: i32,
    pub y: i32,
    pub width: u32,
    pub height: u32,
    /// Scale factor used when `width` / `height` were captured.
    ///
    /// Missing values are valid for state files written before Remote Desktop
    /// DPI handling was added.
    #[serde(default)]
    pub scale_factor: Option<f64>,
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

    /// Update the in-memory geometry without writing the state file.
    fn remember(&self, label: &str, state: WindowState) {
        let mut states = self.states.lock().unwrap_or_else(|poisoned| {
            log::error!("window state mutex poisoned on remember, recovering");
            poisoned.into_inner()
        });
        states.insert(label.to_owned(), state);
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
    let scale_factor = window.scale_factor().map_err(|e| anyhow::anyhow!(e))?;

    let store = app.state::<WindowStateStore>();
    store.save(
        label,
        WindowState {
            x: pos.x,
            y: pos.y,
            width: size.width,
            height: size.height,
            scale_factor: Some(scale_factor),
        },
    )
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

    let window = app
        .get_webview_window(label)
        .ok_or_else(|| anyhow::anyhow!("window not found: {label}"))?;

    let monitors = window
        .available_monitors()
        .map_err(|e| anyhow::anyhow!(e))?;
    let saved_monitor = monitor_containing_point(&monitors, state.x, state.y);
    let cursor_monitor = monitor_from_cursor(&window)?;
    let current_monitor = window.current_monitor().ok().flatten();
    let target_monitor = saved_monitor
        .or(cursor_monitor.as_ref())
        .or(current_monitor.as_ref());
    let target_scale = target_monitor
        .map(Monitor::scale_factor)
        .or(state.scale_factor)
        .unwrap_or(1.0);
    let size =
        constrain_size_to_monitor(state.physical_size_at_scale(target_scale), target_monitor);

    window.set_size(size).map_err(|e| anyhow::anyhow!(e))?;

    let position = match (saved_monitor, target_monitor) {
        (Some(monitor), Some(_)) => {
            clamp_position_to_monitor(PhysicalPosition::new(state.x, state.y), size, monitor)
        }
        (_, Some(monitor)) => center_position_in_monitor(size, monitor),
        (_, None) => PhysicalPosition::new(state.x, state.y),
    };
    window
        .set_position(position)
        .map_err(|e| anyhow::anyhow!(e))?;

    app.state::<WindowStateStore>().remember(
        label,
        WindowState {
            x: position.x,
            y: position.y,
            width: size.width,
            height: size.height,
            scale_factor: Some(target_scale),
        },
    );

    Ok(true)
}

/// Remember live resize/move geometry without forcing an immediate disk write.
pub fn remember_window_metrics(window: &Window) {
    let (Ok(position), Ok(size), Ok(scale_factor)) = (
        window.outer_position(),
        window.inner_size(),
        window.scale_factor(),
    ) else {
        return;
    };

    let Some(store) = window.app_handle().try_state::<WindowStateStore>() else {
        return;
    };
    store.remember(
        window.label(),
        WindowState {
            x: position.x,
            y: position.y,
            width: size.width,
            height: size.height,
            scale_factor: Some(scale_factor),
        },
    );
}

/// Correct a live Windows/RDP scale change. Tao normally preserves logical size,
/// but it keeps physical size when Windows' "show window contents while dragging"
/// setting is disabled. The previous in-memory state supplies the logical anchor.
pub fn handle_scale_factor_changed(
    window: &Window,
    scale_factor: f64,
    suggested_size: PhysicalSize<u32>,
) {
    let Some(store) = window.app_handle().try_state::<WindowStateStore>() else {
        return;
    };
    let saved = store.get(window.label());
    let desired_size = saved
        .as_ref()
        .filter(|state| {
            state
                .scale_factor
                .is_some_and(|saved_scale| (saved_scale - scale_factor).abs() > f64::EPSILON)
        })
        .map(|state| state.physical_size_at_scale(scale_factor))
        .unwrap_or(suggested_size);
    let monitor = window.current_monitor().ok().flatten();
    let size = constrain_size_to_monitor(desired_size, monitor.as_ref());

    if window.inner_size().ok() != Some(size) {
        if let Err(err) = window.set_size(size) {
            log::warn!(
                "resize {} after scale factor change failed: {err}",
                window.label()
            );
        }
    }

    let position = match (window.outer_position(), monitor.as_ref()) {
        (Ok(position), Some(monitor)) => clamp_position_to_monitor(position, size, monitor),
        (Ok(position), None) => position,
        (Err(_), Some(monitor)) => center_position_in_monitor(size, monitor),
        (Err(_), None) => {
            remember_window_metrics(window);
            return;
        }
    };
    if window.outer_position().ok() != Some(position) {
        if let Err(err) = window.set_position(position) {
            log::warn!(
                "reposition {} after scale factor change failed: {err}",
                window.label()
            );
        }
    }

    store.remember(
        window.label(),
        WindowState {
            x: position.x,
            y: position.y,
            width: size.width,
            height: size.height,
            scale_factor: Some(scale_factor),
        },
    );
}

/// Fit a newly built fixed-size window into the current monitor.
pub fn constrain_window_to_current_monitor(window: &tauri::WebviewWindow) -> Result<()> {
    let monitor = window.current_monitor().map_err(|e| anyhow::anyhow!(e))?;
    let Some(monitor) = monitor else {
        return Ok(());
    };

    let size = constrain_size_to_monitor(
        window.inner_size().map_err(|e| anyhow::anyhow!(e))?,
        Some(&monitor),
    );
    window.set_size(size).map_err(|e| anyhow::anyhow!(e))?;
    let position = clamp_position_to_monitor(
        window.outer_position().map_err(|e| anyhow::anyhow!(e))?,
        size,
        &monitor,
    );
    window
        .set_position(position)
        .map_err(|e| anyhow::anyhow!(e))?;

    Ok(())
}

impl WindowState {
    fn physical_size_at_scale(&self, target_scale: f64) -> PhysicalSize<u32> {
        let saved_scale = self
            .scale_factor
            .filter(|scale| scale.is_finite() && *scale > 0.0)
            .unwrap_or(target_scale);
        let target_scale = target_scale.max(0.1);

        PhysicalSize::new(
            (self.width as f64 / saved_scale * target_scale)
                .round()
                .max(1.0) as u32,
            (self.height as f64 / saved_scale * target_scale)
                .round()
                .max(1.0) as u32,
        )
    }
}

fn monitor_from_cursor(window: &tauri::WebviewWindow) -> Result<Option<Monitor>> {
    let cursor = window.cursor_position().map_err(|e| anyhow::anyhow!(e))?;
    let monitor = window
        .monitor_from_point(cursor.x, cursor.y)
        .map_err(|e| anyhow::anyhow!(e))?;

    Ok(monitor)
}

fn monitor_containing_point(monitors: &[Monitor], x: i32, y: i32) -> Option<&Monitor> {
    monitors.iter().find(|monitor| {
        let position = monitor.position();
        let size = monitor.size();
        x >= position.x
            && x < position.x + size.width as i32
            && y >= position.y
            && y < position.y + size.height as i32
    })
}

fn constrain_size_to_monitor(
    size: PhysicalSize<u32>,
    monitor: Option<&Monitor>,
) -> PhysicalSize<u32> {
    let Some(monitor) = monitor else {
        return size;
    };
    let margin = (SCREEN_MARGIN_LOGICAL * monitor.scale_factor()).round() as u32;
    let max_width = monitor
        .size()
        .width
        .saturating_sub(margin.saturating_mul(2))
        .max(1);
    let max_height = monitor
        .size()
        .height
        .saturating_sub(margin.saturating_mul(2))
        .max(1);

    PhysicalSize::new(size.width.min(max_width), size.height.min(max_height))
}

fn clamp_position_to_monitor(
    position: PhysicalPosition<i32>,
    size: PhysicalSize<u32>,
    monitor: &Monitor,
) -> PhysicalPosition<i32> {
    let monitor_position = monitor.position();
    let monitor_size = monitor.size();
    let margin = (SCREEN_MARGIN_LOGICAL * monitor.scale_factor()).round() as i32;
    let min_x = monitor_position.x + margin.min(monitor_size.width as i32 / 2);
    let min_y = monitor_position.y + margin.min(monitor_size.height as i32 / 2);
    let max_x =
        (monitor_position.x + monitor_size.width as i32 - size.width as i32 - margin).max(min_x);
    let max_y =
        (monitor_position.y + monitor_size.height as i32 - size.height as i32 - margin).max(min_y);

    PhysicalPosition::new(
        position.x.clamp(min_x, max_x),
        position.y.clamp(min_y, max_y),
    )
}

fn center_position_in_monitor(size: PhysicalSize<u32>, monitor: &Monitor) -> PhysicalPosition<i32> {
    let monitor_position = monitor.position();
    let monitor_size = monitor.size();

    PhysicalPosition::new(
        monitor_position.x + (monitor_size.width as i32 - size.width as i32) / 2,
        monitor_position.y + (monitor_size.height as i32 - size.height as i32) / 2,
    )
}

#[cfg(test)]
mod tests {
    use super::*;

    fn state(width: u32, height: u32, scale_factor: f64) -> WindowState {
        WindowState {
            x: 0,
            y: 0,
            width,
            height,
            scale_factor: Some(scale_factor),
        }
    }

    #[test]
    fn window_physical_size_is_recomputed_from_saved_scale() {
        let state = state(540, 900, 1.5);

        assert_eq!(
            state.physical_size_at_scale(1.0),
            PhysicalSize::new(360, 600)
        );
        assert_eq!(
            state.physical_size_at_scale(2.0),
            PhysicalSize::new(720, 1200)
        );
    }

    #[test]
    fn legacy_state_without_scale_uses_current_pixels() {
        let state = WindowState {
            x: 0,
            y: 0,
            width: 640,
            height: 480,
            scale_factor: None,
        };

        assert_eq!(
            state.physical_size_at_scale(1.25),
            PhysicalSize::new(640, 480)
        );
    }
}

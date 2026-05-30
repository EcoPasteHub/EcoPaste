use std::collections::HashMap;
use std::fs;
use std::path::PathBuf;
use std::sync::Mutex;

use anyhow::Context;
use serde::{Deserialize, Serialize};
use tauri::{AppHandle, Manager, PhysicalPosition, PhysicalSize};

use crate::core::Result;

const STATE_FILENAME_RELEASE: &str = "window-state.json";
const STATE_FILENAME_DEV: &str = "window-state.dev.json";

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WindowState {
    pub x: i32,
    pub y: i32,
    pub width: u32,
    pub height: u32,
}

pub struct WindowStateStore {
    path: PathBuf,
    states: Mutex<HashMap<String, WindowState>>,
}

impl WindowStateStore {
    pub fn new(app: &AppHandle) -> Result<Self> {
        let dir = app
            .path()
            .app_local_data_dir()
            .context("failed to resolve app local data dir")?;

        fs::create_dir_all(&dir).with_context(|| format!("failed to create dir at {dir:?}"))?;

        let filename = if cfg!(dev) {
            STATE_FILENAME_DEV
        } else {
            STATE_FILENAME_RELEASE
        };
        let path = dir.join(filename);

        let states = if path.exists() {
            let content = fs::read_to_string(&path).unwrap_or_default();
            serde_json::from_str(&content).unwrap_or_default()
        } else {
            HashMap::new()
        };

        log::info!("window state store ready at {path:?}");
        Ok(Self {
            path,
            states: Mutex::new(states),
        })
    }

    pub fn save(&self, label: &str, state: WindowState) -> Result<()> {
        let mut states = self.states.lock().unwrap();
        states.insert(label.to_owned(), state);
        let json =
            serde_json::to_string_pretty(&*states).context("failed to serialize window states")?;
        fs::write(&self.path, json)
            .with_context(|| format!("failed to write window state to {:?}", self.path))?;
        Ok(())
    }

    pub fn get(&self, label: &str) -> Option<WindowState> {
        let states = self.states.lock().unwrap();
        states.get(label).cloned()
    }
}

pub fn save_window_state(app: &AppHandle, label: &str) -> Result<()> {
    let window = app
        .get_webview_window(label)
        .ok_or_else(|| anyhow::anyhow!("window not found: {label}"))?;

    let pos = window.outer_position().map_err(|e| anyhow::anyhow!(e))?;
    let size = window.inner_size().map_err(|e| anyhow::anyhow!(e))?;

    let store = app.state::<WindowStateStore>();
    store.save(
        label,
        WindowState {
            x: pos.x,
            y: pos.y,
            width: size.width,
            height: size.height,
        },
    )
}

pub fn restore_window_state(app: &AppHandle, label: &str) -> Result<bool> {
    let store = app.state::<WindowStateStore>();
    let Some(state) = store.get(label) else {
        return Ok(false);
    };

    let window = app
        .get_webview_window(label)
        .ok_or_else(|| anyhow::anyhow!("window not found: {label}"))?;

    window
        .set_position(PhysicalPosition::new(state.x, state.y))
        .map_err(|e| anyhow::anyhow!(e))?;
    window
        .set_size(PhysicalSize::new(state.width, state.height))
        .map_err(|e| anyhow::anyhow!(e))?;

    Ok(true)
}

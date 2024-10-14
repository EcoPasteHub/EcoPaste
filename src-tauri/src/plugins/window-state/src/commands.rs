use serde::{Deserialize, Serialize};
use std::{collections::HashMap, fs::File, path::PathBuf};
use tauri::{command, AppHandle, Manager, PhysicalPosition, PhysicalSize, Runtime, Window};

#[derive(Debug, Serialize, Deserialize)]
pub struct WindowState {
    width: u32,
    height: u32,
    x: i32,
    y: i32,
}

// 保存窗口状态
#[command]
pub fn save_state<R: Runtime>(app_handle: AppHandle<R>, save_path: PathBuf) {
    let windows = app_handle.webview_windows();

    let mut data: HashMap<String, WindowState> = HashMap::new();

    for (label, window) in windows {
        if let (Ok(size), Ok(position)) = (window.inner_size(), window.outer_position()) {
            let window_state = WindowState {
                width: size.width,
                height: size.height,
                x: position.x,
                y: position.y,
            };

            data.insert(label, window_state);
        }
    }

    let file = File::create(save_path).unwrap();
    serde_json::to_writer_pretty(file, &data).unwrap();
}

// 恢复窗口状态
#[command]
pub fn restore_state<R: Runtime>(window: Window<R>, save_path: PathBuf) {
    let exists = save_path.exists();

    if !exists {
        return;
    }

    let file = File::open(save_path).unwrap();
    let data: HashMap<String, WindowState> = serde_json::from_reader(file).unwrap();

    if let Some(window_state) = data.get(window.label()) {
        let _ = window.set_size(PhysicalSize {
            width: window_state.width,
            height: window_state.height,
        });

        let _ = window.set_position(PhysicalPosition {
            x: window_state.x,
            y: window_state.y,
        });
    }
}

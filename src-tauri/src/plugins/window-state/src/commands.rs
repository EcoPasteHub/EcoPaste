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

// 获取保存路径
fn get_save_path<R: Runtime>(app_handle: AppHandle<R>) -> Option<PathBuf> {
    if let Ok(app_data_dir) = app_handle.path().app_data_dir() {
        return Some(app_data_dir.join(".window-state.json"));
    } else {
        return None;
    }
}

// 保存窗口状态
#[command]
pub fn save_window_state<R: Runtime>(app_handle: AppHandle<R>) {
    if let Some(save_path) = get_save_path(app_handle.clone()) {
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
}

// 恢复窗口状态
#[command]
pub fn restore_window_state<R: Runtime>(window: Window<R>) {
    let app_handle = window.app_handle().clone();

    if let Some(save_path) = get_save_path(app_handle) {
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
}

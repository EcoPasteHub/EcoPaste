//! Rust 侧用户可见文案。
//!
//! 仅放会直接展示给用户的短文案；日志与内部错误上下文不走这里。

pub mod clipboard_menu;
pub mod commands;
mod en_us;
mod keys;
pub mod tray;
mod zh_cn;

use tauri::{AppHandle, Manager};

use crate::settings::{Language, SettingsStore};

/// 读取当前设置语言；设置尚未初始化时回落到默认语言。
pub fn current_language(app: &AppHandle) -> Language {
    app.try_state::<SettingsStore>()
        .map(|s| s.snapshot().appearance.language)
        .unwrap_or_default()
}

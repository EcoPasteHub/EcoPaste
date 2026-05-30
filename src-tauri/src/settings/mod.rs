//! 应用设置：单一真相源放在 `SettingsStore`，前端通过 `get_settings` / `update_settings` 命令访问。
//!
//! 启动顺序：`init` 先建 store 并 manage 进 Tauri State，其它子系统（如全局快捷键）按需读取初始值。

mod model;
mod store;

pub use model::*;
pub use store::SettingsStore;

use tauri::{AppHandle, Manager};

use crate::core::Result;

pub fn init(app: &AppHandle) -> Result<Settings> {
    let store = SettingsStore::new(app)?;
    let initial = store.snapshot();
    app.manage(store);
    Ok(initial)
}

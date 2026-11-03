//! WebDAV 备份与同步模块。

pub mod client;
pub mod model;
pub mod scheduler;
pub mod sync;

use tauri::AppHandle;

pub use model::*;
pub use sync::{restore_from_webdav, sync_now, test_connection};

/// 初始化 WebDAV 模块并拉起后台调度器。
pub fn init(app: &AppHandle) {
    scheduler::spawn(app.clone());
    log::info!("webdav backup service initialized");
}

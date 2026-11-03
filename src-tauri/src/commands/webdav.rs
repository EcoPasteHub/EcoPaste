//! WebDAV 相关命令接口。

use tauri::AppHandle;

use crate::core::Result;
use crate::webdav::{WebDavRestoreResult, WebDavSyncResult, WebDavTestResult};

/// 测试 WebDAV 连接与凭证。
#[tauri::command]
pub async fn test_webdav_connection(app: AppHandle) -> Result<WebDavTestResult> {
    crate::webdav::test_connection(&app).await
}

/// 立即执行一次 WebDAV 增量同步与备份。
#[tauri::command]
pub async fn sync_webdav_now(app: AppHandle) -> Result<WebDavSyncResult> {
    crate::webdav::sync_now(&app).await
}

/// 从 WebDAV 远端恢复备份。
#[tauri::command]
pub async fn restore_from_webdav(app: AppHandle) -> Result<WebDavRestoreResult> {
    crate::webdav::restore_from_webdav(&app).await
}

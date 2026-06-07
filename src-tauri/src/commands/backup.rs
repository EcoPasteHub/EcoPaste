//! 备份相关命令：导出 `.ecopastebak` 与接收壳识别。

use std::path::PathBuf;

use serde::Deserialize;
use tauri::AppHandle;

use crate::backup::{
    BackupContainerMode, BackupReceiveSource, ExportHistoryBackupOptions,
    ExportHistoryBackupResult, ImportHistoryBackupInput, ImportHistoryBackupOptions,
    ImportHistoryBackupResult,
};
use crate::core::Result;
use crate::db::DatabaseState;

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct InspectHistoryBackupInput {
    pub path: String,
    #[serde(default)]
    pub source: Option<BackupReceiveSourceInput>,
}

#[derive(Debug, Clone, Copy, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum BackupReceiveSourceInput {
    DragDrop,
    OpenFile,
}

impl From<BackupReceiveSourceInput> for BackupReceiveSource {
    fn from(value: BackupReceiveSourceInput) -> Self {
        match value {
            BackupReceiveSourceInput::DragDrop => BackupReceiveSource::DragDrop,
            BackupReceiveSourceInput::OpenFile => BackupReceiveSource::OpenFile,
        }
    }
}

/// 导出当前历史数据为 `.ecopastebak` 备份包。
#[tauri::command]
pub async fn export_history_backup(
    app: AppHandle,
    db: tauri::State<'_, DatabaseState>,
    target_path: String,
    options: ExportHistoryBackupOptions,
) -> Result<ExportHistoryBackupResult> {
    let pool = db.pool().await;
    crate::backup::export_history_backup(&app, &pool, target_path, options).await
}

/// 识别备份文件并广播给偏好页；不解密、不导入。
#[tauri::command]
pub async fn inspect_history_backup(
    app: AppHandle,
    input: InspectHistoryBackupInput,
) -> Result<BackupContainerMode> {
    let path = PathBuf::from(input.path);
    let mode = crate::backup::inspect_backup_file(&path)?;
    let source = input
        .source
        .map(BackupReceiveSource::from)
        .unwrap_or(BackupReceiveSource::DragDrop);

    crate::backup::emit_received_backup(&app, path, source)?;

    Ok(mode)
}

/// 导入 `.ecopastebak` 备份包；合并立即写入，覆盖热替换当前数据。
#[tauri::command]
pub async fn import_history_backup(
    app: AppHandle,
    db: tauri::State<'_, DatabaseState>,
    input: ImportHistoryBackupInput,
    options: ImportHistoryBackupOptions,
) -> Result<ImportHistoryBackupResult> {
    crate::backup::import_history_backup(&app, &db, input, options).await
}

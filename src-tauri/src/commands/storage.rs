use std::fs;
use std::path::Path;

use anyhow::Context;
use serde::{Deserialize, Serialize};
use tauri::{AppHandle, Manager};
use tauri_plugin_opener::OpenerExt;

use crate::core::Result;

/// 偏好页侧栏展示的本地存储占用概览。
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct StorageUsage {
    pub total_bytes: u64,
    pub database_bytes: u64,
    pub resources_bytes: u64,
    pub settings_bytes: u64,
}

/// 偏好页允许打开的固定本地目录，避免前端传入任意文件系统路径。
#[derive(Debug, Clone, Copy, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum PreferenceDirectoryTarget {
    Data,
    Logs,
}

/// 统计当前 `env_dir()` 数据目录的递归总占用，并拆分常见分项供侧栏展示。
#[tauri::command]
pub async fn get_storage_usage(app: AppHandle) -> Result<StorageUsage> {
    let total_bytes = dir_size(&crate::core::paths::app_data_dir(&app)?)?;
    let database_bytes = database_bytes(&app)?;
    let resources_bytes = dir_size(&crate::core::paths::resources_dir(&app)?)?;
    let settings_bytes = settings_bytes(&app)?;

    Ok(StorageUsage {
        total_bytes,
        database_bytes,
        resources_bytes,
        settings_bytes,
    })
}

/// 在系统文件管理器中打开偏好页本地数据相关目录。
#[tauri::command]
pub async fn open_preference_directory(
    app: AppHandle,
    target: PreferenceDirectoryTarget,
) -> Result<()> {
    let path = match target {
        PreferenceDirectoryTarget::Data => crate::core::paths::app_data_dir(&app)?,
        PreferenceDirectoryTarget::Logs => app
            .path()
            .app_log_dir()
            .context("failed to resolve app log dir")?,
    };

    fs::create_dir_all(&path).with_context(|| format!("failed to create directory {path:?}"))?;
    let path_str = path
        .to_str()
        .ok_or_else(|| anyhow::Error::msg("directory path is not valid utf-8"))?;

    app.opener()
        .open_path(path_str, None::<&str>)
        .map_err(|err| anyhow::Error::msg(err.to_string()))?;

    Ok(())
}

/// 统计 SQLite 主文件与 WAL / SHM sidecar，反映真实数据库占用。
fn database_bytes(app: &AppHandle) -> Result<u64> {
    let db_path = crate::db::db_path(app)?;
    let mut total = file_size(&db_path)?;

    for suffix in ["-wal", "-shm"] {
        total += file_size(Path::new(&format!("{}{}", db_path.display(), suffix)))?;
    }

    Ok(total)
}

/// 统计设置主文件与备份文件大小。
fn settings_bytes(app: &AppHandle) -> Result<u64> {
    let settings_path = crate::core::paths::app_data_dir(app)?.join("settings.json");
    let backup_path = settings_path.with_extension("json.bak");

    Ok(file_size(&settings_path)? + file_size(&backup_path)?)
}

/// 文件不存在时按 0 处理，避免首次启动时显示错误状态。
fn file_size(path: &Path) -> Result<u64> {
    if !path.exists() {
        return Ok(0);
    }

    Ok(fs::metadata(path)
        .with_context(|| format!("failed to read metadata at {path:?}"))?
        .len())
}

/// 递归统计目录大小；目录不存在时按 0 处理。
fn dir_size(path: &Path) -> Result<u64> {
    if !path.exists() {
        return Ok(0);
    }

    let mut total = 0;
    for entry in
        fs::read_dir(path).with_context(|| format!("failed to read directory at {path:?}"))?
    {
        let entry = entry.with_context(|| format!("failed to read entry under {path:?}"))?;
        let entry_path = entry.path();
        let metadata = entry
            .metadata()
            .with_context(|| format!("failed to read metadata at {entry_path:?}"))?;

        if metadata.is_dir() {
            total += dir_size(&entry_path)?;
            continue;
        }

        total += metadata.len();
    }

    Ok(total)
}

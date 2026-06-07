use std::collections::HashSet;
use std::fs;
use std::path::Path;
#[cfg(test)]
use std::path::PathBuf;

use anyhow::Context;
use serde::{Deserialize, Serialize};
use sqlx::SqlitePool;
use tauri::{AppHandle, Manager};
use tauri_plugin_opener::OpenerExt;

use crate::core::Result;
use crate::db::DatabaseState;

/// 偏好页侧栏展示的本地存储占用概览。
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct StorageUsage {
    pub total_bytes: u64,
    pub database_bytes: u64,
    pub resources_bytes: u64,
    pub settings_bytes: u64,
}

/// 清理本地资源缓存后的结果，用于前端展示与刷新侧栏存储占用。
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct CleanCacheResult {
    pub removed_files: u64,
    pub removed_bytes: u64,
    pub storage_usage: StorageUsage,
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

/// 删除资源目录中不再被历史记录或资源索引引用的文件。
#[tauri::command]
pub async fn clean_resource_cache(
    app: AppHandle,
    db: tauri::State<'_, DatabaseState>,
) -> Result<CleanCacheResult> {
    let pool = db.pool().await;
    let resources_dir = crate::core::paths::resources_dir(&app)?;
    let image_files = referenced_image_files(&pool).await?;
    let app_icon_files = referenced_app_icon_files(&pool).await?;
    let file_icon_files = referenced_file_icon_files(&pool).await?;

    let mut removed = CleanCacheStats::default();
    clean_sharded_files(
        &resources_dir.join("clipboard-images").join("origin"),
        &image_files,
        &mut removed,
    )?;
    clean_sharded_files(
        &resources_dir.join("clipboard-images").join("thumbnails"),
        &image_files,
        &mut removed,
    )?;
    clean_flat_files(
        &resources_dir.join("app-icons"),
        &app_icon_files,
        &mut removed,
    )?;
    clean_flat_files(
        &resources_dir.join("file-icons"),
        &file_icon_files,
        &mut removed,
    )?;

    Ok(CleanCacheResult {
        removed_files: removed.files,
        removed_bytes: removed.bytes,
        storage_usage: get_storage_usage(app).await?,
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

/// 查询仍被 image 历史记录引用的图片文件名。
async fn referenced_image_files(pool: &SqlitePool) -> Result<HashSet<String>> {
    let rows =
        sqlx::query_scalar::<_, String>("SELECT content FROM clipboard_items WHERE kind = 'image'")
            .fetch_all(pool)
            .await
            .context("failed to query referenced image files")?;

    Ok(rows.into_iter().collect())
}

/// 查询仍被来源应用引用的应用图标文件名。
async fn referenced_app_icon_files(pool: &SqlitePool) -> Result<HashSet<String>> {
    let rows = sqlx::query_scalar::<_, String>(
        "SELECT DISTINCT icon_file FROM clipboard_apps WHERE icon_file IS NOT NULL",
    )
    .fetch_all(pool)
    .await
    .context("failed to query referenced app icon files")?;

    Ok(rows.into_iter().collect())
}

/// 查询仍被文件类型图标索引引用的文件名。
async fn referenced_file_icon_files(pool: &SqlitePool) -> Result<HashSet<String>> {
    let rows = sqlx::query_scalar::<_, String>("SELECT DISTINCT icon_file FROM file_type_icons")
        .fetch_all(pool)
        .await
        .context("failed to query referenced file icon files")?;

    Ok(rows.into_iter().collect())
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

#[derive(Default)]
struct CleanCacheStats {
    files: u64,
    bytes: u64,
}

/// 清理平铺目录下没有被引用的文件。
fn clean_flat_files(
    root: &Path,
    referenced_files: &HashSet<String>,
    removed: &mut CleanCacheStats,
) -> Result<()> {
    if !root.exists() {
        return Ok(());
    }

    for entry in
        fs::read_dir(root).with_context(|| format!("failed to read directory at {root:?}"))?
    {
        let entry = entry.with_context(|| format!("failed to read entry under {root:?}"))?;
        let path = entry.path();
        let metadata = entry
            .metadata()
            .with_context(|| format!("failed to read metadata at {path:?}"))?;

        if metadata.is_dir() {
            clean_flat_files(&path, referenced_files, removed)?;
            remove_dir_if_empty(&path);
            continue;
        }

        remove_unreferenced_file(&path, metadata.len(), referenced_files, removed)?;
    }

    remove_dir_if_empty(root);

    Ok(())
}

/// 清理带分片子目录的缓存文件，保留仍被数据库引用的文件名。
fn clean_sharded_files(
    root: &Path,
    referenced_files: &HashSet<String>,
    removed: &mut CleanCacheStats,
) -> Result<()> {
    clean_flat_files(root, referenced_files, removed)
}

/// 文件名不在引用集合中时删除该文件，并累计删除数量与字节数。
fn remove_unreferenced_file(
    path: &Path,
    file_bytes: u64,
    referenced_files: &HashSet<String>,
    removed: &mut CleanCacheStats,
) -> Result<()> {
    let Some(file_name) = path.file_name().and_then(|name| name.to_str()) else {
        return Ok(());
    };
    if referenced_files.contains(file_name) {
        return Ok(());
    }

    fs::remove_file(path).with_context(|| format!("failed to remove cache file {path:?}"))?;
    removed.files += 1;
    removed.bytes += file_bytes;

    Ok(())
}

/// 尽力删除空目录；非空、缺失或无权限时由文件清理主流程处理即可。
fn remove_dir_if_empty(path: &Path) {
    let _ = fs::remove_dir(path);
}

/// 统计设置主文件大小。
fn settings_bytes(app: &AppHandle) -> Result<u64> {
    let settings_path = crate::core::paths::config_dir(app)?.join("settings.json");

    file_size(&settings_path)
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

#[cfg(test)]
mod tests {
    use super::*;

    struct TempDir(PathBuf);

    impl TempDir {
        fn new() -> Self {
            let path =
                std::env::temp_dir().join(format!("ecopaste-clean-cache-{}", uuid::Uuid::new_v4()));
            fs::create_dir_all(&path).unwrap();

            Self(path)
        }

        fn path(&self) -> &Path {
            &self.0
        }
    }

    impl Drop for TempDir {
        fn drop(&mut self) {
            fs::remove_dir_all(&self.0).ok();
        }
    }

    #[test]
    fn clean_flat_files_removes_only_unreferenced_files() {
        let temp = TempDir::new();
        let root = temp.path().join("app-icons");
        fs::create_dir_all(&root).unwrap();
        fs::write(root.join("keep.png"), b"keep").unwrap();
        fs::write(root.join("drop.png"), b"drop-this").unwrap();

        let referenced = HashSet::from(["keep.png".to_string()]);
        let mut removed = CleanCacheStats::default();

        clean_flat_files(&root, &referenced, &mut removed).unwrap();

        assert!(root.join("keep.png").exists());
        assert!(!root.join("drop.png").exists());
        assert_eq!(removed.files, 1);
        assert_eq!(removed.bytes, 9);
    }

    #[test]
    fn clean_sharded_files_removes_empty_shard_directories() {
        let temp = TempDir::new();
        let root = temp.path().join("clipboard-images").join("origin");
        let shard = root.join("ab");
        fs::create_dir_all(&shard).unwrap();
        fs::write(shard.join("abandoned.png"), b"x").unwrap();

        let mut removed = CleanCacheStats::default();

        clean_sharded_files(&root, &HashSet::new(), &mut removed).unwrap();

        assert_eq!(removed.files, 1);
        assert!(!shard.exists());
        assert!(!root.exists());
    }
}

use std::collections::HashSet;
use std::fs;
use std::path::{Path, PathBuf};
use std::sync::{Arc, Mutex};

use anyhow::Context;
use serde::{Deserialize, Serialize};
use sqlx::SqlitePool;
use tauri::{AppHandle, Emitter, Manager};
use tauri_plugin_opener::OpenerExt;

use crate::core::Result;
use crate::db::DatabaseState;
use crate::settings::Settings;

const SETTINGS_UPDATED_EVENT: &str = "settings://updated";
const CLIPBOARD_UPDATED_EVENT: &str = "clipboard://updated";
const STORAGE_CONTENT_DIRS: [&str; 4] = ["db", "resources", "config", "state"];
const CUSTOM_STORAGE_CONTAINER_DIR: &str = "EcoPaste";

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

/// 当前数据目录位置及是否已切到自定义目录。
pub type StorageLocation = crate::core::paths::StorageLocation;

/// 更改或还原数据目录后的刷新结果。
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ChangeStorageLocationResult {
    pub location: StorageLocation,
    pub storage_usage: StorageUsage,
}

/// 偏好页允许打开的固定本地目录，避免前端传入任意文件系统路径。
#[derive(Debug, Clone, Copy, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum PreferenceDirectoryTarget {
    Data,
    Logs,
}

/// 读取当前真实数据目录位置。
#[tauri::command]
pub async fn get_storage_location(app: AppHandle) -> Result<StorageLocation> {
    crate::core::paths::storage_location(&app)
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

/// 将数据目录迁移到用户选择的父目录下，并热切换当前运行时状态。
#[tauri::command]
pub async fn change_storage_location(
    app: AppHandle,
    db: tauri::State<'_, DatabaseState>,
    target_parent_dir: String,
) -> Result<ChangeStorageLocationResult> {
    let target = crate::core::paths::custom_data_dir(Path::new(&target_parent_dir));
    switch_storage_location(app, db.inner(), target).await
}

/// 将数据目录迁回默认位置，并热切换当前运行时状态。
#[tauri::command]
pub async fn reset_storage_location(
    app: AppHandle,
    db: tauri::State<'_, DatabaseState>,
) -> Result<ChangeStorageLocationResult> {
    let target = crate::core::paths::default_data_dir(&app)?;
    switch_storage_location(app, db.inner(), target).await
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

async fn switch_storage_location(
    app: AppHandle,
    db: &DatabaseState,
    target: PathBuf,
) -> Result<ChangeStorageLocationResult> {
    let current = crate::core::paths::app_data_dir(&app)?;
    if current == target {
        return Ok(ChangeStorageLocationResult {
            location: crate::core::paths::storage_location(&app)?,
            storage_usage: get_storage_usage(app).await?,
        });
    }

    reject_nested_storage_move(&current, &target)?;
    if target != crate::core::paths::default_data_dir(&app)? {
        crate::core::paths::validate_storage_target(&target)?;
    }

    let _pause_guard = pause_watcher(&app);
    let switch_error = Arc::new(Mutex::new(None::<String>));
    let switch_error_for_task = switch_error.clone();
    let app_for_db = app.clone();
    let current_for_db = current.clone();
    let target_for_db = target.clone();

    db.close_and_replace(|| async move {
        let switch_result = (|| -> Result<()> {
            copy_storage_data(&current_for_db, &target_for_db)?;
            crate::core::paths::set_app_data_dir(&app_for_db, target_for_db.clone())?;
            Ok(())
        })();

        if let Err(err) = switch_result {
            *switch_error_for_task
                .lock()
                .expect("storage switch error poisoned") = Some(err.to_string());
            crate::core::paths::set_app_data_dir(&app_for_db, current_for_db.clone())?;
            return crate::db::init(&app_for_db).await;
        }

        match crate::db::init(&app_for_db).await {
            Ok(pool) => Ok(pool),
            Err(err) => {
                *switch_error_for_task
                    .lock()
                    .expect("storage switch error poisoned") = Some(err.to_string());
                crate::core::paths::set_app_data_dir(&app_for_db, current_for_db.clone())?;
                crate::db::init(&app_for_db).await
            }
        }
    })
    .await?;

    if let Some(message) = switch_error
        .lock()
        .expect("storage switch error poisoned")
        .take()
    {
        return Err(anyhow::anyhow!(message).into());
    }

    let settings = rebase_storage_states(&app).await?;
    remove_old_storage_data(&app, &current)?;
    emit_settings_updated(&app, &settings);
    emit_clipboard_imported(&app);

    Ok(ChangeStorageLocationResult {
        location: crate::core::paths::storage_location(&app)?,
        storage_usage: get_storage_usage(app).await?,
    })
}

fn reject_nested_storage_move(current: &Path, target: &Path) -> Result<()> {
    if target.starts_with(current) || current.starts_with(target) {
        return Err(anyhow::anyhow!("新旧数据目录不能互相包含").into());
    }

    Ok(())
}

fn copy_storage_data(src: &Path, dst: &Path) -> Result<()> {
    fs::create_dir_all(dst).with_context(|| format!("failed to create storage dir {dst:?}"))?;
    for name in STORAGE_CONTENT_DIRS {
        replace_path(&src.join(name), &dst.join(name))?;
    }
    crate::core::paths::write_storage_identity(dst)?;
    Ok(())
}

fn remove_old_storage_data(app: &AppHandle, old: &Path) -> Result<()> {
    let default = crate::core::paths::default_data_dir(app)?;
    if old == default {
        remove_bootstrap_storage_payload(old)?;
        return Ok(());
    }

    remove_custom_storage_root(old)?;

    Ok(())
}

fn remove_custom_storage_root(old: &Path) -> Result<()> {
    if old.exists() {
        fs::remove_dir_all(old)
            .with_context(|| format!("failed to remove old data dir {old:?}"))?;
    }

    let Some(container) = old.parent() else {
        return Ok(());
    };
    if container.file_name().and_then(|name| name.to_str()) != Some(CUSTOM_STORAGE_CONTAINER_DIR) {
        return Ok(());
    }

    match fs::remove_dir(container) {
        Ok(()) => Ok(()),
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => Ok(()),
        Err(err) if err.kind() == std::io::ErrorKind::DirectoryNotEmpty => Ok(()),
        Err(err) => {
            Err(anyhow::anyhow!("failed to remove old data container {container:?}: {err}").into())
        }
    }
}

fn remove_bootstrap_storage_payload(dir: &Path) -> Result<()> {
    if !dir.exists() {
        return Ok(());
    }

    for entry in
        fs::read_dir(dir).with_context(|| format!("failed to read old data dir {dir:?}"))?
    {
        let entry = entry.with_context(|| format!("failed to read entry under {dir:?}"))?;
        if entry.file_name() == "storage.json" {
            continue;
        }

        remove_path(&entry.path())?;
    }

    Ok(())
}

fn replace_path(src: &Path, dst: &Path) -> Result<()> {
    if dst.exists() {
        remove_path(dst)?;
    }

    if !src.exists() {
        return Ok(());
    }

    if src.is_dir() {
        copy_dir_all(src, dst)?;
        return Ok(());
    }

    if let Some(parent) = dst.parent() {
        fs::create_dir_all(parent)
            .with_context(|| format!("failed to create parent dir {parent:?}"))?;
    }
    fs::copy(src, dst).with_context(|| format!("failed to copy {src:?} to {dst:?}"))?;
    Ok(())
}

fn remove_path(path: &Path) -> Result<()> {
    if path.is_dir() {
        fs::remove_dir_all(path).with_context(|| format!("failed to remove {path:?}"))?;
        return Ok(());
    }

    fs::remove_file(path).with_context(|| format!("failed to remove {path:?}"))?;
    Ok(())
}

fn copy_dir_all(src: &Path, dst: &Path) -> Result<()> {
    fs::create_dir_all(dst).with_context(|| format!("failed to create dir {dst:?}"))?;
    for entry in fs::read_dir(src).with_context(|| format!("failed to read dir {src:?}"))? {
        let entry = entry.with_context(|| format!("failed to read entry under {src:?}"))?;
        let src_path = entry.path();
        let dst_path = dst.join(entry.file_name());
        let metadata = entry
            .metadata()
            .with_context(|| format!("failed to read metadata at {src_path:?}"))?;

        if metadata.is_dir() {
            copy_dir_all(&src_path, &dst_path)?;
            continue;
        }

        fs::copy(&src_path, &dst_path)
            .with_context(|| format!("failed to copy {src_path:?} to {dst_path:?}"))?;
    }
    Ok(())
}

async fn rebase_storage_states(app: &AppHandle) -> Result<Settings> {
    let settings = app.state::<crate::settings::SettingsStore>().rebase(app)?;
    app.state::<crate::window::WindowStateStore>().rebase(app)?;
    app.state::<crate::clipboard::ImageStore>().rebase(app)?;
    app.state::<crate::clipboard::AppIconStore>().rebase(app)?;
    app.state::<crate::clipboard::FileIconStore>().rebase(app)?;

    if let Some(registry) = app.try_state::<crate::clipboard::AppsRegistry>() {
        registry.load_from_db().await?;
    }

    Ok(settings)
}

fn emit_settings_updated(app: &AppHandle, settings: &Settings) {
    if let Err(err) = app.emit(SETTINGS_UPDATED_EVENT, settings) {
        log::warn!("emit settings updated event failed: {err}");
    }
}

fn emit_clipboard_imported(app: &AppHandle) {
    if let Err(err) = app.emit(
        CLIPBOARD_UPDATED_EVENT,
        serde_json::json!({ "imported": true }),
    ) {
        log::warn!("emit clipboard imported event failed: {err}");
    }
}

fn pause_watcher(app: &AppHandle) -> WatcherPauseRestore {
    let pause = app.try_state::<crate::clipboard::WatcherPause>();
    let previous = pause.as_ref().is_some_and(|state| state.is_paused());
    if let Some(state) = pause.as_ref() {
        state.set_paused(true);
    }

    WatcherPauseRestore {
        pause: pause.map(|state| state.inner().clone()),
        previous,
    }
}

struct WatcherPauseRestore {
    pause: Option<crate::clipboard::WatcherPause>,
    previous: bool,
}

impl Drop for WatcherPauseRestore {
    fn drop(&mut self) {
        if let Some(pause) = &self.pause {
            pause.set_paused(self.previous);
        }
    }
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

    #[test]
    fn bootstrap_storage_cleanup_keeps_only_manifest() {
        let temp = TempDir::new();
        fs::write(temp.path().join("storage.json"), "{}").unwrap();
        fs::write(temp.path().join(".ecopaste-storage.json"), "{}").unwrap();
        fs::create_dir_all(temp.path().join("db")).unwrap();
        fs::write(temp.path().join("db").join("clipboard.db"), b"db").unwrap();
        fs::create_dir_all(temp.path().join("resources")).unwrap();
        fs::write(temp.path().join("resources").join("image.png"), b"image").unwrap();

        remove_bootstrap_storage_payload(temp.path()).unwrap();

        assert!(temp.path().join("storage.json").exists());
        assert!(!temp.path().join(".ecopaste-storage.json").exists());
        assert!(!temp.path().join("db").exists());
        assert!(!temp.path().join("resources").exists());
    }

    #[test]
    fn remove_custom_storage_root_deletes_empty_container() {
        let temp = TempDir::new();
        let root = temp.path().join("EcoPaste").join("dev");
        fs::create_dir_all(root.join("config")).unwrap();
        fs::write(root.join("config").join("settings.json"), "{}").unwrap();

        remove_custom_storage_root(&root).unwrap();

        assert!(!root.exists());
        assert!(!temp.path().join("EcoPaste").exists());
    }

    #[test]
    fn remove_custom_storage_root_keeps_container_with_sibling_environment() {
        let temp = TempDir::new();
        let container = temp.path().join("EcoPaste");
        let root = container.join("dev");
        fs::create_dir_all(root.join("config")).unwrap();
        fs::write(root.join("config").join("settings.json"), "{}").unwrap();
        fs::create_dir_all(container.join("prod")).unwrap();

        remove_custom_storage_root(&root).unwrap();

        assert!(!root.exists());
        assert!(container.exists());
        assert!(container.join("prod").exists());
    }
}

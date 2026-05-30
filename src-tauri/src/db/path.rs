use std::path::PathBuf;

use anyhow::Context;
use tauri::{AppHandle, Manager};

use crate::core::Result;

const DB_DIR: &str = "db";
const DB_FILENAME_RELEASE: &str = "clipboard.db";
const DB_FILENAME_DEV: &str = "clipboard.dev.db";

pub fn db_path(app: &AppHandle) -> Result<PathBuf> {
    // 数据库连同 WAL 的两个 sidecar（-wal/-shm）统一归到 db/ 子目录，
    // 与 resources/images 的图片目录对称，避免三个文件散落在 app data 根目录。
    let dir = app
        .path()
        .app_local_data_dir()
        .context("failed to resolve app local data dir")?
        .join(DB_DIR);

    std::fs::create_dir_all(&dir).with_context(|| format!("failed to create db dir at {dir:?}"))?;

    let filename = if cfg!(dev) {
        DB_FILENAME_DEV
    } else {
        DB_FILENAME_RELEASE
    };
    Ok(dir.join(filename))
}

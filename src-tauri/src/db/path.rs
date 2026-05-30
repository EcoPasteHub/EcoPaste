use std::path::PathBuf;

use anyhow::Context;
use tauri::AppHandle;

use crate::core::Result;

const DB_FILENAME: &str = "clipboard.db";

pub fn db_path(app: &AppHandle) -> Result<PathBuf> {
    // 数据库连同 WAL 的两个 sidecar（-wal/-shm）直接落在环境数据目录下。
    // dev/prod 的隔离由 core::paths 的环境子目录（dev/ vs prod/）保证，文件名不再带后缀。
    let dir = crate::core::paths::app_data_dir(app)?;

    std::fs::create_dir_all(&dir).with_context(|| format!("failed to create db dir at {dir:?}"))?;

    Ok(dir.join(DB_FILENAME))
}

use std::path::PathBuf;

use anyhow::Context;
use tauri::{AppHandle, Manager};

use crate::core::Result;

const DB_FILENAME_RELEASE: &str = "clipboard.db";
const DB_FILENAME_DEV: &str = "clipboard.dev.db";

pub fn db_path(app: &AppHandle) -> Result<PathBuf> {
    let dir = app
        .path()
        .app_local_data_dir()
        .context("failed to resolve app local data dir")?;

    std::fs::create_dir_all(&dir)
        .with_context(|| format!("failed to create app local data dir at {dir:?}"))?;

    let filename = if cfg!(dev) {
        DB_FILENAME_DEV
    } else {
        DB_FILENAME_RELEASE
    };
    Ok(dir.join(filename))
}

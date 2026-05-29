use anyhow::Context;
use sqlx::sqlite::{SqliteConnectOptions, SqliteJournalMode, SqlitePoolOptions, SqliteSynchronous};
use sqlx::SqlitePool;
use tauri::AppHandle;

use crate::core::Result;
use crate::db::db_path;

pub async fn init(app: &AppHandle) -> Result<SqlitePool> {
    let path = db_path(app)?;

    let options = SqliteConnectOptions::new()
        .filename(&path)
        .create_if_missing(true)
        .journal_mode(SqliteJournalMode::Wal)
        .synchronous(SqliteSynchronous::Normal)
        .foreign_keys(true);

    let pool = SqlitePoolOptions::new()
        .connect_with(options)
        .await
        .with_context(|| format!("failed to open sqlite database at {path:?}"))?;

    log::info!("sqlite pool ready at {path:?}");
    Ok(pool)
}

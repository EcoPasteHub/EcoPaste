//! file_type_icons 表：按文件类型缓存 icon，避免重复抽取系统 icon。
//!
//! cache_key 生成规则见 `clipboard::icon::get_icon_cache_key`。

use chrono::Utc;
use sqlx::SqlitePool;

use super::models::Platform;
use crate::core::Result;

/// 查询指定 cache_key 的 icon 文件名。
pub async fn get_icon(
    pool: &SqlitePool,
    cache_key: &str,
    platform: Platform,
) -> Result<Option<String>> {
    let row = sqlx::query_as::<_, (String,)>(
        "SELECT icon_file FROM file_type_icons WHERE cache_key = ? AND platform = ?",
    )
    .bind(cache_key)
    .bind(platform)
    .fetch_optional(pool)
    .await
    .map_err(|e| {
        log::error!("query file_type_icons failed: {e}");
        anyhow::anyhow!("{e}")
    })?;
    Ok(row.map(|r| r.0))
}

/// upsert：插入或更新 icon 记录。
pub async fn upsert_icon(
    pool: &SqlitePool,
    cache_key: &str,
    platform: Platform,
    icon_file: &str,
) -> Result<()> {
    let now = Utc::now();
    sqlx::query(
        "INSERT INTO file_type_icons (cache_key, platform, icon_file, created_at, updated_at)
         VALUES (?, ?, ?, ?, ?)
         ON CONFLICT (cache_key, platform) DO UPDATE SET
             icon_file = excluded.icon_file,
             updated_at = excluded.updated_at",
    )
    .bind(cache_key)
    .bind(platform)
    .bind(icon_file)
    .bind(now)
    .bind(now)
    .execute(pool)
    .await
    .map_err(|e| {
        log::error!("upsert file_type_icons failed: {e}");
        anyhow::anyhow!("{e}")
    })?;
    Ok(())
}

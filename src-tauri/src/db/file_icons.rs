//! file_type_icons 表：按文件类型缓存 icon，避免重复抽取系统 icon。
//!
//! cache_key 生成规则见 `clipboard::icon::get_icon_cache_key`。

use chrono::Utc;
use sqlx::{QueryBuilder, Sqlite, SqlitePool};
use std::collections::HashMap;

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

/// 批量查询若干 cache_key 的 icon 文件名，一次 `IN` 查询替代逐路径查询。
///
/// 同一进程内 `platform` 恒定，故只对 `cache_key` 做 `IN`；重复 key 先去重，
/// 结果按 `cache_key → icon_file` 返回，调用方按需取。
pub async fn get_icons(
    pool: &SqlitePool,
    cache_keys: &[String],
    platform: Platform,
) -> Result<HashMap<String, String>> {
    let mut unique: Vec<&String> = Vec::with_capacity(cache_keys.len());
    for key in cache_keys {
        if !unique.contains(&key) {
            unique.push(key);
        }
    }
    if unique.is_empty() {
        return Ok(HashMap::new());
    }

    let mut qb: QueryBuilder<Sqlite> =
        QueryBuilder::new("SELECT cache_key, icon_file FROM file_type_icons WHERE platform = ");
    qb.push_bind(platform).push(" AND cache_key IN (");
    let mut sep = qb.separated(", ");
    for key in unique {
        sep.push_bind(key.clone());
    }
    sep.push_unseparated(")");

    let rows = qb
        .build_query_as::<(String, String)>()
        .fetch_all(pool)
        .await
        .map_err(|e| {
            log::error!("batch query file_type_icons failed: {e}");
            anyhow::anyhow!("{e}")
        })?;

    Ok(rows.into_iter().collect())
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

#[cfg(test)]
mod tests {
    use super::*;
    use crate::db::test_support::memory_pool;

    #[tokio::test]
    async fn batch_get_icons_returns_only_platform_matches() {
        let pool = memory_pool().await;
        upsert_icon(&pool, ".txt", Platform::Windows, "a.png")
            .await
            .unwrap();
        upsert_icon(&pool, ".pdf", Platform::Windows, "b.png")
            .await
            .unwrap();
        // 同 key 异平台：不应被 Windows 的批量查询带出来。
        upsert_icon(&pool, "<dir>", Platform::Macos, "c.png")
            .await
            .unwrap();

        let keys = vec![
            ".txt".to_owned(),
            ".txt".to_owned(),
            ".pdf".to_owned(),
            "<dir>".to_owned(),
            ".missing".to_owned(),
        ];
        let icons = get_icons(&pool, &keys, Platform::Windows).await.unwrap();

        assert_eq!(icons.len(), 2);
        assert_eq!(icons.get(".txt").map(String::as_str), Some("a.png"));
        assert_eq!(icons.get(".pdf").map(String::as_str), Some("b.png"));
    }

    #[tokio::test]
    async fn batch_get_icons_with_empty_keys_skips_query() {
        let pool = memory_pool().await;
        let icons = get_icons(&pool, &[], Platform::Windows).await.unwrap();
        assert!(icons.is_empty());
    }
}

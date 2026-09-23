//! Persistent, disposable image OCR jobs. Tokens prevent late results after clearing/requeuing.

use anyhow::Context;
use serde::Serialize;
use sqlx::{FromRow, SqlitePool};

use crate::core::Result;

#[derive(Debug, FromRow)]
pub struct Job {
    pub item_id: String,
    pub token: String,
    pub content: String,
}

#[derive(Debug, Default, FromRow, Serialize)]
pub struct Counts {
    pub total: i64,
    pub pending: i64,
    pub completed: i64,
    pub failed: i64,
}

/// Enqueue one captured image; already indexed or failed images require explicit historical retry.
pub async fn enqueue(pool: &SqlitePool, id: &str) -> Result<()> {
    sqlx::query("INSERT OR IGNORE INTO image_ocr(item_id, token, created_at, updated_at) SELECT id, lower(hex(randomblob(16))), strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now') FROM clipboard_items WHERE id = ? AND kind = 'image'")
        .bind(id).execute(pool).await.context("enqueue image OCR")?;
    Ok(())
}

/// Explicit backfill queues missing images and retries failures without repeating completed work.
pub async fn enqueue_history(pool: &SqlitePool) -> Result<()> {
    let mut tx = pool.begin().await.context("begin OCR backfill")?;
    sqlx::query("INSERT OR IGNORE INTO image_ocr(item_id, token, created_at, updated_at) SELECT id, lower(hex(randomblob(16))), strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now') FROM clipboard_items WHERE kind = 'image'")
        .execute(&mut *tx).await.context("queue historical images")?;
    sqlx::query("UPDATE image_ocr SET status = 'pending', text = '', token = lower(hex(randomblob(16))), updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE status = 'failed'")
        .execute(&mut *tx).await.context("retry failed OCR jobs")?;
    tx.commit().await.context("commit OCR backfill")?;
    Ok(())
}

pub async fn next_job(pool: &SqlitePool) -> Result<Option<Job>> {
    Ok(sqlx::query_as("SELECT o.item_id, o.token, c.content FROM image_ocr o JOIN clipboard_items c ON c.id=o.item_id WHERE o.status='pending' AND c.kind='image' ORDER BY o.created_at, o.item_id LIMIT 1")
        .fetch_optional(pool).await.context("read next OCR job")?)
}

/// Only update the exact queued attempt. Deletion/clear makes late completion a harmless no-op.
pub async fn finish(pool: &SqlitePool, job: &Job, text: Option<&str>) -> Result<bool> {
    let result = sqlx::query("UPDATE image_ocr SET status = ?, text = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE item_id = ? AND token = ? AND status = 'pending'")
        .bind(if text.is_some() { "completed" } else { "failed" })
        .bind(text.unwrap_or_default()).bind(&job.item_id).bind(&job.token)
        .execute(pool).await.context("complete OCR job")?;
    Ok(result.rows_affected() == 1)
}

pub async fn counts(pool: &SqlitePool) -> Result<Counts> {
    Ok(sqlx::query_as("SELECT COUNT(*) AS total, COALESCE(SUM(status='pending'),0) AS pending, COALESCE(SUM(status='completed'),0) AS completed, COALESCE(SUM(status='failed'),0) AS failed FROM image_ocr")
        .fetch_one(pool).await.context("count OCR jobs")?)
}

pub async fn clear(pool: &SqlitePool) -> Result<()> {
    sqlx::query("DELETE FROM image_ocr")
        .execute(pool)
        .await
        .context("clear OCR index")?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::db::test_support::memory_pool;

    async fn image(pool: &SqlitePool, id: &str) {
        sqlx::query("INSERT INTO clipboard_items(id, kind, content, content_hash, platform, created_at, updated_at) VALUES(?, 'image', 'a.png', ?, 'macos', '2026-01-01', '2026-01-01')")
            .bind(id).bind(id).execute(pool).await.unwrap();
    }

    #[tokio::test]
    async fn upgrade_preserves_existing_history_and_pending_jobs_survive_reopen() {
        let dir = tempfile::tempdir().unwrap();
        let options = sqlx::sqlite::SqliteConnectOptions::new()
            .filename(dir.path().join("test.db"))
            .create_if_missing(true)
            .foreign_keys(true);
        let pool = sqlx::SqlitePool::connect_with(options.clone())
            .await
            .unwrap();
        sqlx::raw_sql(include_str!("../../migrations/0001_init.sql"))
            .execute(&pool)
            .await
            .unwrap();
        image(&pool, "existing").await;
        sqlx::query("UPDATE clipboard_items SET is_favorite=1, is_pinned=1, note='保留备注'")
            .execute(&pool)
            .await
            .unwrap();
        sqlx::raw_sql(include_str!("../../migrations/0002_image_ocr.sql"))
            .execute(&pool)
            .await
            .unwrap();
        enqueue_history(&pool).await.unwrap();
        let original: (String, String, bool, bool, String) = sqlx::query_as(
            "SELECT content, updated_at, is_favorite, is_pinned, note FROM clipboard_items",
        )
        .fetch_one(&pool)
        .await
        .unwrap();
        assert_eq!(
            original,
            (
                "a.png".into(),
                "2026-01-01".into(),
                true,
                true,
                "保留备注".into()
            )
        );
        pool.close().await;
        let reopened = sqlx::SqlitePool::connect_with(options).await.unwrap();
        let job = next_job(&reopened).await.unwrap().unwrap();
        assert_eq!(job.item_id, "existing");
        assert!(finish(&reopened, &job, Some("recognized")).await.unwrap());
        assert_eq!(counts(&reopened).await.unwrap().completed, 1);
    }

    #[tokio::test]
    async fn clear_requeue_and_delete_reject_late_results() {
        let pool = memory_pool().await;
        image(&pool, "image").await;
        enqueue(&pool, "image").await.unwrap();
        let old = next_job(&pool).await.unwrap().unwrap();
        clear(&pool).await.unwrap();
        enqueue(&pool, "image").await.unwrap();
        assert!(!finish(&pool, &old, Some("obsolete")).await.unwrap());
        let current = next_job(&pool).await.unwrap().unwrap();
        assert!(finish(&pool, &current, Some("invoice")).await.unwrap());
        assert_eq!(counts(&pool).await.unwrap().completed, 1);
        sqlx::query("DELETE FROM clipboard_items WHERE id = 'image'")
            .execute(&pool)
            .await
            .unwrap();
        assert_eq!(counts(&pool).await.unwrap().total, 0);
        let hits: i64 = sqlx::query_scalar(
            "SELECT count(*) FROM image_ocr_fts WHERE image_ocr_fts MATCH 'invoice'",
        )
        .fetch_one(&pool)
        .await
        .unwrap();
        assert_eq!(hits, 0);
    }

    #[tokio::test]
    async fn failed_and_empty_images_do_not_loop_and_backfill_is_explicit() {
        let pool = memory_pool().await;
        image(&pool, "old").await;
        image(&pool, "new").await;
        enqueue(&pool, "new").await.unwrap();
        assert_eq!(counts(&pool).await.unwrap().total, 1);
        let job = next_job(&pool).await.unwrap().unwrap();
        finish(&pool, &job, None).await.unwrap();
        enqueue(&pool, "new").await.unwrap();
        assert!(next_job(&pool).await.unwrap().is_none());
        enqueue_history(&pool).await.unwrap();
        assert_eq!(counts(&pool).await.unwrap().pending, 2);
        let job = next_job(&pool).await.unwrap().unwrap();
        finish(&pool, &job, Some("")).await.unwrap();
        enqueue_history(&pool).await.unwrap();
        assert_eq!(counts(&pool).await.unwrap().completed, 1);
        assert_eq!(counts(&pool).await.unwrap().pending, 1);
    }
}

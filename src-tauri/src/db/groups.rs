use anyhow::Context;
use sqlx::SqlitePool;

use crate::core::Result;
use crate::db::models::ClipboardGroup;

const LIST_GROUPS_SQL: &str =
    "SELECT id, name, sort_order, created_at, updated_at FROM clipboard_groups \
     ORDER BY sort_order ASC, created_at ASC";

/// 列出全部分组，按 `sort_order` 升序（同序时按 `created_at` 兜底，保证顺序稳定）。
pub async fn list_groups(pool: &SqlitePool) -> Result<Vec<ClipboardGroup>> {
    let groups = sqlx::query_as::<_, ClipboardGroup>(LIST_GROUPS_SQL)
        .fetch_all(pool)
        .await
        .context("failed to list clipboard groups")?;
    Ok(groups)
}

/// 新建分组。
#[allow(dead_code)]
pub async fn insert_group(pool: &SqlitePool, group: &ClipboardGroup) -> Result<()> {
    sqlx::query(
        "INSERT INTO clipboard_groups (id, name, sort_order, created_at, updated_at) \
         VALUES (?, ?, ?, ?, ?)",
    )
    .bind(group.id.as_str())
    .bind(group.name.as_str())
    .bind(group.sort_order)
    .bind(group.created_at)
    .bind(group.updated_at)
    .execute(pool)
    .await
    .context("failed to insert clipboard group")?;
    Ok(())
}

/// 重命名分组。
#[allow(dead_code)]
pub async fn rename_group(pool: &SqlitePool, id: &str, name: &str) -> Result<()> {
    let now = chrono::Utc::now();
    sqlx::query("UPDATE clipboard_groups SET name = ?, updated_at = ? WHERE id = ?")
        .bind(name)
        .bind(now)
        .bind(id)
        .execute(pool)
        .await
        .context("failed to rename clipboard group")?;
    Ok(())
}

/// 删除分组；其下记录的 `group_id` 由外键 `ON DELETE SET NULL` 自动置空（已启用 `foreign_keys`）。
#[allow(dead_code)]
pub async fn delete_group(pool: &SqlitePool, id: &str) -> Result<()> {
    sqlx::query("DELETE FROM clipboard_groups WHERE id = ?")
        .bind(id)
        .execute(pool)
        .await
        .context("failed to delete clipboard group")?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::db::items::{content_hash, find_item_by_id, insert_item};
    use crate::db::models::{ClipboardItem, ClipboardKind, Platform};
    use crate::db::test_support::memory_pool;
    use chrono::DateTime;

    fn sample_group(id: &str, sort_order: i64) -> ClipboardGroup {
        let ts = DateTime::from_timestamp(1_700_000_000, 0).unwrap();
        ClipboardGroup {
            id: id.to_owned(),
            name: format!("name-{id}"),
            sort_order,
            created_at: ts,
            updated_at: ts,
        }
    }

    fn ids(groups: &[ClipboardGroup]) -> Vec<&str> {
        groups.iter().map(|group| group.id.as_str()).collect()
    }

    #[tokio::test]
    async fn insert_and_list_orders_by_sort_order() {
        let pool = memory_pool().await;
        insert_group(&pool, &sample_group("b", 2)).await.unwrap();
        insert_group(&pool, &sample_group("a", 1)).await.unwrap();
        insert_group(&pool, &sample_group("c", 3)).await.unwrap();

        assert_eq!(ids(&list_groups(&pool).await.unwrap()), ["a", "b", "c"]);
    }

    #[tokio::test]
    async fn list_breaks_sort_order_ties_by_created_at() {
        let pool = memory_pool().await;
        let mut first = sample_group("first", 0);
        first.created_at = DateTime::from_timestamp(1_700_000_000, 0).unwrap();
        let mut second = sample_group("second", 0);
        second.created_at = DateTime::from_timestamp(1_700_000_010, 0).unwrap();
        insert_group(&pool, &second).await.unwrap();
        insert_group(&pool, &first).await.unwrap();

        assert_eq!(ids(&list_groups(&pool).await.unwrap()), ["first", "second"]);
    }

    #[tokio::test]
    async fn rename_group_updates_name() {
        let pool = memory_pool().await;
        insert_group(&pool, &sample_group("g", 0)).await.unwrap();

        rename_group(&pool, "g", "renamed").await.unwrap();

        assert_eq!(list_groups(&pool).await.unwrap()[0].name, "renamed");
    }

    #[tokio::test]
    async fn delete_group_removes_it_and_nulls_item_group_id() {
        let pool = memory_pool().await;
        insert_group(&pool, &sample_group("g", 0)).await.unwrap();

        let ts = DateTime::from_timestamp(1_700_000_000, 0).unwrap();
        let item = ClipboardItem {
            id: "i".to_owned(),
            kind: ClipboardKind::Text,
            sub_kind: None,
            group_id: Some("g".to_owned()),
            source_app_id: None,
            content: "content".to_owned(),
            content_hash: content_hash(ClipboardKind::Text, "content"),
            search_text: None,
            summary: None,
            file_types: None,
            size: None,
            width: None,
            height: None,
            use_count: 1,
            is_favorite: false,
            is_pinned: false,
            platform: Platform::Macos,
            note: None,
            created_at: ts,
            updated_at: ts,
            source_app_name: None,
            source_app_icon_file: None,
            source_app_icon_path: None,
            image_thumbnail_path: None,
            file_icon_paths: None,
        };
        insert_item(&pool, &item).await.unwrap();

        delete_group(&pool, "g").await.unwrap();

        assert!(list_groups(&pool).await.unwrap().is_empty());
        // 外键 ON DELETE SET NULL：分组删除后记录仍在，但 group_id 被置空。
        let item = find_item_by_id(&pool, "i").await.unwrap().unwrap();
        assert_eq!(item.group_id, None);
    }
}

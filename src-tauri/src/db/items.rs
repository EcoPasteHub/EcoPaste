use anyhow::Context;
use chrono::Utc;
use sqlx::{QueryBuilder, Sqlite, SqlitePool};

use crate::core::Result;
use crate::db::models::{ClipboardItem, ClipboardItemQuery, ClipboardItemSort};

const SELECT_ITEM: &str = "SELECT id, kind, sub_kind, group_id, content, search_text, size, \
     width, height, use_count, is_favorite, is_pinned, platform, note, created_at, updated_at \
     FROM clipboard_items";

/// 插入一条剪贴板记录（去重/计数由上层在调用前处理，见阶段 1.4）。
pub async fn insert_item(pool: &SqlitePool, item: &ClipboardItem) -> Result<()> {
    sqlx::query(
        "INSERT INTO clipboard_items \
         (id, kind, sub_kind, group_id, content, search_text, size, width, height, \
          use_count, is_favorite, is_pinned, platform, note, created_at, updated_at) \
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
    )
    .bind(item.id.as_str())
    .bind(item.kind)
    .bind(item.sub_kind)
    .bind(item.group_id.as_deref())
    .bind(item.content.as_str())
    .bind(item.search_text.as_deref())
    .bind(item.size)
    .bind(item.width)
    .bind(item.height)
    .bind(item.use_count)
    .bind(item.is_favorite)
    .bind(item.is_pinned)
    .bind(item.platform)
    .bind(item.note.as_deref())
    .bind(item.created_at)
    .bind(item.updated_at)
    .execute(pool)
    .await
    .context("failed to insert clipboard item")?;
    Ok(())
}

/// 统一列表查询：过滤 + 分页 + 排序（置顶项恒前置）。
/// `keyword` 非空时委托 [`search_items_fts`] 走 FTS5，否则普通检索。
pub async fn query_items(pool: &SqlitePool, q: &ClipboardItemQuery) -> Result<Vec<ClipboardItem>> {
    match fts_match_expr(q.keyword.as_deref()) {
        Some(_) => search_items_fts(pool, q).await,
        None => fetch_items(pool, q, None).await,
    }
}

/// 基于 `clipboard_items_fts` 的关键词前缀检索，复用 [`query_items`] 的过滤/分页/排序。
/// 由 [`query_items`] 在 `keyword` 非空时调用；直接调用且 `keyword` 为空时退化为普通查询。
pub async fn search_items_fts(
    pool: &SqlitePool,
    q: &ClipboardItemQuery,
) -> Result<Vec<ClipboardItem>> {
    let match_expr = fts_match_expr(q.keyword.as_deref());
    fetch_items(pool, q, match_expr).await
}

/// 按 `id` 查找单条记录，不存在时返回 `None`。
pub async fn find_item_by_id(pool: &SqlitePool, id: &str) -> Result<Option<ClipboardItem>> {
    let mut qb: QueryBuilder<Sqlite> = QueryBuilder::new(SELECT_ITEM);
    qb.push(" WHERE id = ").push_bind(id.to_owned());

    let item = qb
        .build_query_as::<ClipboardItem>()
        .fetch_optional(pool)
        .await
        .context("failed to find clipboard item by id")?;
    Ok(item)
}

/// 翻转 `is_favorite`（收藏 / 取消收藏）。
pub async fn toggle_item_favorite(pool: &SqlitePool, id: &str) -> Result<()> {
    sqlx::query("UPDATE clipboard_items SET is_favorite = NOT is_favorite WHERE id = ?")
        .bind(id)
        .execute(pool)
        .await
        .context("failed to toggle clipboard item favorite")?;
    Ok(())
}

/// 翻转 `is_pinned`（置顶 / 取消置顶）。
pub async fn toggle_item_pinned(pool: &SqlitePool, id: &str) -> Result<()> {
    sqlx::query("UPDATE clipboard_items SET is_pinned = NOT is_pinned WHERE id = ?")
        .bind(id)
        .execute(pool)
        .await
        .context("failed to toggle clipboard item pinned")?;
    Ok(())
}

/// 更新备注，传 `None` 清空备注。
pub async fn update_item_note(pool: &SqlitePool, id: &str, note: Option<&str>) -> Result<()> {
    sqlx::query("UPDATE clipboard_items SET note = ? WHERE id = ?")
        .bind(note)
        .bind(id)
        .execute(pool)
        .await
        .context("failed to update clipboard item note")?;
    Ok(())
}

/// `use_count + 1` 并刷新 `updated_at`（命中去重时复用，见阶段 1.4）。
pub async fn increment_item_use_count(pool: &SqlitePool, id: &str) -> Result<()> {
    sqlx::query(
        "UPDATE clipboard_items SET use_count = use_count + 1, updated_at = ? WHERE id = ?",
    )
    .bind(Utc::now())
    .bind(id)
    .execute(pool)
    .await
    .context("failed to increment clipboard item use_count")?;
    Ok(())
}

/// 删除单条记录。
pub async fn delete_item(pool: &SqlitePool, id: &str) -> Result<()> {
    sqlx::query("DELETE FROM clipboard_items WHERE id = ?")
        .bind(id)
        .execute(pool)
        .await
        .context("failed to delete clipboard item")?;
    Ok(())
}

/// 批量删除，返回实际删除行数；`ids` 为空时不发查询。
pub async fn delete_items(pool: &SqlitePool, ids: &[String]) -> Result<u64> {
    if ids.is_empty() {
        return Ok(0);
    }

    let mut qb: QueryBuilder<Sqlite> =
        QueryBuilder::new("DELETE FROM clipboard_items WHERE id IN (");
    let mut separated = qb.separated(", ");
    for id in ids {
        separated.push_bind(id);
    }
    qb.push(")");

    let result = qb
        .build()
        .execute(pool)
        .await
        .context("failed to delete clipboard items")?;
    Ok(result.rows_affected())
}

/// 清空全部记录，返回删除行数；`keep_favorite` 为真时保留收藏项。
pub async fn clear_items(pool: &SqlitePool, keep_favorite: bool) -> Result<u64> {
    let mut qb: QueryBuilder<Sqlite> = QueryBuilder::new("DELETE FROM clipboard_items");
    if keep_favorite {
        qb.push(" WHERE is_favorite = 0");
    }

    let result = qb
        .build()
        .execute(pool)
        .await
        .context("failed to clear clipboard items")?;
    Ok(result.rows_affected())
}

/// 把用户关键词拆成 FTS5 前缀匹配表达式（如 `foo bar` -> `"foo"* "bar"*`）。
/// 双引号包裹 + 转义，避免关键词中的 FTS5 语法字符被当作运算符。空白关键词返回 `None`。
fn fts_match_expr(keyword: Option<&str>) -> Option<String> {
    let keyword = keyword?.trim();
    if keyword.is_empty() {
        return None;
    }

    let expr = keyword
        .split_whitespace()
        .map(|token| format!("\"{}\"*", token.replace('"', "\"\"")))
        .collect::<Vec<_>>()
        .join(" ");

    (!expr.is_empty()).then_some(expr)
}

/// 拼装查询：过滤（含可选 FTS 匹配） + 排序（置顶恒前置） + 分页。
/// 所有 bind 均传入拥有所有权/Copy 的值，避免 `QueryBuilder` 借用 `q` 引发的生命周期问题。
async fn fetch_items(
    pool: &SqlitePool,
    q: &ClipboardItemQuery,
    match_expr: Option<String>,
) -> Result<Vec<ClipboardItem>> {
    let mut qb: QueryBuilder<Sqlite> = QueryBuilder::new(SELECT_ITEM);
    qb.push(" WHERE 1 = 1");

    if let Some(expr) = match_expr {
        qb.push(
            " AND rowid IN (SELECT rowid FROM clipboard_items_fts WHERE clipboard_items_fts MATCH ",
        )
        .push_bind(expr)
        .push(")");
    }
    if let Some(kind) = q.kind {
        qb.push(" AND kind = ").push_bind(kind);
    }
    if let Some(group_id) = &q.group_id {
        qb.push(" AND group_id = ").push_bind(group_id.clone());
    }
    if let Some(favorite) = q.favorite {
        qb.push(" AND is_favorite = ").push_bind(favorite);
    }
    if let Some(pinned) = q.pinned {
        qb.push(" AND is_pinned = ").push_bind(pinned);
    }

    qb.push(" ORDER BY is_pinned DESC, ");
    match q.sort {
        ClipboardItemSort::CreatedAtDesc => {
            qb.push("created_at DESC");
        }
        ClipboardItemSort::UseCountDesc => {
            qb.push("use_count DESC, created_at DESC");
        }
    }

    qb.push(" LIMIT ").push_bind(q.limit);
    qb.push(" OFFSET ").push_bind(q.offset);

    let items = qb
        .build_query_as::<ClipboardItem>()
        .fetch_all(pool)
        .await
        .context("failed to query clipboard items")?;
    Ok(items)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::db::groups::insert_group;
    use crate::db::models::{ClipboardGroup, ClipboardKind, Platform};
    use crate::db::test_support::memory_pool;
    use chrono::DateTime;

    fn sample_item(id: &str) -> ClipboardItem {
        let ts = DateTime::from_timestamp(1_700_000_000, 0).unwrap();
        ClipboardItem {
            id: id.to_owned(),
            kind: ClipboardKind::Text,
            sub_kind: None,
            group_id: None,
            content: format!("content-{id}"),
            search_text: None,
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
        }
    }

    fn ids(items: &[ClipboardItem]) -> Vec<&str> {
        items.iter().map(|item| item.id.as_str()).collect()
    }

    #[tokio::test]
    async fn insert_and_find_by_id_roundtrip() {
        let pool = memory_pool().await;
        insert_item(&pool, &sample_item("a")).await.unwrap();

        let found = find_item_by_id(&pool, "a")
            .await
            .unwrap()
            .expect("item should exist");
        assert_eq!(found.id, "a");
        assert_eq!(found.content, "content-a");
        assert_eq!(found.kind, ClipboardKind::Text);
        assert_eq!(found.use_count, 1);

        assert!(find_item_by_id(&pool, "missing").await.unwrap().is_none());
    }

    #[tokio::test]
    async fn query_filters_by_kind() {
        let pool = memory_pool().await;
        let mut text = sample_item("text");
        text.kind = ClipboardKind::Text;
        let mut image = sample_item("image");
        image.kind = ClipboardKind::Image;
        insert_item(&pool, &text).await.unwrap();
        insert_item(&pool, &image).await.unwrap();

        let q = ClipboardItemQuery {
            kind: Some(ClipboardKind::Image),
            ..Default::default()
        };
        assert_eq!(ids(&query_items(&pool, &q).await.unwrap()), ["image"]);
    }

    #[tokio::test]
    async fn query_filters_by_favorite_and_pinned() {
        let pool = memory_pool().await;
        let plain = sample_item("plain");
        let mut fav = sample_item("fav");
        fav.is_favorite = true;
        let mut pin = sample_item("pin");
        pin.is_pinned = true;
        for item in [&plain, &fav, &pin] {
            insert_item(&pool, item).await.unwrap();
        }

        let favs = query_items(
            &pool,
            &ClipboardItemQuery {
                favorite: Some(true),
                ..Default::default()
            },
        )
        .await
        .unwrap();
        assert_eq!(ids(&favs), ["fav"]);

        let pins = query_items(
            &pool,
            &ClipboardItemQuery {
                pinned: Some(true),
                ..Default::default()
            },
        )
        .await
        .unwrap();
        assert_eq!(ids(&pins), ["pin"]);
    }

    #[tokio::test]
    async fn query_filters_by_group() {
        let pool = memory_pool().await;
        let group = ClipboardGroup {
            id: "g1".to_owned(),
            name: "G1".to_owned(),
            sort_order: 0,
            created_at: DateTime::from_timestamp(1_700_000_000, 0).unwrap(),
        };
        insert_group(&pool, &group).await.unwrap();

        let mut grouped = sample_item("grouped");
        grouped.group_id = Some("g1".to_owned());
        insert_item(&pool, &grouped).await.unwrap();
        insert_item(&pool, &sample_item("ungrouped")).await.unwrap();

        let q = ClipboardItemQuery {
            group_id: Some("g1".to_owned()),
            ..Default::default()
        };
        assert_eq!(ids(&query_items(&pool, &q).await.unwrap()), ["grouped"]);
    }

    #[tokio::test]
    async fn query_orders_pinned_first_then_by_sort() {
        let pool = memory_pool().await;
        let mut a = sample_item("a");
        a.created_at = DateTime::from_timestamp(1_700_000_000, 0).unwrap();
        a.use_count = 9;
        let mut b = sample_item("b");
        b.created_at = DateTime::from_timestamp(1_700_000_010, 0).unwrap();
        b.is_pinned = true;
        b.use_count = 1;
        let mut c = sample_item("c");
        c.created_at = DateTime::from_timestamp(1_700_000_020, 0).unwrap();
        c.use_count = 5;
        for item in [&a, &b, &c] {
            insert_item(&pool, item).await.unwrap();
        }

        // 置顶项 b 恒前置；其余按时间倒序 c(20) > a(0)。
        let by_time = query_items(
            &pool,
            &ClipboardItemQuery {
                sort: ClipboardItemSort::CreatedAtDesc,
                ..Default::default()
            },
        )
        .await
        .unwrap();
        assert_eq!(ids(&by_time), ["b", "c", "a"]);

        // 置顶项 b 恒前置；其余按使用次数倒序 a(9) > c(5)。
        let by_use = query_items(
            &pool,
            &ClipboardItemQuery {
                sort: ClipboardItemSort::UseCountDesc,
                ..Default::default()
            },
        )
        .await
        .unwrap();
        assert_eq!(ids(&by_use), ["b", "a", "c"]);
    }

    #[tokio::test]
    async fn query_paginates_with_limit_and_offset() {
        let pool = memory_pool().await;
        for n in 0..5i64 {
            let mut item = sample_item(&format!("id{n}"));
            item.created_at = DateTime::from_timestamp(1_700_000_000 + n, 0).unwrap();
            insert_item(&pool, &item).await.unwrap();
        }

        let page = |limit, offset| ClipboardItemQuery {
            limit,
            offset,
            ..Default::default()
        };
        // created_at 倒序：id4, id3, id2, id1, id0
        assert_eq!(
            ids(&query_items(&pool, &page(2, 0)).await.unwrap()),
            ["id4", "id3"]
        );
        assert_eq!(
            ids(&query_items(&pool, &page(2, 2)).await.unwrap()),
            ["id2", "id1"]
        );
        assert_eq!(
            ids(&query_items(&pool, &page(2, 4)).await.unwrap()),
            ["id0"]
        );
    }

    #[tokio::test]
    async fn search_fts_matches_prefix_across_columns() {
        let pool = memory_pool().await;
        let mut a = sample_item("a");
        a.content = "hello rustacean".to_owned();
        let mut b = sample_item("b");
        b.content = "goodbye world".to_owned();
        b.search_text = Some("searchable token".to_owned());
        let mut c = sample_item("c");
        c.content = "plain".to_owned();
        c.note = Some("annotated".to_owned());
        for item in [&a, &b, &c] {
            insert_item(&pool, item).await.unwrap();
        }

        let search = |kw: &str| ClipboardItemQuery {
            keyword: Some(kw.to_owned()),
            ..Default::default()
        };
        assert_eq!(
            ids(&query_items(&pool, &search("rust")).await.unwrap()),
            ["a"]
        );
        assert_eq!(
            ids(&query_items(&pool, &search("token")).await.unwrap()),
            ["b"]
        );
        assert_eq!(
            ids(&query_items(&pool, &search("annot")).await.unwrap()),
            ["c"]
        );
        assert!(query_items(&pool, &search("zzz")).await.unwrap().is_empty());
    }

    #[tokio::test]
    async fn search_fts_still_applies_filters() {
        let pool = memory_pool().await;
        let mut x = sample_item("x");
        x.content = "shared text".to_owned();
        x.is_favorite = true;
        let mut y = sample_item("y");
        y.content = "shared note".to_owned();
        insert_item(&pool, &x).await.unwrap();
        insert_item(&pool, &y).await.unwrap();

        let q = ClipboardItemQuery {
            keyword: Some("shared".to_owned()),
            favorite: Some(true),
            ..Default::default()
        };
        assert_eq!(ids(&query_items(&pool, &q).await.unwrap()), ["x"]);
    }

    #[tokio::test]
    async fn toggle_favorite_and_pinned_flip_flags() {
        let pool = memory_pool().await;
        insert_item(&pool, &sample_item("a")).await.unwrap();

        toggle_item_favorite(&pool, "a").await.unwrap();
        assert!(
            find_item_by_id(&pool, "a")
                .await
                .unwrap()
                .unwrap()
                .is_favorite
        );
        toggle_item_favorite(&pool, "a").await.unwrap();
        assert!(
            !find_item_by_id(&pool, "a")
                .await
                .unwrap()
                .unwrap()
                .is_favorite
        );

        toggle_item_pinned(&pool, "a").await.unwrap();
        assert!(
            find_item_by_id(&pool, "a")
                .await
                .unwrap()
                .unwrap()
                .is_pinned
        );
        toggle_item_pinned(&pool, "a").await.unwrap();
        assert!(
            !find_item_by_id(&pool, "a")
                .await
                .unwrap()
                .unwrap()
                .is_pinned
        );
    }

    #[tokio::test]
    async fn update_note_sets_and_clears() {
        let pool = memory_pool().await;
        insert_item(&pool, &sample_item("a")).await.unwrap();

        update_item_note(&pool, "a", Some("my note")).await.unwrap();
        assert_eq!(
            find_item_by_id(&pool, "a")
                .await
                .unwrap()
                .unwrap()
                .note
                .as_deref(),
            Some("my note")
        );

        update_item_note(&pool, "a", None).await.unwrap();
        assert_eq!(
            find_item_by_id(&pool, "a").await.unwrap().unwrap().note,
            None
        );
    }

    #[tokio::test]
    async fn increment_use_count_bumps_count_and_updated_at() {
        let pool = memory_pool().await;
        let item = sample_item("a");
        let original_updated_at = item.updated_at;
        insert_item(&pool, &item).await.unwrap();

        increment_item_use_count(&pool, "a").await.unwrap();

        let after = find_item_by_id(&pool, "a").await.unwrap().unwrap();
        assert_eq!(after.use_count, 2);
        assert!(after.updated_at > original_updated_at);
    }

    #[tokio::test]
    async fn delete_item_removes_row() {
        let pool = memory_pool().await;
        insert_item(&pool, &sample_item("a")).await.unwrap();

        delete_item(&pool, "a").await.unwrap();
        assert!(find_item_by_id(&pool, "a").await.unwrap().is_none());
    }

    #[tokio::test]
    async fn delete_items_returns_count_and_ignores_empty() {
        let pool = memory_pool().await;
        for id in ["a", "b", "c"] {
            insert_item(&pool, &sample_item(id)).await.unwrap();
        }

        assert_eq!(delete_items(&pool, &[]).await.unwrap(), 0);

        let removed = delete_items(
            &pool,
            &["a".to_owned(), "b".to_owned(), "missing".to_owned()],
        )
        .await
        .unwrap();
        assert_eq!(removed, 2);
        assert_eq!(
            ids(&query_items(&pool, &ClipboardItemQuery::default())
                .await
                .unwrap()),
            ["c"]
        );
    }

    #[tokio::test]
    async fn clear_items_optionally_keeps_favorites() {
        let pool = memory_pool().await;
        let mut fav = sample_item("fav");
        fav.is_favorite = true;
        insert_item(&pool, &fav).await.unwrap();
        insert_item(&pool, &sample_item("plain")).await.unwrap();

        assert_eq!(clear_items(&pool, true).await.unwrap(), 1);
        assert_eq!(
            ids(&query_items(&pool, &ClipboardItemQuery::default())
                .await
                .unwrap()),
            ["fav"]
        );

        assert_eq!(clear_items(&pool, false).await.unwrap(), 1);
        assert!(query_items(&pool, &ClipboardItemQuery::default())
            .await
            .unwrap()
            .is_empty());
    }

    #[test]
    fn fts_match_expr_builds_escaped_prefix_terms() {
        assert_eq!(fts_match_expr(None), None);
        assert_eq!(fts_match_expr(Some("   ")), None);
        assert_eq!(fts_match_expr(Some("foo")).as_deref(), Some("\"foo\"*"));
        assert_eq!(
            fts_match_expr(Some("foo bar")).as_deref(),
            Some("\"foo\"* \"bar\"*")
        );
        assert_eq!(fts_match_expr(Some("a\"b")).as_deref(), Some("\"a\"\"b\"*"));
    }
}

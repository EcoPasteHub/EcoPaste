use anyhow::Context;
use chrono::Utc;
use sha2::{Digest, Sha256};
use sqlx::{QueryBuilder, Sqlite, SqlitePool};

use crate::core::Result;
use crate::db::models::{ClipboardItem, ClipboardItemQuery, ClipboardItemSort, ClipboardKind};

const SELECT_ITEM: &str = "SELECT id, kind, sub_kind, group_id, source_app_id, content, \
     content_hash, search_text, file_types, size, width, height, use_count, is_favorite, is_pinned, \
     platform, note, created_at, updated_at FROM clipboard_items";

/// 入库去重的结果：`id` 为生效行的主键（命中时是已有行，未命中时是新插入行），
/// `deduplicated` 表示是否命中了已有内容（命中则只 `use_count + 1` 未插入新行）。
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct UpsertResult {
    pub id: String,
    pub deduplicated: bool,
}

/// 计算去重指纹：`sha256("<kind>:<content>")`。
/// 加 `kind` 前缀，避免 text 与 files 恰好同串内容被误判为重复。
/// text 直接哈希内容串即可；image/files 的 `content` 是落盘引用/路径，
/// 调用方（阶段 2.3，持有原始字节时）可改为对原始内容字节哈希后写入 `content_hash`。
pub fn content_hash(kind: ClipboardKind, content: &str) -> String {
    let mut hasher = Sha256::new();
    hasher.update(kind_tag(kind).as_bytes());
    hasher.update(b":");
    hasher.update(content.as_bytes());
    format!("{:x}", hasher.finalize())
}

fn kind_tag(kind: ClipboardKind) -> &'static str {
    match kind {
        ClipboardKind::Text => "text",
        ClipboardKind::Image => "image",
        ClipboardKind::Files => "files",
    }
}

/// 入库主入口：按 `item.content_hash` 去重。
/// 命中已有记录 → 复用 [`increment_item_use_count`] 累加并刷新 `updated_at`，不插入新行；
/// 未命中 → 调用 [`insert_item`] 插入。返回生效行 id 与是否去重。
pub async fn upsert_item(pool: &SqlitePool, item: &ClipboardItem) -> Result<UpsertResult> {
    if let Some(existing) = find_item_by_content_hash(pool, &item.content_hash).await? {
        increment_item_use_count(pool, &existing.id).await?;
        return Ok(UpsertResult {
            id: existing.id,
            deduplicated: true,
        });
    }

    insert_item(pool, item).await?;
    Ok(UpsertResult {
        id: item.id.clone(),
        deduplicated: false,
    })
}

/// 按 `content_hash` 查最近一条同内容记录（命中 `idx_clipboard_items_content_hash` 索引）。
pub async fn find_item_by_content_hash(
    pool: &SqlitePool,
    hash: &str,
) -> Result<Option<ClipboardItem>> {
    let mut qb: QueryBuilder<Sqlite> = QueryBuilder::new(SELECT_ITEM);
    qb.push(" WHERE content_hash = ")
        .push_bind(hash.to_owned())
        .push(" ORDER BY created_at DESC LIMIT 1");

    let item = qb
        .build_query_as::<ClipboardItem>()
        .fetch_optional(pool)
        .await
        .context("failed to find clipboard item by content_hash")?;
    Ok(item)
}

/// 插入一条剪贴板记录（不做去重；去重请走 [`upsert_item`]）。
pub async fn insert_item(pool: &SqlitePool, item: &ClipboardItem) -> Result<()> {
    sqlx::query(
        "INSERT INTO clipboard_items \
         (id, kind, sub_kind, group_id, source_app_id, content, content_hash, search_text, \
          file_types, size, width, height, use_count, is_favorite, is_pinned, platform, note, \
          created_at, updated_at) \
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
    )
    .bind(item.id.as_str())
    .bind(item.kind)
    .bind(item.sub_kind)
    .bind(item.group_id.as_deref())
    .bind(item.source_app_id.as_deref())
    .bind(item.content.as_str())
    .bind(item.content_hash.as_str())
    .bind(item.search_text.as_deref())
    .bind(item.file_types.as_deref())
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

/// 幂等地将 `is_favorite` 置为 true（已收藏的无变化）。auto-favorite 场景用。
pub async fn mark_item_favorite(pool: &SqlitePool, id: &str) -> Result<()> {
    sqlx::query("UPDATE clipboard_items SET is_favorite = 1 WHERE id = ?")
        .bind(id)
        .execute(pool)
        .await
        .context("failed to mark clipboard item favorite")?;
    Ok(())
}

/// 翻转 `is_pinned`（置顶 / 取消置顶）。
#[allow(dead_code)]
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

/// 删除单条记录。若删除的是图片记录，返回其落盘文件名（`<sha256>.png`），供调用方删图；
/// 否则返回 `None`。记录不存在时也返回 `None`。
///
/// image 去重指纹源自 PNG 字节、落盘文件名即字节 sha256，故库里同图至多一行，
/// 删行后该文件必为孤儿，调用方可直接删，无需引用计数。
pub async fn delete_item(pool: &SqlitePool, id: &str) -> Result<Option<String>> {
    let row = sqlx::query_as::<_, (ClipboardKind, String)>(
        "DELETE FROM clipboard_items WHERE id = ? RETURNING kind, content",
    )
    .bind(id)
    .fetch_optional(pool)
    .await
    .context("failed to delete clipboard item")?;
    Ok(row.and_then(|(kind, content)| image_file_name(kind, content)))
}

/// 取被删行里需要连带删除的图片文件名：kind 为 image 时 `content` 即文件名，否则 `None`。
fn image_file_name(kind: ClipboardKind, content: String) -> Option<String> {
    (kind == ClipboardKind::Image).then_some(content)
}

/// 批量删除，返回实际删除行数；`ids` 为空时不发查询。
#[allow(dead_code)]
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

/// 历史清理的结果：删除行数 + 其中图片记录的落盘文件名（供调用方删图）。
#[derive(Debug, Default)]
pub struct CleanupOutcome {
    pub removed: u64,
    pub image_files: Vec<String>,
}

/// 历史清理：按「时间下限」和「最大条数」删除条目；置顶 / 收藏项一律保留。
/// 返回删除行数与被删图片的文件名。`older_than = None` 跳过时长清理；`max_count = None` 或 `Some(0)` 跳过条数清理。
pub async fn cleanup_history(
    pool: &SqlitePool,
    older_than: Option<chrono::DateTime<chrono::Utc>>,
    max_count: Option<u32>,
) -> Result<CleanupOutcome> {
    let mut outcome = CleanupOutcome::default();

    if let Some(cutoff) = older_than {
        let rows = sqlx::query_as::<_, (ClipboardKind, String)>(
            "DELETE FROM clipboard_items \
             WHERE is_pinned = 0 AND is_favorite = 0 AND created_at < ? \
             RETURNING kind, content",
        )
        .bind(cutoff)
        .fetch_all(pool)
        .await
        .context("failed to cleanup clipboard items by retention")?;
        absorb_deleted(&mut outcome, rows);
    }

    if let Some(max) = max_count.filter(|n| *n > 0) {
        // 仅在非置顶 / 非收藏集合内按 created_at DESC 保留前 max 条，多余的删除。
        // SQLite 中 `LIMIT -1 OFFSET n` 表示「跳过前 n 条，剩下全要」。
        let rows = sqlx::query_as::<_, (ClipboardKind, String)>(
            "DELETE FROM clipboard_items WHERE id IN ( \
                 SELECT id FROM clipboard_items \
                 WHERE is_pinned = 0 AND is_favorite = 0 \
                 ORDER BY created_at DESC \
                 LIMIT -1 OFFSET ? \
             ) \
             RETURNING kind, content",
        )
        .bind(max as i64)
        .fetch_all(pool)
        .await
        .context("failed to cleanup clipboard items by max_count")?;
        absorb_deleted(&mut outcome, rows);
    }

    Ok(outcome)
}

/// 把一批被删行计入 outcome：累加行数，并收集其中的图片文件名。
fn absorb_deleted(outcome: &mut CleanupOutcome, rows: Vec<(ClipboardKind, String)>) {
    outcome.removed += rows.len() as u64;
    outcome.image_files.extend(
        rows.into_iter()
            .filter_map(|(kind, content)| image_file_name(kind, content)),
    );
}

/// 清空全部记录，返回删除行数；`keep_favorite` 为真时保留收藏项。
#[allow(dead_code)]
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
        let content = format!("content-{id}");
        ClipboardItem {
            id: id.to_owned(),
            kind: ClipboardKind::Text,
            sub_kind: None,
            group_id: None,
            source_app_id: None,
            content_hash: content_hash(ClipboardKind::Text, &content),
            content,
            search_text: None,
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
    async fn upsert_inserts_when_content_is_new() {
        let pool = memory_pool().await;
        let result = upsert_item(&pool, &sample_item("a")).await.unwrap();
        assert_eq!(
            result,
            UpsertResult {
                id: "a".to_owned(),
                deduplicated: false,
            }
        );
        assert_eq!(
            find_item_by_id(&pool, "a")
                .await
                .unwrap()
                .unwrap()
                .use_count,
            1
        );
    }

    #[tokio::test]
    async fn upsert_dedups_same_content_bumping_use_count() {
        let pool = memory_pool().await;
        let first = sample_item("first");
        upsert_item(&pool, &first).await.unwrap();

        // 不同 id，但内容相同 → content_hash 相同 → 命中去重，不插入新行。
        let mut dup = sample_item("second");
        dup.content = first.content.clone();
        dup.content_hash = first.content_hash.clone();
        let result = upsert_item(&pool, &dup).await.unwrap();

        assert_eq!(
            result,
            UpsertResult {
                id: "first".to_owned(),
                deduplicated: true,
            }
        );
        // 仍只有一行，且命中行 use_count 累加到 2。
        let all = query_items(&pool, &ClipboardItemQuery::default())
            .await
            .unwrap();
        assert_eq!(ids(&all), ["first"]);
        assert_eq!(all[0].use_count, 2);
        assert!(find_item_by_id(&pool, "second").await.unwrap().is_none());
    }

    #[tokio::test]
    async fn upsert_keeps_distinct_content_separate() {
        let pool = memory_pool().await;
        upsert_item(&pool, &sample_item("a")).await.unwrap();
        upsert_item(&pool, &sample_item("b")).await.unwrap();

        let all = query_items(&pool, &ClipboardItemQuery::default())
            .await
            .unwrap();
        assert_eq!(all.len(), 2);
    }

    #[test]
    fn content_hash_is_stable_and_kind_scoped() {
        // 同 kind 同内容 → 同哈希（稳定、可去重）。
        assert_eq!(
            content_hash(ClipboardKind::Text, "hello"),
            content_hash(ClipboardKind::Text, "hello")
        );
        // 内容不同 → 哈希不同。
        assert_ne!(
            content_hash(ClipboardKind::Text, "hello"),
            content_hash(ClipboardKind::Text, "world")
        );
        // 内容相同但 kind 不同 → 哈希不同（kind 前缀隔离）。
        assert_ne!(
            content_hash(ClipboardKind::Text, "same"),
            content_hash(ClipboardKind::Files, "same")
        );
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

        // 文本行删除：无图片文件名返回。
        assert_eq!(delete_item(&pool, "a").await.unwrap(), None);
        assert!(find_item_by_id(&pool, "a").await.unwrap().is_none());

        // 记录不存在：同样返回 None，不报错。
        assert_eq!(delete_item(&pool, "missing").await.unwrap(), None);
    }

    #[tokio::test]
    async fn delete_image_item_returns_file_name() {
        let pool = memory_pool().await;
        let mut img = sample_item("img");
        img.kind = ClipboardKind::Image;
        img.content = "deadbeef.png".to_owned();
        img.content_hash = content_hash(ClipboardKind::Image, "deadbeef.png");
        insert_item(&pool, &img).await.unwrap();

        // 图片行删除：返回落盘文件名供调用方删图。
        assert_eq!(
            delete_item(&pool, "img").await.unwrap().as_deref(),
            Some("deadbeef.png")
        );
        assert!(find_item_by_id(&pool, "img").await.unwrap().is_none());
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

    #[tokio::test]
    async fn cleanup_history_drops_old_items_keeping_pinned_and_favorite() {
        let pool = memory_pool().await;
        let mk = |id: &str, ts: i64, fav: bool, pin: bool| {
            let mut it = sample_item(id);
            it.created_at = DateTime::from_timestamp(ts, 0).unwrap();
            it.is_favorite = fav;
            it.is_pinned = pin;
            it
        };
        let old_plain = mk("old", 1_000, false, false);
        let old_fav = mk("old-fav", 1_000, true, false);
        let old_pin = mk("old-pin", 1_000, false, true);
        let recent = mk("recent", 9_000, false, false);
        for item in [&old_plain, &old_fav, &old_pin, &recent] {
            insert_item(&pool, item).await.unwrap();
        }

        let cutoff = DateTime::from_timestamp(5_000, 0).unwrap();
        let outcome = cleanup_history(&pool, Some(cutoff), None).await.unwrap();
        assert_eq!(outcome.removed, 1);
        assert!(outcome.image_files.is_empty());

        let all = query_items(&pool, &ClipboardItemQuery::default())
            .await
            .unwrap();
        // 置顶项恒前置；保留集合：old-pin、old-fav、recent。
        assert_eq!(ids(&all), ["old-pin", "recent", "old-fav"]);
    }

    #[tokio::test]
    async fn cleanup_history_enforces_max_count_keeping_pinned_and_favorite() {
        let pool = memory_pool().await;
        // 五条普通项 + 一条收藏 + 一条置顶；max_count = 2 时仅保留两条最新的普通项。
        for n in 0..5i64 {
            let mut it = sample_item(&format!("p{n}"));
            it.created_at = DateTime::from_timestamp(1_000 + n, 0).unwrap();
            insert_item(&pool, &it).await.unwrap();
        }
        let mut fav = sample_item("fav");
        fav.is_favorite = true;
        fav.created_at = DateTime::from_timestamp(500, 0).unwrap();
        insert_item(&pool, &fav).await.unwrap();
        let mut pin = sample_item("pin");
        pin.is_pinned = true;
        pin.created_at = DateTime::from_timestamp(400, 0).unwrap();
        insert_item(&pool, &pin).await.unwrap();

        let outcome = cleanup_history(&pool, None, Some(2)).await.unwrap();
        assert_eq!(outcome.removed, 3); // p0, p1, p2 被删
        assert!(outcome.image_files.is_empty());

        let all = query_items(&pool, &ClipboardItemQuery::default())
            .await
            .unwrap();
        assert_eq!(ids(&all), ["pin", "p4", "p3", "fav"]);
    }

    #[tokio::test]
    async fn cleanup_history_no_op_when_both_disabled() {
        let pool = memory_pool().await;
        insert_item(&pool, &sample_item("a")).await.unwrap();
        assert_eq!(cleanup_history(&pool, None, None).await.unwrap().removed, 0);
        assert_eq!(
            cleanup_history(&pool, None, Some(0)).await.unwrap().removed,
            0
        );
        let all = query_items(&pool, &ClipboardItemQuery::default())
            .await
            .unwrap();
        assert_eq!(ids(&all), ["a"]);
    }

    #[tokio::test]
    async fn cleanup_history_collects_deleted_image_file_names() {
        let pool = memory_pool().await;
        // 一条旧图片 + 一条旧文本，都早于 cutoff；只有图片应进入 image_files。
        let mut img = sample_item("img");
        img.kind = ClipboardKind::Image;
        img.content = "cafe1234.png".to_owned();
        img.content_hash = content_hash(ClipboardKind::Image, "cafe1234.png");
        img.created_at = DateTime::from_timestamp(1_000, 0).unwrap();
        let mut txt = sample_item("txt");
        txt.created_at = DateTime::from_timestamp(1_000, 0).unwrap();
        insert_item(&pool, &img).await.unwrap();
        insert_item(&pool, &txt).await.unwrap();

        let cutoff = DateTime::from_timestamp(5_000, 0).unwrap();
        let outcome = cleanup_history(&pool, Some(cutoff), None).await.unwrap();
        assert_eq!(outcome.removed, 2);
        assert_eq!(outcome.image_files, vec!["cafe1234.png".to_owned()]);
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

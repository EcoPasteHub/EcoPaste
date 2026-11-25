//! 剪贴板列表的内存排序索引。
//!
//! 背景：首屏查询 `ORDER BY is_pinned DESC, updated_at DESC, created_at DESC LIMIT n`
//! 在库里没有可用索引（`migrations/0001_init.sql` 只建了 `content_hash` / `source_app_id`
//! 两个二级索引），执行计划是 `SCAN clipboard_items` + `USE TEMP B-TREE FOR ORDER BY`。
//! 该表的行记录里带着 `content` / `search_text` 两个大字段（本机 16.9k 条实测 49.9 MB +
//! 24.5 MB），为了取排序键必须逐行遍历整表记录，同一句 SQL 实测 160–490 ms。
//!
//! 本模块的做法：在窗口打开之前就把「排序键 + 过滤键」整理进内存，打开时只按主键取
//! 可见的几十行（实测 0.26 ms）。内存里不保留 `content` / `search_text` / `summary` /
//! `note` / `file_types` 等大字段，只留排序、过滤、计数所需的最小列。
//!
//! 三条硬约束：
//! 1. 不改库结构、不加索引、不落盘缓存文件 —— 索引纯内存，进程内重建。
//! 2. 排序必须与 SQL 完全一致 —— 时间戳一律按**原始 TEXT 字节**比较（库内同一列的时间戳
//!    存在 `+00:00` / `.123456+00:00` / `.123456789+00:00` 三种精度，若先解析成时间再比较，
//!    同秒条目的次序会与 SQLite BINARY 排序不一致）。
//! 3. 索引未就绪时一律回落 SQL —— [`ListIndex::page`] 返回 `None`，调用方走原查询路径。

use std::cmp::Ordering;
use std::collections::HashMap;
use std::sync::atomic::{AtomicU64, Ordering as AtomicOrdering};
use std::sync::{Arc, OnceLock, RwLock};
use std::time::Instant;

use sqlx::SqlitePool;

use crate::core::Result;
use crate::db::models::{
    ClipboardGroupFilter, ClipboardItemQuery, ClipboardItemSort, ClipboardKind,
};

const FLAG_FAVORITE: u8 = 1 << 0;
const FLAG_PINNED: u8 = 1 << 1;

const KIND_TEXT: u8 = 0;
const KIND_IMAGE: u8 = 1;
const KIND_FILES: u8 = 2;

const SORTS: [ClipboardItemSort; 3] = [
    ClipboardItemSort::CreatedAt,
    ClipboardItemSort::UpdatedAt,
    ClipboardItemSort::UseCount,
];

/// 删除条数超过该值时改为整份重排，避免「每删一行修一次下标」在清理历史时退化成秒级。
const BULK_REMOVE_THRESHOLD: usize = 64;

/// 索引中的一行：只保留排序键、过滤键与主键。
///
/// `created_at` / `updated_at` 保留库里的 TEXT 原文（用 `Box<str>` 省掉 `String` 的容量字段），
/// `seq` 是行在库里的 rowid，用来给「排序键完全相同」的条目一个稳定次序。
struct Row {
    id: Box<str>,
    created_at: Box<str>,
    updated_at: Box<str>,
    group_id: Option<Box<str>>,
    use_count: i64,
    seq: i64,
    kind: u8,
    flags: u8,
}

impl Row {
    fn is_favorite(&self) -> bool {
        self.flags & FLAG_FAVORITE != 0
    }

    fn is_pinned(&self) -> bool {
        self.flags & FLAG_PINNED != 0
    }
}

/// 五个标准 Tab 的计数：写路径增量维护，查询时 O(1)。
/// 非常规组合（自定义分组 / 仅看置顶）退回一次谓词扫描（17k 条约 40 µs）。
#[derive(Default, Clone, Copy)]
struct Counters {
    all: usize,
    text: usize,
    image: usize,
    files: usize,
    favorite: usize,
}

impl Counters {
    fn apply(&mut self, row: &Row, delta: i64) {
        let step = |value: &mut usize| {
            if delta > 0 {
                *value += 1;
            } else {
                *value = value.saturating_sub(1);
            }
        };

        step(&mut self.all);
        match row.kind {
            KIND_TEXT => step(&mut self.text),
            KIND_IMAGE => step(&mut self.image),
            _ => step(&mut self.files),
        }
        if row.is_favorite() {
            step(&mut self.favorite);
        }
    }

    fn for_tab(&self, tab: ClipboardGroupFilter) -> usize {
        match tab {
            ClipboardGroupFilter::All => self.all,
            ClipboardGroupFilter::Text => self.text,
            ClipboardGroupFilter::Image => self.image,
            ClipboardGroupFilter::Files => self.files,
            ClipboardGroupFilter::Favorite => self.favorite,
        }
    }
}

/// 索引主体：`orders` 里存的是 `rows` 的下标，三份分别按三种排序键预先排好。
#[derive(Default)]
struct Inner {
    rows: Vec<Row>,
    ids: HashMap<Box<str>, u32>,
    order_created: Vec<u32>,
    order_updated: Vec<u32>,
    order_use: Vec<u32>,
    counts: Counters,
    /// 全量载入完成前为 `false`，此间所有查询回落 SQL。
    ready: bool,
    /// 未就绪期间到达的写次数；载入完成时若该值变化，说明快照可能落后，需要再载一次。
    writes_during_load: u64,
}

impl Inner {
    fn order_ref(&self, sort: ClipboardItemSort) -> &[u32] {
        match sort {
            ClipboardItemSort::CreatedAt => &self.order_created,
            ClipboardItemSort::UpdatedAt => &self.order_updated,
            ClipboardItemSort::UseCount => &self.order_use,
        }
    }

    fn from_rows(rows: Vec<Row>) -> Self {
        let mut ids = HashMap::with_capacity(rows.len());
        let mut counts = Counters::default();
        for (slot, row) in rows.iter().enumerate() {
            ids.insert(row.id.clone(), slot as u32);
            counts.apply(row, 1);
        }

        let slots = rows.len() as u32;
        let mut inner = Self {
            rows,
            ids,
            order_created: (0..slots).collect(),
            order_updated: (0..slots).collect(),
            order_use: (0..slots).collect(),
            counts,
            ready: true,
            writes_during_load: 0,
        };

        let Self {
            rows,
            order_created,
            order_updated,
            order_use,
            ..
        } = &mut inner;
        order_created.sort_unstable_by(|&a, &b| {
            cmp(
                ClipboardItemSort::CreatedAt,
                &rows[a as usize],
                &rows[b as usize],
            )
        });
        order_updated.sort_unstable_by(|&a, &b| {
            cmp(
                ClipboardItemSort::UpdatedAt,
                &rows[a as usize],
                &rows[b as usize],
            )
        });
        order_use.sort_unstable_by(|&a, &b| {
            cmp(
                ClipboardItemSort::UseCount,
                &rows[a as usize],
                &rows[b as usize],
            )
        });
        inner
    }

    /// 插入或替换一行，同时修正三份排序向量与计数。
    fn upsert(&mut self, row: Row) {
        let Some(&slot) = self.ids.get(&row.id) else {
            let slot = self.rows.len() as u32;
            self.ids.insert(row.id.clone(), slot);
            self.counts.apply(&row, 1);
            self.rows.push(row);
            for sort in SORTS {
                insert_new(self, sort, slot);
            }
            return;
        };

        // 替换：先用旧行算出它在三份向量里的位置，再原地换行、按位移除、按新键插回。
        let old_positions = {
            let old = &self.rows[slot as usize];
            [
                pos_in(
                    &self.order_created,
                    &self.rows,
                    ClipboardItemSort::CreatedAt,
                    old,
                ),
                pos_in(
                    &self.order_updated,
                    &self.rows,
                    ClipboardItemSort::UpdatedAt,
                    old,
                ),
                pos_in(
                    &self.order_use,
                    &self.rows,
                    ClipboardItemSort::UseCount,
                    old,
                ),
            ]
        };

        self.counts.apply(&self.rows[slot as usize], -1);
        self.counts.apply(&row, 1);
        self.rows[slot as usize] = row;

        for (index, sort) in SORTS.iter().copied().enumerate() {
            let Self {
                rows,
                order_created,
                order_updated,
                order_use,
                ..
            } = self;
            let order = match sort {
                ClipboardItemSort::CreatedAt => order_created,
                ClipboardItemSort::UpdatedAt => order_updated,
                ClipboardItemSort::UseCount => order_use,
            };
            order.remove(old_positions[index]);
            insert_by_key(order, rows, sort, slot);
        }
    }

    /// 按主键删除一行。
    fn remove(&mut self, id: &str) {
        let Some(slot) = self.ids.remove(id) else {
            return;
        };

        self.counts.apply(&self.rows[slot as usize], -1);
        self.rows.remove(slot as usize);

        let Self {
            order_created,
            order_updated,
            order_use,
            ..
        } = self;
        for order in [order_created, order_updated, order_use] {
            if let Some(position) = order.iter().position(|&s| s == slot) {
                order.remove(position);
            }
            for s in order.iter_mut() {
                if *s > slot {
                    *s -= 1;
                }
            }
        }
        for value in self.ids.values_mut() {
            if *value > slot {
                *value -= 1;
            }
        }
    }

    /// 批量删除：逐条删的代价是「每删一行都要修下标、搬内存」，清理上千条会退化成秒级；
    /// 超过阈值时直接过滤后整份重排（17k 行约 20 ms），只做一次。
    fn remove_many(&mut self, ids: &[String]) {
        let doomed: std::collections::HashSet<&str> = ids.iter().map(String::as_str).collect();
        let kept: Vec<Row> = std::mem::take(&mut self.rows)
            .into_iter()
            .filter(|row| !doomed.contains(row.id.as_ref()))
            .collect();
        let writes_during_load = self.writes_during_load;
        *self = Self::from_rows(kept);
        self.writes_during_load = writes_during_load;
    }

    /// 清空全部行（保留就绪状态，计数归零）。单测用。
    #[cfg(test)]
    fn clear(&mut self) {
        let writes_during_load = self.writes_during_load;
        *self = Self {
            ready: true,
            writes_during_load,
            ..Self::default()
        };
    }
}

/// 把 `slot` 插回已有序的向量：二分找到第一个「新行应排在它前面」的位置。
/// 排序键含唯一的 `seq`，因此全序严格，二分命中的位置就是唯一合法位置。
fn insert_by_key(order: &mut Vec<u32>, rows: &[Row], sort: ClipboardItemSort, slot: u32) {
    let position = order
        .partition_point(|&s| cmp(sort, &rows[s as usize], &rows[slot as usize]) == Ordering::Less);
    order.insert(position, slot);
}

/// 新行刚 push 到 `rows` 末尾，插回对应排序向量。
fn insert_new(inner: &mut Inner, sort: ClipboardItemSort, slot: u32) {
    let Inner {
        rows,
        order_created,
        order_updated,
        order_use,
        ..
    } = inner;
    let order = match sort {
        ClipboardItemSort::CreatedAt => order_created,
        ClipboardItemSort::UpdatedAt => order_updated,
        ClipboardItemSort::UseCount => order_use,
    };
    insert_by_key(order, rows, sort, slot);
}

/// 反查某行在排序向量中的位置：按全序二分，命中的位置即它当前所在的下标。
fn pos_in(order: &[u32], rows: &[Row], sort: ClipboardItemSort, target: &Row) -> usize {
    order.partition_point(|&s| cmp(sort, &rows[s as usize], target) == Ordering::Less)
}

/// 与 SQL `ORDER BY` 等价的比较：`is_pinned DESC` 恒为第一键，随后按排序键 DESC
/// （`updated_at` / `use_count` 再带 `created_at DESC` 兜底），最后用 rowid 升序保证全序严格。
fn cmp(sort: ClipboardItemSort, a: &Row, b: &Row) -> Ordering {
    let pinned = b.is_pinned().cmp(&a.is_pinned());
    if pinned != Ordering::Equal {
        return pinned;
    }

    let primary = match sort {
        ClipboardItemSort::CreatedAt => b.created_at.cmp(&a.created_at),
        ClipboardItemSort::UpdatedAt => b.updated_at.cmp(&a.updated_at),
        ClipboardItemSort::UseCount => b.use_count.cmp(&a.use_count),
    };
    if primary != Ordering::Equal {
        return primary;
    }

    if sort != ClipboardItemSort::CreatedAt {
        let secondary = b.created_at.cmp(&a.created_at);
        if secondary != Ordering::Equal {
            return secondary;
        }
    }

    a.seq.cmp(&b.seq)
}

fn kind_code(kind: ClipboardKind) -> u8 {
    match kind {
        ClipboardKind::Text => KIND_TEXT,
        ClipboardKind::Image => KIND_IMAGE,
        ClipboardKind::Files => KIND_FILES,
    }
}

/// 一次索引查询的结果：本页主键（已按排序位排好）与同过滤条件下的总数。
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct IndexPage {
    pub ids: Vec<String>,
    pub total: i64,
}

/// 列表索引句柄。生产环境用进程级单例 [`shared`]；
/// 单测直接 `ListIndex::default()` 建独立实例，互不干扰。
pub struct ListIndex {
    inner: RwLock<Inner>,
}

impl Default for ListIndex {
    fn default() -> Self {
        Self {
            inner: RwLock::new(Inner::default()),
        }
    }
}

impl ListIndex {
    pub fn is_ready(&self) -> bool {
        self.inner.read().expect("list index lock poisoned").ready
    }

    /// 从库里全量重建索引。载入期间到达的写会在结束时被检出，并再载一次（最多三轮）。
    pub async fn rebuild(&self, pool: &SqlitePool) -> Result<usize> {
        let started = Instant::now();
        let mut rows_len = 0;

        for _ in 0..3 {
            let mark = self
                .inner
                .read()
                .expect("list index lock poisoned")
                .writes_during_load;
            let rows = fetch_all_keys(pool).await?;
            rows_len = rows.len();
            let mut rebuilt = Inner::from_rows(rows);

            let mut guard = self.inner.write().expect("list index lock poisoned");
            rebuilt.writes_during_load = guard.writes_during_load;
            if guard.writes_during_load == mark {
                *guard = rebuilt;
                log::info!(
                    "clipboard list index ready: {rows_len} rows in {:.1}ms",
                    started.elapsed().as_secs_f64() * 1000.0
                );
                return Ok(rows_len);
            }
            // 载入期间有写入落库，快照可能不完整：丢弃本轮，重载。
            log::debug!("clipboard list index reload: writes landed during load");
        }

        log::warn!("clipboard list index kept falling behind during load; serving from sql");
        Ok(rows_len)
    }

    /// 未就绪时记一笔「载入期间有写入」，供 rebuild 判断快照是否可用。
    fn note_write_while_loading(&self) {
        let mut inner = self.inner.write().expect("list index lock poisoned");
        if !inner.ready {
            inner.writes_during_load += 1;
        }
    }

    /// 用库里最新的行覆盖索引中的同 id 行，并删除已不存在的 id。
    fn apply_rows(&self, rows: Vec<Row>, requested: &[String]) {
        let mut inner = self.inner.write().expect("list index lock poisoned");
        if !inner.ready {
            inner.writes_during_load += 1;
            return;
        }

        let mut returned: Vec<String> = Vec::with_capacity(rows.len());
        for row in rows {
            returned.push(row.id.to_string());
            inner.upsert(row);
        }
        for id in requested {
            if !returned.iter().any(|seen| seen == id) {
                inner.remove(id);
            }
        }
    }

    fn apply_removed(&self, ids: &[String]) {
        let mut inner = self.inner.write().expect("list index lock poisoned");
        if !inner.ready {
            inner.writes_during_load += 1;
            return;
        }
        if ids.len() > BULK_REMOVE_THRESHOLD {
            inner.remove_many(ids);
            return;
        }
        for id in ids {
            inner.remove(id);
        }
    }

    /// 按查询条件返回本页主键与总数；索引未就绪或带关键词时返回 `None`（调用方回落 SQL）。
    pub fn page(&self, q: &ClipboardItemQuery) -> Option<IndexPage> {
        if !self.can_serve(q) {
            return None;
        }

        let inner = self.inner.read().expect("list index lock poisoned");
        let (effective_kind, effective_favorite) = crate::db::items::effective_kind_favorite(q);
        let kind_filter = effective_kind.map(kind_code);
        let order = inner.order_ref(q.sort);

        let matches = |row: &Row| {
            if let Some(kind) = kind_filter {
                if row.kind != kind {
                    return false;
                }
            }
            if let Some(favorite) = effective_favorite {
                if row.is_favorite() != favorite {
                    return false;
                }
            }
            if let Some(group_id) = &q.group_id {
                match &row.group_id {
                    Some(row_group) if row_group.as_ref() == group_id.as_str() => {}
                    _ => return false,
                }
            }
            if let Some(pinned) = q.pinned {
                if row.is_pinned() != pinned {
                    return false;
                }
            }
            true
        };

        let total = match (q.group, q.group_id.is_some(), q.pinned.is_some()) {
            (Some(tab), false, false) => inner.counts.for_tab(tab) as i64,
            _ => order
                .iter()
                .filter(|&&slot| matches(&inner.rows[slot as usize]))
                .count() as i64,
        };

        let skip = q.offset.max(0) as usize;
        let take = if q.limit < 0 {
            usize::MAX
        } else {
            q.limit as usize
        };
        let mut ids = Vec::with_capacity(take.min(order.len()));
        if take > 0 {
            let mut skipped = 0usize;
            for &slot in order.iter() {
                let row = &inner.rows[slot as usize];
                if !matches(row) {
                    continue;
                }
                if skipped < skip {
                    skipped += 1;
                    continue;
                }
                ids.push(row.id.to_string());
                if ids.len() == take {
                    break;
                }
            }
        }

        Some(IndexPage { ids, total })
    }

    /// 索引能否服务该查询：已就绪且不带关键词。
    pub fn can_serve(&self, q: &ClipboardItemQuery) -> bool {
        self.is_ready()
            && q.keyword
                .as_deref()
                .map(str::trim)
                .is_none_or(|keyword| keyword.is_empty())
    }

    #[cfg(test)]
    fn rows_len(&self) -> usize {
        self.inner
            .read()
            .expect("list index lock poisoned")
            .rows
            .len()
    }
}

// ---------------------------------------------------------------------------
// 进程级单例
// ---------------------------------------------------------------------------

static GLOBAL: OnceLock<Arc<ListIndex>> = OnceLock::new();

/// 列表数据版本号：任何会改变列表内容的事件（行增删改、设置变更、切换数据目录）
/// 都让它自增。列表页缓存以它为唯一有效凭据 —— 版本不符即作废，避免返回旧数据。
///
/// 与索引是否就绪无关：即使索引未安装（单测）或正在重建，版本号仍照常累加。
static REVISION: AtomicU64 = AtomicU64::new(0);

/// 当前列表数据版本号。
pub fn revision() -> u64 {
    REVISION.load(AtomicOrdering::Acquire)
}

/// 主动作废所有按版本号缓存的列表结果。供「设置变更 / 切换数据目录」等
/// 不经过行写路径、但会改变列表渲染结果的场景调用。
pub fn invalidate() {
    REVISION.fetch_add(1, AtomicOrdering::AcqRel);
}

/// 取进程级索引实例；未创建时创建（未载入 → 所有查询回落 SQL）。
pub fn shared() -> &'static Arc<ListIndex> {
    GLOBAL.get_or_init(|| Arc::new(ListIndex::default()))
}

/// 已安装的进程级索引；单测里从未安装时为 `None`，所有 hook 变成空操作。
pub fn installed() -> Option<Arc<ListIndex>> {
    GLOBAL.get().cloned()
}

/// 绑定当前数据库并后台重建索引。启动、「切换数据目录」「备份覆盖导入」后调用；
/// 立即返回，不等载入完成 —— 载入期间查询走 SQL。
pub fn activate(pool: SqlitePool) {
    let index = shared().clone();
    tauri::async_runtime::spawn(async move {
        if let Err(err) = index.rebuild(&pool).await {
            log::warn!("clipboard list index rebuild failed: {err:?}");
        }
    });
}

/// 写路径通知：这些 id 的行已落库，按库里最新值刷新索引
/// （`updated_at` / `use_count` 由 SQL 生成，所以回读而不是在内存里推算）。
pub async fn note_rows_changed(pool: &SqlitePool, ids: &[String]) {
    // 版本号先自增：并发的列表缓存构建即使引用到旧行，也会因版本不符而不被采纳。
    invalidate();
    let Some(index) = installed() else {
        return;
    };
    if ids.is_empty() {
        return;
    }
    if !index.is_ready() {
        index.note_write_while_loading();
        return;
    }

    match fetch_keys_by_ids(pool, ids).await {
        Ok(rows) => index.apply_rows(rows, ids),
        Err(err) => {
            // 拿不到最新键就整份作废，避免继续返回错数据；随后立即后台重建。
            log::warn!("clipboard list index refresh failed: {err:?}");
            index.inner.write().expect("list index lock poisoned").ready = false;

            let pool = pool.clone();
            let index = index.clone();
            tauri::async_runtime::spawn(async move {
                if let Err(err) = index.rebuild(&pool).await {
                    log::warn!("clipboard list index rebuild after failure failed: {err:?}");
                }
            });
        }
    }
}

/// 写路径通知：这些 id 的行已从库里删除。
pub fn note_rows_removed(ids: &[String]) {
    invalidate();
    let Some(index) = installed() else {
        return;
    };
    if ids.is_empty() {
        return;
    }
    index.apply_removed(ids);
}

// ---------------------------------------------------------------------------
// 取数
// ---------------------------------------------------------------------------

type KeyRow = (
    i64,
    String,
    ClipboardKind,
    Option<String>,
    bool,
    bool,
    i64,
    String,
    String,
);

const KEY_COLUMNS: &str = "rowid, id, kind, group_id, is_favorite, is_pinned, use_count, \
     created_at, updated_at";

/// 全量载入用的静态 SQL（排序键 + 过滤键 + 主键，不含任何大字段）。
const SELECT_ALL_KEYS: &str = "SELECT rowid, id, kind, group_id, is_favorite, is_pinned, \
     use_count, created_at, updated_at FROM clipboard_items";

fn into_row(tuple: KeyRow) -> Row {
    let (seq, id, kind, group_id, is_favorite, is_pinned, use_count, created_at, updated_at) =
        tuple;
    let mut flags = 0u8;
    if is_favorite {
        flags |= FLAG_FAVORITE;
    }
    if is_pinned {
        flags |= FLAG_PINNED;
    }

    Row {
        id: id.into_boxed_str(),
        created_at: created_at.into_boxed_str(),
        updated_at: updated_at.into_boxed_str(),
        group_id: group_id.map(String::into_boxed_str),
        use_count,
        seq,
        kind: kind_code(kind),
        flags,
    }
}

async fn fetch_all_keys(pool: &SqlitePool) -> Result<Vec<Row>> {
    let rows = sqlx::query_as::<_, KeyRow>(SELECT_ALL_KEYS)
        .fetch_all(pool)
        .await
        .map_err(|err| {
            log::error!("load clipboard list index keys failed: {err}");
            anyhow::anyhow!("{err}")
        })?;

    Ok(rows.into_iter().map(into_row).collect())
}

async fn fetch_keys_by_ids(pool: &SqlitePool, ids: &[String]) -> Result<Vec<Row>> {
    let mut qb = sqlx::QueryBuilder::<sqlx::Sqlite>::new(format!(
        "SELECT {KEY_COLUMNS} FROM clipboard_items WHERE id IN ("
    ));
    let mut separated = qb.separated(", ");
    for id in ids {
        separated.push_bind(id.clone());
    }
    separated.push_unseparated(")");

    let rows = qb
        .build_query_as::<KeyRow>()
        .fetch_all(pool)
        .await
        .map_err(|err| {
            log::error!("refresh clipboard list index keys failed: {err}");
            anyhow::anyhow!("{err}")
        })?;

    Ok(rows.into_iter().map(into_row).collect())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::db::items::{
        content_hash, delete_item, insert_item, query_items, toggle_item_favorite,
        toggle_item_pinned,
    };
    use crate::db::models::{ClipboardItem, Platform};
    use crate::db::test_support::memory_pool;
    use chrono::DateTime;

    fn item(id: &str, created: i64, updated: i64) -> ClipboardItem {
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
            summary: None,
            file_types: None,
            size: None,
            width: None,
            height: None,
            use_count: 1,
            is_favorite: false,
            is_pinned: false,
            is_sensitive: false,
            platform: Platform::Windows,
            note: None,
            created_at: DateTime::from_timestamp(created, 0).unwrap(),
            updated_at: DateTime::from_timestamp(updated, 0).unwrap(),
            source_app_name: None,
            source_app_icon_file: None,
            source_app_icon_path: None,
            image_thumbnail_path: None,
            file_entries: None,
            files_preview_kind: None,
            available_actions: Vec::new(),
            color_preview: None,
            display_created_at: String::new(),
        }
    }

    fn sql_ids(items: &[ClipboardItem]) -> Vec<String> {
        items.iter().map(|item| item.id.clone()).collect()
    }

    #[tokio::test]
    async fn index_matches_sql_for_every_tab_and_sort() {
        let pool = memory_pool().await;
        for n in 0..12i64 {
            let mut it = item(&format!("id{n}"), 1_700_000_000 + n, 1_700_000_020 - n);
            if n % 3 == 0 {
                it.is_favorite = true;
            }
            if n % 4 == 0 {
                it.is_pinned = true;
            }
            if n % 5 == 0 {
                it.kind = ClipboardKind::Image;
                it.content = format!("image-{n}.png");
                it.content_hash = content_hash(ClipboardKind::Image, &it.content);
            }
            it.use_count = n + 1;
            insert_item(&pool, &it).await.unwrap();
        }

        let index = ListIndex::default();
        index.rebuild(&pool).await.unwrap();

        for tab in [
            ClipboardGroupFilter::All,
            ClipboardGroupFilter::Text,
            ClipboardGroupFilter::Image,
            ClipboardGroupFilter::Files,
            ClipboardGroupFilter::Favorite,
        ] {
            for sort in SORTS {
                for offset in [0i64, 5, 9, 20] {
                    let q = ClipboardItemQuery {
                        group: Some(tab),
                        sort,
                        limit: 4,
                        offset,
                        ..Default::default()
                    };
                    let page = index.page(&q).expect("index should serve this query");
                    let expected = sql_ids(&query_items(&pool, &q).await.unwrap());
                    assert_eq!(
                        page.ids, expected,
                        "tab={tab:?} sort={sort:?} offset={offset}"
                    );

                    let count_q = ClipboardItemQuery {
                        group: Some(tab),
                        sort,
                        limit: 100,
                        offset: 0,
                        ..Default::default()
                    };
                    assert_eq!(
                        page.total,
                        query_items(&pool, &count_q).await.unwrap().len() as i64,
                        "total tab={tab:?} sort={sort:?}"
                    );
                }
            }
        }
    }

    #[tokio::test]
    async fn index_matches_sql_after_mutations() {
        let pool = memory_pool().await;
        insert_item(&pool, &item("a", 1_700_000_000, 1_700_000_000))
            .await
            .unwrap();
        insert_item(&pool, &item("b", 1_700_000_001, 1_700_000_001))
            .await
            .unwrap();

        let index = ListIndex::default();
        index.rebuild(&pool).await.unwrap();

        toggle_item_pinned(&pool, "b").await.unwrap();
        toggle_item_favorite(&pool, "a").await.unwrap();
        let keys = fetch_all_keys(&pool).await.unwrap();
        index.apply_rows(keys, &["a".to_owned(), "b".to_owned()]);

        for sort in SORTS {
            let q = ClipboardItemQuery {
                sort,
                limit: 10,
                ..Default::default()
            };
            let page = index.page(&q).unwrap();
            assert_eq!(
                page.ids,
                sql_ids(&query_items(&pool, &q).await.unwrap()),
                "sort={sort:?}"
            );
        }

        delete_item(&pool, "a").await.unwrap();
        index.apply_removed(&["a".to_owned()]);
        assert_eq!(index.rows_len(), 1);
        let q = ClipboardItemQuery {
            limit: 10,
            ..Default::default()
        };
        assert_eq!(index.page(&q).unwrap().total, 1);
        assert_eq!(index.page(&q).unwrap().ids, ["b"]);
    }

    #[tokio::test]
    async fn index_honours_custom_group_filter() {
        let pool = memory_pool().await;
        let group = crate::db::models::ClipboardGroup {
            id: "g1".to_owned(),
            name: "G1".to_owned(),
            icon: "i-lets-icons:folder".to_owned(),
            is_hidden: false,
            sort_order: 0,
            created_at: DateTime::from_timestamp(1_700_000_000, 0).unwrap(),
            updated_at: DateTime::from_timestamp(1_700_000_000, 0).unwrap(),
        };
        crate::db::groups::insert_group(&pool, &group)
            .await
            .unwrap();

        let mut grouped = item("grouped", 1_700_000_000, 1_700_000_000);
        grouped.group_id = Some("g1".to_owned());
        insert_item(&pool, &grouped).await.unwrap();
        insert_item(&pool, &item("plain", 1_700_000_001, 1_700_000_001))
            .await
            .unwrap();

        let index = ListIndex::default();
        index.rebuild(&pool).await.unwrap();

        let q = ClipboardItemQuery {
            group_id: Some("g1".to_owned()),
            limit: 10,
            ..Default::default()
        };
        let page = index.page(&q).unwrap();
        assert_eq!(page.ids, ["grouped"]);
        assert_eq!(page.total, 1);
        assert_eq!(page.ids, sql_ids(&query_items(&pool, &q).await.unwrap()));
    }

    #[tokio::test]
    async fn index_bulk_remove_matches_sql() {
        let pool = memory_pool().await;
        let mut ids = Vec::new();
        for n in 0..90i64 {
            let it = item(&format!("id{n:02}"), 1_700_000_000 + n, 1_700_000_000 + n);
            insert_item(&pool, &it).await.unwrap();
            ids.push(it.id);
        }

        let index = ListIndex::default();
        index.rebuild(&pool).await.unwrap();

        // 一次删 70 条，走整份重排的批量路径。
        let doomed: Vec<String> = ids.drain(0..70).collect();
        crate::db::items::delete_items(&pool, &doomed)
            .await
            .unwrap();
        crate::db::list_index::note_rows_removed(&doomed);
        index.apply_removed(&doomed);

        assert_eq!(index.rows_len(), 20);
        for sort in SORTS {
            let q = ClipboardItemQuery {
                sort,
                limit: 50,
                ..Default::default()
            };
            assert_eq!(
                index.page(&q).unwrap().ids,
                sql_ids(&query_items(&pool, &q).await.unwrap()),
                "sort={sort:?}"
            );
        }
    }

    #[tokio::test]
    async fn index_rejects_keyword_queries_and_empty_state() {
        let pool = memory_pool().await;
        let index = ListIndex::default();
        index.rebuild(&pool).await.unwrap();

        let keyword = ClipboardItemQuery {
            keyword: Some("foo".to_owned()),
            ..Default::default()
        };
        assert!(index.page(&keyword).is_none());

        let blank = ClipboardItemQuery {
            keyword: Some("   ".to_owned()),
            ..Default::default()
        };
        assert!(index.page(&blank).is_some());
        assert_eq!(index.page(&blank).unwrap().total, 0);
    }

    #[tokio::test]
    async fn index_clears_all_rows() {
        let pool = memory_pool().await;
        insert_item(&pool, &item("a", 1, 1)).await.unwrap();
        let index = ListIndex::default();
        index.rebuild(&pool).await.unwrap();

        index.inner.write().unwrap().clear();
        assert_eq!(index.rows_len(), 0);
        assert!(index
            .page(&ClipboardItemQuery::default())
            .unwrap()
            .ids
            .is_empty());
    }

    #[test]
    fn ordering_prefers_pinned_then_sort_key() {
        let rows = vec![
            into_row((
                1,
                "a".to_owned(),
                ClipboardKind::Text,
                None,
                false,
                false,
                1,
                "2026-01-01T00:00:00+00:00".to_owned(),
                "2026-01-01T00:00:00+00:00".to_owned(),
            )),
            into_row((
                2,
                "b".to_owned(),
                ClipboardKind::Text,
                None,
                false,
                true,
                1,
                "2026-01-01T00:00:00+00:00".to_owned(),
                "2026-01-01T00:00:00+00:00".to_owned(),
            )),
        ];
        let inner = Inner::from_rows(rows);
        assert_eq!(inner.order_updated[0], 1, "pinned row ranks first");
        assert_eq!(inner.order_created[0], 1, "pinned row ranks first");
    }

    /// 用真实库副本量测索引路径与旧 SQL 路径的耗时、并逐项核对结果一致。
    ///
    /// 默认 `#[ignore]`；显式跑：
    /// `ECOPASTE_INDEX_BENCH_DB=<clipboard.db 副本路径> cargo test --lib -- --ignored --nocapture real_database`
    #[tokio::test]
    #[ignore = "需要真实库副本，通过 ECOPASTE_INDEX_BENCH_DB 指定路径"]
    async fn real_database_matches_sql_and_times() {
        use sqlx::sqlite::{SqliteConnectOptions, SqlitePoolOptions};
        use sqlx::SqlitePool;
        use std::str::FromStr;
        use std::time::Instant;

        let Ok(path) = std::env::var("ECOPASTE_INDEX_BENCH_DB") else {
            return;
        };
        let pool: SqlitePool = SqlitePoolOptions::new()
            .max_connections(4)
            .connect_with(
                SqliteConnectOptions::from_str(&format!("sqlite://{path}"))
                    .expect("valid sqlite path")
                    .create_if_missing(false),
            )
            .await
            .expect("open bench database");

        let index = ListIndex::default();
        let loaded = index.rebuild(&pool).await.unwrap();
        println!("index loaded rows: {loaded}");

        let q = ClipboardItemQuery {
            group: Some(ClipboardGroupFilter::All),
            limit: 61,
            offset: 0,
            ..Default::default()
        };

        // 索引路径：内存过滤 + 主键拼装。
        let page = index.page(&q).expect("index ready");
        let ids = page.ids.clone();
        let mut best_page_ms = f64::MAX;
        for _ in 0..20 {
            let started = Instant::now();
            let _ = index.page(&q);
            best_page_ms = best_page_ms.min(started.elapsed().as_secs_f64() * 1000.0);
        }
        let mut best_index_ms = f64::MAX;
        for _ in 0..20 {
            let started = Instant::now();
            let _ = crate::db::items::fetch_items_by_ids(&pool, &ids)
                .await
                .unwrap();
            best_index_ms = best_index_ms.min(started.elapsed().as_secs_f64() * 1000.0);
        }
        println!(
            "index path: memory page {:.3}ms + hydrate {} rows {:.2}ms, total={}",
            best_page_ms,
            ids.len(),
            best_index_ms,
            page.total
        );

        // 索引内存占用（结构体内存 + 字符串堆 + 三份排序下标）。
        let footprint = {
            let inner = index.inner.read().unwrap();
            let inline = inner.rows.len() * std::mem::size_of::<Row>();
            let heap: usize = inner
                .rows
                .iter()
                .map(|row| {
                    row.id.len()
                        + row.created_at.len()
                        + row.updated_at.len()
                        + row.group_id.as_deref().map(str::len).unwrap_or(0)
                })
                .sum();
            let orders =
                (inner.order_created.len() + inner.order_updated.len() + inner.order_use.len())
                    * std::mem::size_of::<u32>();
            inline + heap + orders
        };
        println!(
            "index footprint: {:.2} MB ({} rows, {:.0} B/row)",
            footprint as f64 / (1024.0 * 1024.0),
            loaded,
            footprint as f64 / loaded as f64
        );

        // 旧路径：全局索引未安装 → query_items_page 走 SQL。
        let mut sql_ms = Vec::new();
        for _ in 0..10 {
            let started = Instant::now();
            let _ = crate::db::items::query_items_page(&pool, &q).await.unwrap();
            sql_ms.push(started.elapsed().as_secs_f64() * 1000.0);
        }
        sql_ms.sort_by(|a, b| a.partial_cmp(b).unwrap());
        println!(
            "sql path: best {:.2}ms, median {:.2}ms",
            sql_ms[0],
            sql_ms[sql_ms.len() / 2]
        );

        // 逐项核对：五 Tab × 三排序 × 多 offset。
        let mut mismatches = 0;
        for tab in [
            ClipboardGroupFilter::All,
            ClipboardGroupFilter::Text,
            ClipboardGroupFilter::Image,
            ClipboardGroupFilter::Files,
            ClipboardGroupFilter::Favorite,
        ] {
            for sort in SORTS {
                for offset in [0i64, 30, 90] {
                    let q = ClipboardItemQuery {
                        group: Some(tab),
                        sort,
                        limit: 61,
                        offset,
                        ..Default::default()
                    };
                    let page = index.page(&q).expect("index ready");
                    let sql = crate::db::items::query_items(&pool, &q).await.unwrap();
                    let sql_ids: Vec<String> = sql.iter().map(|it| it.id.clone()).collect();
                    if page.ids != sql_ids {
                        mismatches += 1;
                        println!(
                            "MISMATCH tab={tab:?} sort={sort:?} offset={offset}: index={} sql={}",
                            page.ids.len(),
                            sql_ids.len()
                        );
                    }
                }
            }
        }
        println!("mismatches: {mismatches}");
        assert_eq!(mismatches, 0, "index order must match sql order");
    }
}

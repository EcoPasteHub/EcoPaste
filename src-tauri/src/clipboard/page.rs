//! 列表一页的组装、进程内缓存与「隐藏期预热」。
//!
//! 目标：把「窗口可见 → 列表内容为最新」这段压在单帧内。打开窗口时前端要的
//! 永远只有「当前过滤条件下的第一页」（limit/offset 固定在前 30 条），
//! 而这段组装（DB 取行 + 逐条 enrich + 序列化）原本 100% 发生在可见之后。
//!
//! 做法：
//! 1. 组装结果按「查询指纹（过滤/排序/偏移）+ 数据版本号」放进内存，只留最近一条，
//!    且按最大宽度缓存 —— 首屏 30 条与预取 60 条共用同一份，不会交替击穿；
//! 2. 每次剪贴板写入后，若剪贴板窗口不可见，后台去抖重算这一页 —— 打开时直接命中；
//! 3. 版本号由 `db::list_index` 统一自增，写路径收口处已覆盖，无需逐命令维护。

use std::collections::HashMap;
use std::path::Path;
use std::sync::atomic::{AtomicBool, AtomicU64, Ordering};
use std::sync::{Mutex, MutexGuard, OnceLock};
use std::time::{Duration, Instant};

use chrono::{DateTime, Datelike, Local};
use serde::Serialize;
use sqlx::SqlitePool;
use tauri::{AppHandle, Emitter, Manager};

use super::{
    sanitize_css_color, validate_image_file_name, AppIconStore, FileIconStore, ImageStore,
    DIR_CACHE_KEY,
};
use crate::core::{AppError, Result};
use crate::db::list_index;
use crate::db::models::{
    ClipboardAction, ClipboardItem, ClipboardItemPage, ClipboardItemQuery, ClipboardKind,
    ClipboardSubKind, FileEntry, FilesPreviewKind, Platform,
};
use crate::db::DatabaseState;
use crate::settings::SettingsStore;
use crate::window::CLIPBOARD_WINDOW_LABEL;

/// 隐藏期预热的去抖窗口：连续复制（批量导入 / 连续截图）只在最后一次之后重算一次。
const PREWARM_DEBOUNCE: Duration = Duration::from_millis(60);

/// 预热完成后把整页推给前端，让内容刷新在窗口不可见时就已经完成。
///
/// 这样打开窗口时前端手里的列表已经是最新版本：`window://visibility` 到达后
/// 那条刷新会因为版本号一致而退化为 no-op，连一次 IPC 往返都不需要走。
const PAGE_READY_EVENT: &str = "clipboard://page-ready";

/// 索引尚在构建时，预热会回落成全表扫描（大库实测 280–340ms），因此跳过；
/// 改为每 [`INDEX_WAIT_POLL`] 查一次就绪状态，就绪后补做一次预热，覆盖
/// 「应用刚启动 1–10 秒内打开窗口」这一段原本最慢的窗口。
const INDEX_WAIT_POLL: Duration = Duration::from_millis(300);
const INDEX_WAIT_STEPS: usize = 200;

/// 路径存在性缓存的 TTL：文件条目每次组装都要 `exists()`，跨盘 / 网络路径会阻塞；
/// 同一路径在 TTL 内复用结论，避免每次打开窗口都重复探测磁盘。
const PATH_EXISTS_TTL: Duration = Duration::from_secs(30);
const PATH_EXISTS_CAPACITY: usize = 4096;

// ---------------------------------------------------------------------------
// 页缓存
// ---------------------------------------------------------------------------

/// 查询指纹：过滤 + 排序 + 偏移，**刻意不含 limit**。
///
/// 同一个区间会以不同宽度被请求：打开窗口时前端先要 30 条（`reload`），
/// Virtuoso 随后按预取半径要 60 条（`loadRange`）。两者是同一次查询的不同「视窗宽度」，
/// 若把 limit 算进指纹，会形成「这次 30、下次 60」交替击穿缓存、每次打开各重建两次。
#[derive(Debug, Clone, PartialEq, Eq)]
struct QueryKey {
    kind: Option<ClipboardKind>,
    group_id: Option<String>,
    favorite: Option<bool>,
    pinned: Option<bool>,
    group: Option<crate::db::models::ClipboardGroupFilter>,
    keyword: Option<String>,
    sort: crate::db::models::ClipboardItemSort,
    offset: i64,
}

impl From<&ClipboardItemQuery> for QueryKey {
    fn from(q: &ClipboardItemQuery) -> Self {
        Self {
            kind: q.kind,
            group_id: q.group_id.clone(),
            favorite: q.favorite,
            pinned: q.pinned,
            group: q.group,
            // 关键字只按去掉首尾空白后的值比较：前端输入过程中的空白差异不应击穿缓存。
            keyword: q
                .keyword
                .as_deref()
                .map(str::trim)
                .filter(|k| !k.is_empty())
                .map(str::to_owned),
            sort: q.sort,
            offset: q.offset,
        }
    }
}

/// 偏移 0 处一次组装的最小宽度：同时覆盖「首屏 30 条」与「预取后的 60 条」两种请求。
/// 前端 `PAGE_SIZE = 30`、`PRELOAD_ROWS = 30` 决定了这两个宽度。
const OPEN_CACHE_CAPACITY: i64 = 60;

struct CachedPage {
    key: QueryKey,
    revision: u64,
    page: ClipboardItemPage,
}

impl CachedPage {
    fn capacity(&self) -> i64 {
        self.page.list.len() as i64
    }
}

fn page_cache() -> MutexGuard<'static, Option<CachedPage>> {
    static CACHE: OnceLock<Mutex<Option<CachedPage>>> = OnceLock::new();
    CACHE
        .get_or_init(|| Mutex::new(None))
        .lock()
        .unwrap_or_else(|poisoned| poisoned.into_inner())
}

/// 命中条件：同查询（过滤/排序/偏移）+ 同数据版本 + 缓存宽度够本次请求。
/// 列表本身已到底（`has_more = false`）时宽度不足也算命中——库里没有更多行了。
fn cache_get(key: &QueryKey, revision: u64, offset: i64, limit: i64) -> Option<ClipboardItemPage> {
    let cached = page_cache();
    let entry = cached.as_ref()?;
    if entry.revision != revision || &entry.key != key {
        return None;
    }
    if entry.capacity() >= limit || !entry.page.has_more {
        return Some(truncate_page(&entry.page, offset, limit));
    }
    None
}

fn cache_put(key: &QueryKey, page: &ClipboardItemPage) {
    let mut cached = page_cache();
    if let Some(existing) = cached.as_ref() {
        // 同一页已有更宽的同版本缓存时不缩水；其余情况（不同页/新版本）直接替换。
        if existing.key == *key
            && existing.revision == page.revision
            && existing.capacity() >= page.list.len() as i64
        {
            return;
        }
    }
    *cached = Some(CachedPage {
        key: key.clone(),
        revision: page.revision,
        page: page.clone(),
    });
}

/// 把宽度为 N 的缓存页截成调用方要的 `limit` 条，并按新的长度重算 `has_more`。
fn truncate_page(page: &ClipboardItemPage, offset: i64, limit: i64) -> ClipboardItemPage {
    let take = limit.max(0) as usize;
    let list: Vec<crate::db::models::ClipboardItem> =
        page.list.iter().take(take).cloned().collect();
    let has_more = offset + (list.len() as i64) < page.total;

    ClipboardItemPage {
        list,
        total: page.total,
        has_more,
        revision: page.revision,
    }
}

// ---------------------------------------------------------------------------
// 最近一次查询（预热的对象）
// ---------------------------------------------------------------------------

fn last_query_cell() -> MutexGuard<'static, ClipboardItemQuery> {
    static LAST_QUERY: OnceLock<Mutex<ClipboardItemQuery>> = OnceLock::new();
    LAST_QUERY
        .get_or_init(|| Mutex::new(ClipboardItemQuery::default()))
        .lock()
        .unwrap_or_else(|poisoned| poisoned.into_inner())
}

fn remember_query(query: &ClipboardItemQuery) {
    *last_query_cell() = query.clone();
}

fn last_query() -> ClipboardItemQuery {
    last_query_cell().clone()
}

// ---------------------------------------------------------------------------
// 对外的取页入口
// ---------------------------------------------------------------------------

/// 取列表一页：命中缓存直接返回；未命中则组装，且只在「组装期间没有写入」时写回缓存。
pub async fn load_page(app: &AppHandle, query: ClipboardItemQuery) -> Result<ClipboardItemPage> {
    let key = QueryKey::from(&query);
    let revision = list_index::revision();
    remember_query(&query);

    if let Some(page) = cache_get(&key, revision, query.offset, query.limit) {
        #[cfg(debug_assertions)]
        log::debug!(
            "clipboard page: cache hit (requested={}, served={}, total={}, revision={revision})",
            query.limit,
            page.list.len(),
            page.total
        );
        return Ok(page);
    }

    // 偏移 0 的请求统一按 [`OPEN_CACHE_CAPACITY`] 组装：一次取宽一点，
    // 让随后到来的更大宽度请求也在同一份缓存里命中。
    // 索引未就绪时不做这份加宽 —— 那时组装要回落全表扫描，成本与行数近似线性，
    // 为缓存多取一倍行会在启动期最慢的那次打开上凭空翻倍。
    let index_ready = list_index::installed()
        .map(|index| index.is_ready())
        .unwrap_or(false);
    let build_query = if index_ready && query.offset == 0 && query.limit < OPEN_CACHE_CAPACITY {
        ClipboardItemQuery {
            limit: OPEN_CACHE_CAPACITY,
            ..query.clone()
        }
    } else {
        query.clone()
    };

    #[cfg(debug_assertions)]
    let started = Instant::now();
    let mut page = build_page(app, &build_query).await?;
    let revision_after = list_index::revision();
    page.revision = revision_after;

    // 组装期间发生过写入（revision 变了）：结果可能落后于库，不写缓存，下次重算。
    if revision_after == revision {
        cache_put(&key, &page);
    }

    #[cfg(debug_assertions)]
    log::debug!(
        "clipboard page: built in {:.2}ms (capacity={}, requested={}, total={}, revision={}, cacheable={})",
        started.elapsed().as_secs_f64() * 1000.0,
        page.list.len(),
        query.limit,
        page.total,
        revision_after,
        revision_after == revision
    );

    Ok(truncate_page(&page, query.offset, query.limit))
}

/// 组装一页：DB 取行 + 逐条补齐渲染所需字段。
async fn build_page(app: &AppHandle, query: &ClipboardItemQuery) -> Result<ClipboardItemPage> {
    let pool = app.state::<DatabaseState>().pool().await;
    let image_store = app.state::<ImageStore>();
    let app_icon_store = app.state::<AppIconStore>();
    let file_icon_store = app.state::<FileIconStore>();

    #[cfg(debug_assertions)]
    let started = Instant::now();
    let (mut items, total) = crate::db::items::query_items_page(&pool, query).await?;
    #[cfg(debug_assertions)]
    let db_elapsed = started.elapsed();

    let now = Local::now();
    let settings = app.state::<SettingsStore>().snapshot();
    let file_entry_limit = settings.clipboard.display.file_entry_limit();
    let redact_sensitive = settings.clipboard.sensitive.redact_secrets;
    for item in &mut items {
        attach_image_thumbnail_path(&image_store, item).await?;
        attach_source_app_icon_path(&app_icon_store, item);
        attach_file_entries(&pool, &file_icon_store, item, file_entry_limit).await?;
        attach_color_preview(item);
        attach_display_created_at(item, &now);
        redact_sensitive_list_item(item, redact_sensitive);
        item.available_actions = compute_available_actions(item);
    }
    let has_more = query.offset + (items.len() as i64) < total;

    #[cfg(debug_assertions)]
    log::debug!(
        "clipboard page: db {:.2}ms, enrich {:.2}ms, items={}, total={}",
        db_elapsed.as_secs_f64() * 1000.0,
        (started.elapsed() - db_elapsed).as_secs_f64() * 1000.0,
        items.len(),
        total
    );

    Ok(ClipboardItemPage {
        list: items,
        total,
        has_more,
        revision: 0,
    })
}

// ---------------------------------------------------------------------------
// 隐藏期预热
// ---------------------------------------------------------------------------

/// 剪贴板窗口当前是否可见。可见时前端自己会立刻发起查询，不需要预热。
fn clipboard_window_visible(app: &AppHandle) -> bool {
    app.get_webview_window(CLIPBOARD_WINDOW_LABEL)
        .and_then(|window| window.is_visible().ok())
        .unwrap_or(false)
}

/// 剪贴板写入后调用：窗口隐藏时在后台把「最近一次查询」的页重算一遍并缓存，
/// 使下一次打开窗口的刷新退化为一次纯内存命中。
///
/// 索引未就绪时不当场预热（那时组装会回落全表扫描，本机实测 280–340ms，
/// 不应把这份 IO 搬到后台与索引重建抢同一个库文件），改为预约「就绪后补一次」。
pub fn maybe_prewarm(app: &AppHandle) {
    if clipboard_window_visible(app) {
        return;
    }

    let index_ready = list_index::installed()
        .map(|index| index.is_ready())
        .unwrap_or(false);
    if !index_ready {
        schedule_prewarm_after_index_ready(app);
        return;
    }

    static EPOCH: AtomicU64 = AtomicU64::new(0);
    let epoch = EPOCH.fetch_add(1, Ordering::SeqCst) + 1;

    let app = app.clone();
    tauri::async_runtime::spawn(async move {
        tokio::time::sleep(PREWARM_DEBOUNCE).await;
        // 去抖：期间又有写入，或窗口已被打开，就交给更晚的那次 / 前端自己去算。
        if EPOCH.load(Ordering::SeqCst) != epoch || clipboard_window_visible(&app) {
            return;
        }

        prewarm_once(&app).await;
    });
}

/// 索引构建中时的预约：轮询到就绪（或超时）后补做一次预热。
///
/// 覆盖「应用刚启动、索引还没建好就按了热键」这一段——那是全链路最慢的一档
/// （回落全表扫描 + 宽页组装），也是用户唯一可能明显感到卡顿的时机。
fn schedule_prewarm_after_index_ready(app: &AppHandle) {
    static WAITING: AtomicBool = AtomicBool::new(false);
    // 索引就绪前的写入可能很密集，只排一个等待任务即可。
    if WAITING.swap(true, Ordering::SeqCst) {
        return;
    }

    let app = app.clone();
    tauri::async_runtime::spawn(async move {
        let mut ready = false;
        for _ in 0..INDEX_WAIT_STEPS {
            if list_index::installed()
                .map(|index| index.is_ready())
                .unwrap_or(false)
            {
                ready = true;
                break;
            }
            tokio::time::sleep(INDEX_WAIT_POLL).await;
        }

        WAITING.store(false, Ordering::SeqCst);
        if !ready || clipboard_window_visible(&app) {
            return;
        }

        prewarm_once(&app).await;
    });
}

/// 真算一次「最近一次查询」的页：命中已缓存则近似空转，未命中才组装；
/// 算完若窗口仍不可见，就把整页推给前端，让刷新在隐藏期完成。
async fn prewarm_once(app: &AppHandle) {
    let mut query = last_query();
    // 按缓存宽度组装：同一份结果既能服务首屏 30 条，也能服务随后的 60 条预取。
    if query.offset == 0 && query.limit < OPEN_CACHE_CAPACITY {
        query.limit = OPEN_CACHE_CAPACITY;
    }

    #[cfg(debug_assertions)]
    let started = Instant::now();
    // 这里走的就是普通取页路径：命中已缓存则近乎空转。
    let page = match load_page(app, query.clone()).await {
        Ok(page) => page,
        Err(err) => {
            log::warn!("clipboard page prewarm failed: {err:?}");
            return;
        }
    };

    #[cfg(debug_assertions)]
    let (item_count, revision) = (page.list.len(), page.revision);

    let payload = PageReadyPayload { query, page };
    #[cfg(debug_assertions)]
    let pushed = push_if_hidden(app, payload);
    // release 构建没有那条 debug 日志，返回值无人消费。
    #[cfg(not(debug_assertions))]
    push_if_hidden(app, payload);

    #[cfg(debug_assertions)]
    log::debug!(
        "clipboard page prewarm done in {:.2}ms (items={}, revision={}, pushed={})",
        started.elapsed().as_secs_f64() * 1000.0,
        item_count,
        revision,
        pushed
    );
}

/// 窗口不可见时把整页推给前端。可见时不推：前端自己会拿到更新的数据，
/// 再塞一份整页只会和用户正在看的列表打架。
fn push_if_hidden(app: &AppHandle, payload: PageReadyPayload) -> bool {
    if clipboard_window_visible(app) {
        return false;
    }

    match app.emit(PAGE_READY_EVENT, payload) {
        Ok(()) => true,
        Err(err) => {
            log::warn!("clipboard page ready emit failed: {err:?}");
            false
        }
    }
}

/// 推给前端的预热结果：查询指纹 + 整页，前端据此判断能否直接采纳。
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
struct PageReadyPayload {
    query: ClipboardItemQuery,
    page: ClipboardItemPage,
}

// ---------------------------------------------------------------------------
// 路径存在性缓存
// ---------------------------------------------------------------------------

fn path_exists_cell() -> MutexGuard<'static, HashMap<String, (bool, Instant)>> {
    static EXISTS: OnceLock<Mutex<HashMap<String, (bool, Instant)>>> = OnceLock::new();
    EXISTS
        .get_or_init(|| Mutex::new(HashMap::new()))
        .lock()
        .unwrap_or_else(|poisoned| poisoned.into_inner())
}

/// 带 TTL 的 `Path::exists()`。命中缓存只省磁盘探测，不改变语义：
/// 文件在 TTL 内被外部删除时，列表最多晚 [`PATH_EXISTS_TTL`] 才反映。
fn path_exists_cached(path: &str) -> bool {
    {
        let cache = path_exists_cell();
        if let Some((exists, checked_at)) = cache.get(path) {
            if checked_at.elapsed() < PATH_EXISTS_TTL {
                return *exists;
            }
        }
    }

    let exists = Path::new(path).exists();
    let mut cache = path_exists_cell();
    if cache.len() >= PATH_EXISTS_CAPACITY {
        cache.clear();
    }
    cache.insert(path.to_owned(), (exists, Instant::now()));
    exists
}

// ---------------------------------------------------------------------------
// 逐条补齐字段（列表与单条查询共用）
// ---------------------------------------------------------------------------

/// 为 image 条目补齐缩略图绝对路径，前端可直接渲染。
/// 历史脏数据（非 `<hash>.png`）降级为 `None`。
///
/// 原图本身不存在时不再 spawn 解码任务：读了也必然失败，只会白占一个 blocking 线程。
pub(crate) async fn attach_image_thumbnail_path(
    store: &ImageStore,
    item: &mut ClipboardItem,
) -> Result<()> {
    if item.kind != ClipboardKind::Image {
        return Ok(());
    }

    if validate_image_file_name(&item.content).is_err() {
        item.image_thumbnail_path = None;
        return Ok(());
    }

    let file_name = item.content.clone();
    let thumb_path = store.thumbnail_path(&file_name);
    let thumb_exists = thumb_path.exists();

    // 缩略图缺失时先指向原图，保证前端立刻有东西可渲染；后台再补缩略图。
    let immediate_path = if thumb_exists {
        thumb_path
    } else {
        store.origin_path(&file_name)
    };

    item.image_thumbnail_path = immediate_path.to_str().map(str::to_owned);

    // 缩略图不存在时后台异步预热，避免把大图缩放阻塞在列表/单条查询的返回路径上。
    if !thumb_exists {
        let store = store.clone();
        tauri::async_runtime::spawn_blocking(move || {
            if let Err(err) = store.ensure_thumbnail(&file_name) {
                log::warn!("ensure image thumbnail failed for {:?}: {err}", file_name);
            }
        });
    }

    Ok(())
}

/// 把 `clipboard_apps.icon_file` 解析为磁盘绝对路径写回 [`ClipboardItem::source_app_icon_path`]，
/// utf-8 转换失败时直接置 `None`（前端可降级到首字母占位）。
pub(crate) fn attach_source_app_icon_path(store: &AppIconStore, item: &mut ClipboardItem) {
    item.source_app_icon_path = item
        .source_app_icon_file
        .as_deref()
        .and_then(|name| store.icon_path(name).to_str().map(str::to_owned));
}

/// 仅当 `sub_kind = Color` 时，把 `summary`（或 `content` 兜底）规范化为可信 CSS 颜色串
/// 写入 `color_preview`，前端无需再自行校验即可塞 `style.background`。
pub(crate) fn attach_color_preview(item: &mut ClipboardItem) {
    if item.sub_kind != Some(ClipboardSubKind::Color) {
        return;
    }

    let source = item.summary.as_deref().unwrap_or(&item.content);
    item.color_preview = sanitize_css_color(source);
}

/// 按当前设置对敏感文本列表项返回脱敏摘要，避免前端列表暴露完整凭据。
pub(crate) fn redact_sensitive_list_item(item: &mut ClipboardItem, redact_sensitive: bool) {
    if !redact_sensitive || !item.is_sensitive || item.kind != ClipboardKind::Text {
        return;
    }

    if let Some(summary) = item.summary.as_mut() {
        *summary = mask_sensitive_text(summary);
    }
    item.color_preview = None;
}

/// 生成敏感文本遮罩：每行保留头尾少量字符，中间以星号替换。
pub(crate) fn mask_sensitive_text(text: &str) -> String {
    text.split('\n')
        .map(mask_sensitive_line)
        .collect::<Vec<_>>()
        .join("\n")
}

/// 遮罩单行敏感文本；短文本全量替换，长文本保留前后 4 个字符。
fn mask_sensitive_line(line: &str) -> String {
    const VISIBLE_EDGE_CHARS: usize = 4;
    const MAX_MASK_CHARS: usize = 8;

    let chars: Vec<char> = line.chars().collect();
    let len = chars.len();
    if len == 0 {
        return String::new();
    }
    if len <= VISIBLE_EDGE_CHARS * 2 {
        return "*".repeat(len);
    }

    let head = chars[..VISIBLE_EDGE_CHARS].iter().collect::<String>();
    let tail = chars[len - VISIBLE_EDGE_CHARS..].iter().collect::<String>();
    let mask_len = (len - VISIBLE_EDGE_CHARS * 2).min(MAX_MASK_CHARS);

    [head, "*".repeat(mask_len), tail].concat()
}

/// 把 `created_at`（UTC）按本地时区做三档展示格式化：
/// 今天 → `HH:mm`，今年内 → `MM-DD HH:mm`，跨年 → `YYYY-MM-DD HH:mm`。
/// `now` 由调用方在批处理外取一次，避免列表内逐条 syscall。
pub(crate) fn attach_display_created_at(item: &mut ClipboardItem, now: &DateTime<Local>) {
    let local = item.created_at.with_timezone(&Local);
    let today = now.date_naive() == local.date_naive();
    let same_year = now.year() == local.year();

    item.display_created_at = if today {
        local.format("%H:%M").to_string()
    } else if same_year {
        local.format("%m-%d %H:%M").to_string()
    } else {
        local.format("%Y-%m-%d %H:%M").to_string()
    };
}

/// 按 `kind` / `sub_kind` 计算右键菜单可用动作，按建议展示顺序返回。
/// 前端只负责把每个动作映射成 `MenuItemOptions`（文案 + 快捷键），
/// 不再自行判定「能否打开链接 / 是否文件可揭示」等业务规则。
pub(crate) fn compute_available_actions(item: &ClipboardItem) -> Vec<ClipboardAction> {
    let mut actions = Vec::with_capacity(10);

    actions.push(ClipboardAction::Paste);

    match item.kind {
        ClipboardKind::Text => actions.push(ClipboardAction::PasteAsPlainText),
        ClipboardKind::Files => actions.push(ClipboardAction::PasteAsPath),
        ClipboardKind::Image => {}
    }

    actions.push(ClipboardAction::Copy);
    if item.kind == ClipboardKind::Image {
        actions.push(ClipboardAction::SaveImage);
    }

    match item.sub_kind {
        Some(ClipboardSubKind::Url) => actions.push(ClipboardAction::OpenLink),
        Some(ClipboardSubKind::Email) => actions.push(ClipboardAction::SendEmail),
        _ => {}
    }

    let can_reveal =
        item.kind == ClipboardKind::Files || item.sub_kind == Some(ClipboardSubKind::Path);
    if can_reveal {
        #[cfg(target_os = "macos")]
        actions.push(ClipboardAction::RevealInFinder);
        #[cfg(target_os = "windows")]
        actions.push(ClipboardAction::RevealInExplorer);
    }

    actions.push(ClipboardAction::ToggleFavorite);
    actions.push(ClipboardAction::TogglePinned);
    actions.push(ClipboardAction::EditNote);
    actions.push(ClipboardAction::Delete);

    actions
}

/// 为 files 条目按设置组装前若干项 [`FileEntry`]：
/// 路径 / 文件名 / 目录标记 / 图片标记 / icon 绝对路径，前端直接渲染。
/// 非 files 条目或无路径时保持 `file_entries = None`。
///
/// icon 查询按条目批量合并为一次 `IN` 查询；磁盘存在性走 TTL 缓存，
/// 避免每次打开窗口都对同一批路径重复探测。
pub(crate) async fn attach_file_entries(
    pool: &SqlitePool,
    store: &FileIconStore,
    item: &mut ClipboardItem,
    limit: usize,
) -> Result<()> {
    if item.kind != ClipboardKind::Files {
        return Ok(());
    }

    let paths: Vec<&str> = item
        .content
        .split('\n')
        .filter(|p| !p.is_empty())
        .take(limit)
        .collect();
    if paths.is_empty() {
        return Ok(());
    }

    let types: Vec<&str> = item
        .file_types
        .as_deref()
        .unwrap_or("")
        .split(',')
        .collect();

    let platform = current_platform();
    let mut cache_keys: Vec<String> = Vec::with_capacity(paths.len());
    for (index, path) in paths.iter().enumerate() {
        let is_dir = types.get(index).copied() == Some("d");
        cache_keys.push(icon_cache_key(path, is_dir));
    }

    let icons = crate::db::file_icons::get_icons(pool, &cache_keys, platform).await?;

    let mut entries = Vec::with_capacity(paths.len());
    for (index, path) in paths.iter().enumerate() {
        let is_dir = types.get(index).copied() == Some("d");
        let exists = path_exists_cached(path);
        let icon_path =
            resolve_icon_with_prefetch(pool, store, path, &cache_keys[index], &icons, exists)
                .await?;
        let name = Path::new(path)
            .file_name()
            .and_then(|n| n.to_str())
            .map(str::to_owned)
            .unwrap_or_else(|| (*path).to_owned());
        let is_image = !is_dir && is_image_path(path);

        entries.push(FileEntry {
            path: (*path).to_owned(),
            name,
            is_dir,
            is_image,
            exists,
            icon_path,
        });
    }

    item.file_entries = Some(entries);
    item.files_preview_kind = Some(match item.file_entries.as_deref() {
        Some([only]) if only.is_image && only.exists => FilesPreviewKind::ImagePreview,
        _ => FilesPreviewKind::List,
    });
    Ok(())
}

fn current_platform() -> Platform {
    if cfg!(target_os = "macos") {
        Platform::Macos
    } else {
        Platform::Windows
    }
}

/// 与 [`resolve_file_icon_path`] 相同的 key 生成规则：目录走共享 key，其余按路径/类型推断。
fn icon_cache_key(path: &str, is_dir: bool) -> String {
    if is_dir {
        DIR_CACHE_KEY.to_string()
    } else {
        super::get_icon_cache_key(Path::new(path))
    }
}

/// 用批量查询的结果填充 icon 路径；未命中且文件存在时才抽取（并回写缓存）。
async fn resolve_icon_with_prefetch(
    pool: &SqlitePool,
    store: &FileIconStore,
    path: &str,
    cache_key: &str,
    icons: &HashMap<String, String>,
    exists: bool,
) -> Result<Option<String>> {
    if let Some(icon_file) = icons.get(cache_key) {
        let icon_path = store.icon_path(icon_file);
        if icon_path.exists() {
            return Ok(icon_path.to_str().map(str::to_owned));
        }
    }

    if !exists {
        return Ok(None);
    }

    let path_for_extract = Path::new(path).to_path_buf();
    let png_bytes =
        tauri::async_runtime::spawn_blocking(move || super::icon_png(&path_for_extract, None))
            .await
            .map_err(|err| AppError::Clipboard(format!("icon extract task join failed: {err}")))?;

    let Some(png) = png_bytes else {
        return Ok(None);
    };

    let icon_file = store.store(&png)?;
    crate::db::file_icons::upsert_icon(pool, cache_key, current_platform(), &icon_file).await?;

    Ok(store.icon_path(&icon_file).to_str().map(str::to_owned))
}

/// 解析文件 icon 路径：优先命中缓存，未命中时在路径存在的前提下抽取并落盘缓存。
/// 返回 `(icon_path, exists)`，其中 `icon_path` 可能为 `None`（抽取失败或已删除且无缓存）。
///
/// 供「单条取图标」「预览窗口」等非列表批量场景使用；列表路径见
/// [`attach_file_entries`]（那里先把 icon 查询合并成一次）。
pub(crate) async fn resolve_file_icon_path(
    pool: &SqlitePool,
    file_icon_store: &FileIconStore,
    path: &str,
    file_types: Option<&str>,
    index: usize,
) -> Result<(Option<String>, bool)> {
    let path_obj = Path::new(path);
    let exists = path_exists_cached(path);
    let platform = current_platform();

    let is_directory = file_types
        .and_then(|types| types.split(',').nth(index))
        .map(|t| t == "d");

    let cache_key = if is_directory == Some(true) {
        DIR_CACHE_KEY.to_string()
    } else {
        // 路径存在时实时判断（覆盖入库后类型变化的情况）；已删除时按扩展名推断。
        super::get_icon_cache_key(path_obj)
    };

    // DB 命中后还要确认 icon 文件仍在磁盘上：用户清缓存 / 手动删 file-icons 目录后，
    // 表里的 <hash>.png 映射就成了死引用，落到前端 convertFileSrc 会 404。缺了就当 miss 重抽。
    if let Some(icon_file) = crate::db::file_icons::get_icon(pool, &cache_key, platform).await? {
        let icon_path = file_icon_store.icon_path(&icon_file);
        if icon_path.exists() {
            return Ok((icon_path.to_str().map(str::to_owned), exists));
        }
    }

    if !exists {
        return Ok((None, false));
    }

    let path_for_extract = path_obj.to_path_buf();
    let png_bytes =
        tauri::async_runtime::spawn_blocking(move || super::icon_png(&path_for_extract, None))
            .await
            .map_err(|err| AppError::Clipboard(format!("icon extract task join failed: {err}")))?;

    let Some(png) = png_bytes else {
        return Ok((None, exists));
    };

    let icon_file = file_icon_store.store(&png)?;
    crate::db::file_icons::upsert_icon(pool, &cache_key, platform, &icon_file).await?;

    let icon_path = file_icon_store.icon_path(&icon_file);
    Ok((icon_path.to_str().map(str::to_owned), exists))
}

/// 与前端 `utils/is.ts` 的 `isImage` 同义：按扩展名（大小写不敏感）判断常见图片格式。
pub(crate) fn is_image_path(path: &str) -> bool {
    let Some(ext) = Path::new(path).extension().and_then(|e| e.to_str()) else {
        return false;
    };

    matches!(
        ext.to_ascii_lowercase().as_str(),
        "jpg"
            | "jpeg"
            | "png"
            | "webp"
            | "avif"
            | "gif"
            | "svg"
            | "bmp"
            | "ico"
            | "tif"
            | "tiff"
            | "heic"
            | "apng"
    )
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn query_key_ignores_whitespace_only_keyword() {
        let mut q = ClipboardItemQuery {
            keyword: Some("   ".to_owned()),
            ..Default::default()
        };
        let key = QueryKey::from(&q);

        q.keyword = None;
        assert_eq!(QueryKey::from(&q), key);
    }

    #[test]
    fn query_key_separates_offsets() {
        let mut q = ClipboardItemQuery::default();
        let first = QueryKey::from(&q);
        q.offset = 30;
        assert_ne!(QueryKey::from(&q), first);
    }

    /// 30 条与 60 条是同一份缓存的两种宽度，指纹必须相同。
    #[test]
    fn query_key_ignores_limit() {
        let mut q = ClipboardItemQuery {
            limit: 30,
            ..Default::default()
        };
        let first = QueryKey::from(&q);
        q.limit = 60;
        assert_eq!(QueryKey::from(&q), first);
    }

    fn page_with(len: usize, total: i64, has_more: bool) -> ClipboardItemPage {
        let mut page = ClipboardItemPage {
            list: Vec::new(),
            total,
            has_more,
            revision: 7,
        };
        page.list = (0..len)
            .map(|index| dummy_item(&format!("id-{index}")))
            .collect();
        page
    }

    /// 截断：只影响 list 长度与 has_more，total 与版本号原样透传。
    #[test]
    fn truncate_page_recomputes_has_more() {
        let page = page_with(60, 50277, true);

        let truncated = truncate_page(&page, 0, 30);
        assert_eq!(truncated.list.len(), 30);
        assert_eq!(truncated.total, 50277);
        assert!(truncated.has_more);
        assert_eq!(truncated.revision, 7);

        let tail = truncate_page(&page, 0, 60);
        assert_eq!(tail.list.len(), 60);
        assert!(tail.has_more);
    }

    #[test]
    fn truncate_page_marks_has_more_false_when_covering_end() {
        let page = page_with(5, 5, false);

        let truncated = truncate_page(&page, 0, 30);
        assert_eq!(truncated.list.len(), 5);
        assert!(!truncated.has_more);
    }

    fn dummy_item(id: &str) -> ClipboardItem {
        ClipboardItem {
            id: id.to_owned(),
            kind: ClipboardKind::Text,
            sub_kind: None,
            group_id: None,
            source_app_id: None,
            content: String::new(),
            content_hash: String::new(),
            search_text: None,
            summary: None,
            file_types: None,
            size: None,
            width: None,
            height: None,
            use_count: 0,
            is_favorite: false,
            is_pinned: false,
            is_sensitive: false,
            platform: Platform::Windows,
            note: None,
            created_at: chrono::Utc::now(),
            updated_at: chrono::Utc::now(),
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
}

//! 剪贴板相关命令：手动重新读取、解析图片路径。供前端按需触发。

use std::collections::HashMap;
use std::path::Path;
use std::sync::Arc;

use chrono::{DateTime, Datelike, Local, Utc};
use serde::{Deserialize, Serialize};
use sqlx::SqlitePool;
use tauri::{AppHandle, Emitter, Manager, State};

use crate::clipboard::{
    add_app_from_path, build_item_with_settings, delete_unreferenced_apps, detect_frontmost,
    materialize_source, persist_and_notify, refresh_running_apps, sanitize_css_color, AppIconStore,
    AppsRegistry, ClipboardReader, FileIconStore, ImageStore, WritebackGuard,
};
use crate::core::{AppError, Result};
use crate::db::items::{
    clear_items, find_item_by_id, find_item_for_list_by_id, increment_item_use_count,
};
use crate::db::models::{
    ClipboardAction, ClipboardApp, ClipboardGroup, ClipboardItem, ClipboardItemPage,
    ClipboardItemQuery, ClipboardKind, ClipboardSubKind, FileEntry, Platform,
};
use crate::db::DatabaseState;
use crate::settings::SettingsStore;
use crate::window::{self, MAIN_WINDOW_LABEL};

/// 与前端 `src/constants/events.ts` 的 `TAURI_EVENT.CLIPBOARD_UPDATED` 一一对应。
const CLIPBOARD_UPDATED_EVENT: &str = "clipboard://updated";

/// 与前端 `src/constants/events.ts` 的 `TAURI_EVENT.CLIPBOARD_GROUPS_UPDATED` 一一对应。
const CLIPBOARD_GROUPS_UPDATED_EVENT: &str = "clipboard-groups://updated";

const DEFAULT_CLIPBOARD_GROUP_ICON: &str = "i-lets-icons:folder";
const MAX_GROUP_NAME_CHARS: usize = 32;
const MAX_GROUP_ICON_BYTES: usize = 256 * 1024;

/// `read_clipboard` 的返回：入库后的记录 + 是否命中去重（前端据此决定提示/滚动行为）。
#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ReadClipboardResult {
    pub item: ClipboardItem,
    pub deduplicated: bool,
    /// 剪贴板为空 / 无可识别内容时为 `false`，此时 `item` 为 `None`。
    pub captured: bool,
}

/// 新建或更新自定义剪贴板分组的输入。
#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ClipboardGroupInput {
    pub name: String,
    pub icon: String,
    #[serde(default)]
    pub is_hidden: bool,
}

/// 批量保存自定义分组排序和主界面显隐状态的输入。
#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ClipboardGroupLayoutInput {
    pub order: Vec<String>,
    pub visible_ids: Vec<String>,
}

/// 手动读取当前剪贴板并入库（「重新读取」按钮）。
///
/// 复用监听管线：read_all → build_item（含图片落盘）→ persist_and_notify（去重入库 + emit）。
/// 与 OS 监听走同一条入库路径，语义一致；emit 同样触发前端列表刷新。
///
/// `ClipboardReader` 持有的 `ClipboardContext` 是 `!Send`，故读取与转换全部在
/// await 之前的同步块内完成并 drop，之后才进入异步入库——保证命令 future 满足 `Send`。
#[tauri::command]
pub async fn read_clipboard(
    app: AppHandle,
    db: State<'_, DatabaseState>,
    store: State<'_, ImageStore>,
    app_icon_store: State<'_, AppIconStore>,
    registry: State<'_, AppsRegistry>,
) -> Result<Option<ReadClipboardResult>> {
    // 先在同步段抓前台应用 + 读取剪贴板。
    // 注意：手动「重新读取」时前台应用就是 EcoPaste 自己，不像 OS 监听场景能拿到原应用——
    // 这里仍保留探测：若用户在外部应用复制后立刻命令式触发，多少能捕获到正确来源；
    // 命中我们自己也无害（apps 表只是多记一条「EcoPaste」）。
    let (item_opt, source) = {
        let source = detect_frontmost();
        let reader = ClipboardReader::new()?;
        let settings = app.state::<SettingsStore>().snapshot();
        let payload = reader.read_with_capture(&settings.clipboard.capture)?;
        let item = match payload {
            Some(payload) => build_item_with_settings(
                &store,
                &payload,
                &settings.clipboard.capture,
                &settings.clipboard.sensitive,
                settings.clipboard.content.copy_plain,
            )?,
            None => None,
        };
        (item, source)
        // reader 在此 drop，!Send 句柄不跨下方 await。
    };

    let Some(mut item) = item_opt else {
        return Ok(None);
    };

    let source_app = source.map(|src| materialize_source(&app_icon_store, Some(&registry), src));
    if let Some(src) = &source_app {
        item.source_app_id = Some(src.id.clone());
    }

    let pool = db.pool().await;
    let result = persist_and_notify(&app, &pool, &item, source_app.as_ref()).await?;
    Ok(Some(ReadClipboardResult {
        item,
        deduplicated: result.deduplicated,
        captured: true,
    }))
}

/// 把入库的图片文件名解析为磁盘绝对路径，供前端预览取图。
/// `thumbnail = true` 取缩略图，否则取原图。
///
/// 缩略图按需生成：不存在时在 `spawn_blocking` 里从原图解码 + 缩放 + 编码后落盘，再返回路径。
/// 解码/编码是 CPU 阻塞活，放到阻塞线程池避免占用 async worker；且「返回时文件已确保存在」，
/// 前端拿到路径即可安全 `convertFileSrc` 加载，不会撞到半成品文件。
///
/// `file_name` 来自前端（即记录的 `content`），是唯一的外部输入，需防路径穿越：
/// 仅允许「单层、纯 `<hash>.png` 形态」的文件名，含分隔符 / `..` / 子目录一律拒绝。
#[tauri::command]
pub async fn get_clipboard_image_path(
    store: State<'_, ImageStore>,
    file_name: String,
    thumbnail: bool,
) -> Result<String> {
    validate_image_file_name(&file_name)?;

    let path = if thumbnail {
        let store = store.inner().clone();
        tauri::async_runtime::spawn_blocking(move || store.ensure_thumbnail(&file_name))
            .await
            .map_err(|err| AppError::Clipboard(format!("thumbnail task join failed: {err}")))??
    } else {
        store.origin_path(&file_name)
    };

    path.to_str()
        .map(str::to_owned)
        .ok_or_else(|| AppError::Clipboard("image path is not valid utf-8".to_owned()))
}

/// 把 `clipboard_apps.icon_file`（形如 `<hash>.png`）解析为磁盘绝对路径，
/// 供前端用 `convertFileSrc` 渲染。文件名复用图片同款防穿越校验。
#[tauri::command]
pub async fn get_clipboard_app_icon_path(
    store: State<'_, AppIconStore>,
    file_name: String,
) -> Result<String> {
    validate_image_file_name(&file_name)?;
    store
        .icon_path(&file_name)
        .to_str()
        .map(str::to_owned)
        .ok_or_else(|| AppError::Clipboard("app icon path is not valid utf-8".to_owned()))
}

/// `get_file_icon_path` 的返回：icon 绝对路径 + 文件当前是否存在于磁盘。
/// 前端据 `exists = false` 给已删除的文件显示弱化样式 / 提示。
#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct FileIconResult {
    /// icon 文件磁盘绝对路径；抽取失败 / 路径已删除且未缓存时为 `None`。
    pub icon_path: Option<String>,
    /// 当前路径是否仍存在于磁盘。
    pub exists: bool,
}

/// 偏好页来源应用列表项：在 DB 模型基础上补齐前端可直接渲染的 icon 绝对路径。
#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ClipboardAppView {
    pub id: String,
    pub name: String,
    pub icon_file: Option<String>,
    pub icon_path: Option<String>,
    pub platform: Platform,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

const PREVIEW_FILE_ENTRY_LIMIT: usize = 64;

/// 预览窗口专用 payload：只暴露 Content Viewer 渲染所需的归一化字段。
#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ClipboardPreviewPayload {
    pub id: String,
    pub kind: ClipboardKind,
    pub sub_kind: Option<ClipboardSubKind>,
    pub updated_at: DateTime<Utc>,
    /// 预览窗口展示用纯文本。HTML / RTF 条目返回 `search_text`，不返回富文本源。
    pub text: Option<String>,
    pub image_path: Option<String>,
    pub image_width: Option<i64>,
    pub image_height: Option<i64>,
    pub size: Option<i64>,
    pub is_sensitive: bool,
    pub image_exists: bool,
    pub files: Vec<ClipboardPreviewFileEntry>,
    pub total_files: usize,
}

/// 预览窗口里的单个文件条目，比列表卡片保留更多文件并带上 size。
#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ClipboardPreviewFileEntry {
    pub path: String,
    pub name: String,
    pub is_dir: bool,
    pub is_image: bool,
    pub exists: bool,
    pub size: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub icon_path: Option<String>,
}

/// 获取文件类型 icon 的磁盘绝对路径，供前端 FilesCard 渲染。
///
/// 懒加载：缓存 miss 时，若路径存在则抽取 icon 并缓存；路径已删除则 `iconPath = None`（前端显示弱化态）。
/// `file_types` 来自 `clipboard_items.file_types`（如 "d,f,f"），`index` 为路径在列表中的索引（0-based）。
#[tauri::command]
pub async fn get_file_icon_path(
    db: State<'_, DatabaseState>,
    file_icon_store: State<'_, FileIconStore>,
    path: String,
    file_types: Option<String>,
    index: usize,
) -> Result<FileIconResult> {
    let pool = db.pool().await;
    let (icon_path, exists) =
        resolve_file_icon_path(&pool, &file_icon_store, &path, file_types.as_deref(), index)
            .await?;

    Ok(FileIconResult { icon_path, exists })
}

/// 读取单条剪贴板记录的完整预览 payload。
///
/// 列表查询会裁剪 text content；预览必须走完整记录，再按 Content Viewer 的三类视图归一化。
#[tauri::command]
pub async fn get_clipboard_preview_payload(
    app: AppHandle,
    db: State<'_, DatabaseState>,
    image_store: State<'_, ImageStore>,
    file_icon_store: State<'_, FileIconStore>,
    item_id: String,
) -> Result<Option<ClipboardPreviewPayload>> {
    let pool = db.pool().await;
    let Some(item) = find_item_by_id(&pool, &item_id).await? else {
        return Ok(None);
    };

    let redact_sensitive = app
        .state::<SettingsStore>()
        .snapshot()
        .clipboard
        .sensitive
        .redact_secrets;
    let payload = build_clipboard_preview_payload(
        &pool,
        &image_store,
        &file_icon_store,
        item,
        redact_sensitive,
    )
    .await?;
    Ok(Some(payload))
}

/// 播放一次复制成功提示音，用于偏好设置页试听。
#[tauri::command]
pub async fn play_copy_sound() {
    crate::clipboard::play_copy_sound();
}

/// 把指定历史记录写回系统剪贴板（不触发模拟粘贴）。
/// `plain = true` 强制纯文本，剥离 HTML/RTF。
///
/// `ClipboardContext` 是 `!Send`，写回逻辑在 `clipboard::write_to_clipboard` 内同步完成，
/// 之后无 await，命令 future 仍满足 `Send`。
#[tauri::command]
pub async fn write_to_clipboard(
    app: AppHandle,
    db: State<'_, DatabaseState>,
    store: State<'_, ImageStore>,
    guard: State<'_, Arc<WritebackGuard>>,
    id: String,
    plain: bool,
) -> Result<()> {
    let pool = db.pool().await;
    let item = find_item_by_id(&pool, &id)
        .await?
        .ok_or_else(|| AppError::Clipboard(format!("clipboard item not found: {id}")))?;

    let settings = app.state::<SettingsStore>().snapshot();
    let write_plain =
        should_write_plain_for_copy(plain, item.kind, settings.clipboard.content.copy_plain);
    let hide_after_copy = settings.clipboard.content.copy_then_hide_window;

    crate::clipboard::write_to_clipboard(&store, guard.inner().as_ref(), &item, write_plain)?;
    mark_item_reused_if_enabled(&app, &pool, &id, item.kind).await?;

    if hide_after_copy {
        hide_main_window_after_copy(&app);
    }

    Ok(())
}

/// 「点击列表项 → 自动粘贴」的组合命令：写回剪贴板 + 隐藏主窗口 + 触发系统级粘贴。
///
/// 窗口已是非激活面板（macOS NSPanel `nonactivating_panel` / Windows `focusable=false`），
/// show 时不会把前台 App 推走，前台焦点始终在用户原窗口。
/// macOS 上 panel 会成为 key window，CGEvent ⌘V 若不先 hide 会被 panel 自己吞掉，
/// hide 后插入 50ms 让 panel 真正 order_out（hide_window 是 run_on_main_thread 异步派发，
/// 右键菜单触发时主线程仍在处理菜单关闭，不等会出现 ⌘V 早于 hide 完成的竞态）。
#[tauri::command]
pub async fn paste_clipboard_item(
    app: AppHandle,
    db: State<'_, DatabaseState>,
    store: State<'_, ImageStore>,
    guard: State<'_, Arc<WritebackGuard>>,
    id: String,
    plain: bool,
) -> Result<()> {
    let pool = db.pool().await;
    let item = find_item_by_id(&pool, &id)
        .await?
        .ok_or_else(|| AppError::Clipboard(format!("clipboard item not found: {id}")))?;

    let settings = app.state::<SettingsStore>().snapshot();
    let write_plain = should_write_plain_for_paste(
        plain,
        item.kind,
        settings.clipboard.content.paste_plain,
        settings.clipboard.content.paste_files_as_path,
    );

    crate::clipboard::write_to_clipboard(&store, guard.inner().as_ref(), &item, write_plain)?;
    mark_item_reused_if_enabled(&app, &pool, &id, item.kind).await?;

    if window::is_main_window_pinned() {
        // 固定时窗口保持可见：macOS 上 panel 仍是 key window 会吞掉 ⌘V，需先 resign key
        // 让键焦点回到前台 App 的窗口；Windows 主窗口 focusable=false，无需处理。
        #[cfg(target_os = "macos")]
        if let Err(err) = window::macos::resign_main_panel_key(&app) {
            log::warn!("resign main panel key before paste failed: {err:?}");
        }
    } else if let Err(err) = window::hide_window(&app, MAIN_WINDOW_LABEL) {
        log::warn!("hide main window before paste failed: {err:?}");
    }

    // hide / resign 都是 run_on_main_thread 异步派发；不等一拍，simulate_paste 的 ⌘V
    // 会赶在 panel 真正 order_out / 让出 key 前命中 panel 自己（webview 吞掉）。
    tokio::time::sleep(std::time::Duration::from_millis(50)).await;

    crate::keystroke::simulate_paste()?;

    // 固定窗口下粘贴完把 key 拿回来，让用户继续用键盘 / 列表操作；
    // 再等一拍让 ⌘V 事件被目标 App 消费完，避免 make_key 抢回焦点把按键吞回 panel。
    if window::is_main_window_pinned() {
        #[cfg(target_os = "macos")]
        {
            tokio::time::sleep(std::time::Duration::from_millis(50)).await;
            if let Err(err) = window::macos::make_main_panel_key(&app) {
                log::warn!("restore main panel key after paste failed: {err:?}");
            }
        }
    }

    Ok(())
}

/// 计算复制写回是否强制走纯文本；默认复制纯文本只作用于文本记录。
fn should_write_plain_for_copy(force_plain: bool, kind: ClipboardKind, copy_plain: bool) -> bool {
    force_plain || kind == ClipboardKind::Text && copy_plain
}

/// 计算粘贴写回是否强制走纯文本；文本去格式与文件路径粘贴分别由各自设置控制。
fn should_write_plain_for_paste(
    force_plain: bool,
    kind: ClipboardKind,
    paste_plain: bool,
    paste_files_as_path: bool,
) -> bool {
    force_plain
        || kind == ClipboardKind::Text && paste_plain
        || kind == ClipboardKind::Files && paste_files_as_path
}

/// 按设置决定是否把复制 / 粘贴历史记录计为一次复用。
async fn mark_item_reused_if_enabled(
    app: &AppHandle,
    pool: &SqlitePool,
    id: &str,
    kind: ClipboardKind,
) -> Result<()> {
    let settings = app.state::<SettingsStore>().snapshot();
    if !settings.clipboard.content.update_on_reuse {
        return Ok(());
    }

    increment_item_use_count(pool, id).await?;
    if let Err(err) = app.emit(
        CLIPBOARD_UPDATED_EVENT,
        serde_json::json!({
            "id": id,
            "kind": kind,
            "deduplicated": true,
        }),
    ) {
        log::warn!("emit {CLIPBOARD_UPDATED_EVENT} after item reuse failed: {err}");
    }

    Ok(())
}

/// 从历史复制后按设置隐藏主窗口；固定状态下尊重用户显式 pin。
fn hide_main_window_after_copy(app: &AppHandle) {
    if window::is_main_window_pinned() {
        return;
    }

    if let Err(err) = window::hide_window(app, MAIN_WINDOW_LABEL) {
        log::warn!("hide main window after copy failed: {err:?}");
    }
}

/// 列表查询命令（薄封装）：参数缺省时走 Rust 端默认（limit=20, offset=0, updatedAtDesc）；
/// `keyword` 非空时由 `query_items` 内部自动委派 FTS5。
/// 返回 [`ClipboardItemPage`]：顶页项 + 同过滤下的总数 + `hasMore`，
/// 供前端一次 IPC 同时拿到「列表 / Footer 总数 / 是否还有下一页」。
/// 顺带把 `source_app_icon_file` 解析为绝对路径写到 `source_app_icon_path`，
/// 前端无需再拉 `get_clipboard_app_icon_path`。
#[tauri::command]
pub async fn list_clipboard_items(
    app: AppHandle,
    db: State<'_, DatabaseState>,
    image_store: State<'_, ImageStore>,
    app_icon_store: State<'_, AppIconStore>,
    file_icon_store: State<'_, FileIconStore>,
    query: Option<ClipboardItemQuery>,
) -> Result<ClipboardItemPage> {
    let pool = db.pool().await;
    let q = query.unwrap_or_default();
    let (mut items, total) = crate::db::items::query_items_page(&pool, &q).await?;
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
    let has_more = q.offset + (items.len() as i64) < total;
    Ok(ClipboardItemPage {
        list: items,
        total,
        has_more,
    })
}

/// 单条查询命令：监听到 `clipboard://updated` 后前端按 id 拉单条，
/// 避免事件驱动刷新时整页 refetch。不存在返回 `None`，前端按需降级。
///
/// 这里故意走列表视图（text 类型 content 置空）保持与列表查询同款裁剪；
/// 需要完整 content 的写回/预览走各自专用命令。
#[tauri::command]
pub async fn get_clipboard_item(
    app: AppHandle,
    db: State<'_, DatabaseState>,
    image_store: State<'_, ImageStore>,
    app_icon_store: State<'_, AppIconStore>,
    file_icon_store: State<'_, FileIconStore>,
    id: String,
) -> Result<Option<ClipboardItem>> {
    let pool = db.pool().await;
    let mut item = find_item_for_list_by_id(&pool, &id).await?;
    if let Some(item) = item.as_mut() {
        let settings = app.state::<SettingsStore>().snapshot();
        let file_entry_limit = settings.clipboard.display.file_entry_limit();
        let redact_sensitive = settings.clipboard.sensitive.redact_secrets;
        attach_image_thumbnail_path(&image_store, item).await?;
        attach_source_app_icon_path(&app_icon_store, item);
        attach_file_entries(&pool, &file_icon_store, item, file_entry_limit).await?;
        attach_color_preview(item);
        attach_display_created_at(item, &Local::now());
        redact_sensitive_list_item(item, redact_sensitive);
        item.available_actions = compute_available_actions(item);
    }
    Ok(item)
}

/// 为 image 条目补齐缩略图绝对路径，前端可直接渲染。
/// 历史脏数据（非 `<hash>.png`）或缩略图生成失败时降级为 `None`，不影响列表返回。
async fn attach_image_thumbnail_path(store: &ImageStore, item: &mut ClipboardItem) -> Result<()> {
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
fn attach_source_app_icon_path(store: &AppIconStore, item: &mut ClipboardItem) {
    item.source_app_icon_path = item
        .source_app_icon_file
        .as_deref()
        .and_then(|name| store.icon_path(name).to_str().map(str::to_owned));
}

/// 把来源应用 DB 模型转换为偏好页列表可直接渲染的视图模型。
fn build_clipboard_app_view(store: &AppIconStore, app: ClipboardApp) -> ClipboardAppView {
    let icon_path = app
        .icon_file
        .as_deref()
        .and_then(|name| store.icon_path(name).to_str().map(str::to_owned));

    ClipboardAppView {
        id: app.id,
        name: app.name,
        icon_file: app.icon_file,
        icon_path,
        platform: app.platform,
        created_at: app.created_at,
        updated_at: app.updated_at,
    }
}

/// 仅当 `sub_kind = Color` 时，把 `summary`（或 `content` 兜底）规范化为可信 CSS 颜色串
/// 写入 `color_preview`，前端无需再自行校验即可塞 `style.background`。
fn attach_color_preview(item: &mut ClipboardItem) {
    if item.sub_kind != Some(ClipboardSubKind::Color) {
        return;
    }

    let source = item.summary.as_deref().unwrap_or(&item.content);
    item.color_preview = sanitize_css_color(source);
}

/// 按当前设置对敏感文本列表项返回脱敏摘要，避免前端列表暴露完整凭据。
fn redact_sensitive_list_item(item: &mut ClipboardItem, redact_sensitive: bool) {
    if !redact_sensitive || !item.is_sensitive || item.kind != ClipboardKind::Text {
        return;
    }

    if let Some(summary) = item.summary.as_mut() {
        *summary = mask_sensitive_text(summary);
    }
    item.color_preview = None;
}

/// 生成敏感文本遮罩：每行保留头尾少量字符，中间以星号替换。
fn mask_sensitive_text(text: &str) -> String {
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

/// 返回预览 payload 的文本子类型；脱敏展示时敏感内容强制按纯文本展示。
fn preview_sub_kind(item: &ClipboardItem, redact_sensitive: bool) -> Option<ClipboardSubKind> {
    if redact_sensitive && item.is_sensitive && item.kind == ClipboardKind::Text {
        return None;
    }

    item.sub_kind
}

/// 返回预览窗口展示用文本；HTML / RTF 使用 OS 提供的纯文本表示，避免前端渲染富文本源。
fn preview_text(item: &ClipboardItem, redact_sensitive: bool) -> String {
    let source = match item.sub_kind {
        Some(ClipboardSubKind::Html | ClipboardSubKind::Rtf) => {
            item.search_text.as_deref().unwrap_or(&item.content)
        }
        _ => &item.content,
    };

    if redact_sensitive && item.is_sensitive {
        return mask_sensitive_text(source);
    }

    source.to_owned()
}

/// 把 `created_at`（UTC）按本地时区做三档展示格式化：
/// 今天 → `HH:mm:ss`，今年内 → `MM-DD HH:mm`，跨年 → `YYYY-MM-DD HH:mm`。
/// `now` 由调用方在批处理外取一次，避免列表内逐条 syscall。
fn attach_display_created_at(item: &mut ClipboardItem, now: &chrono::DateTime<Local>) {
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
fn compute_available_actions(item: &ClipboardItem) -> Vec<ClipboardAction> {
    let mut actions = Vec::with_capacity(10);

    actions.push(ClipboardAction::Paste);

    match item.kind {
        ClipboardKind::Text => actions.push(ClipboardAction::PasteAsPlainText),
        ClipboardKind::Files => actions.push(ClipboardAction::PasteAsPath),
        ClipboardKind::Image => {}
    }

    actions.push(ClipboardAction::Copy);

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
async fn attach_file_entries(
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

    let mut entries = Vec::with_capacity(paths.len());
    for (index, path) in paths.iter().enumerate() {
        let (icon_path, exists) =
            resolve_file_icon_path(pool, store, path, item.file_types.as_deref(), index).await?;
        let is_dir = types.get(index).copied() == Some("d");
        let name = std::path::Path::new(path)
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
        Some([only]) if only.is_image && only.exists => {
            crate::db::models::FilesPreviewKind::ImagePreview
        }
        _ => crate::db::models::FilesPreviewKind::List,
    });
    Ok(())
}

/// 将完整 `ClipboardItem` 转为预览窗口的轻量数据模型。
async fn build_clipboard_preview_payload(
    pool: &SqlitePool,
    image_store: &ImageStore,
    file_icon_store: &FileIconStore,
    item: ClipboardItem,
    redact_sensitive: bool,
) -> Result<ClipboardPreviewPayload> {
    let mut text = None;
    let mut image_path = None;
    let mut image_exists = false;
    let mut files = Vec::new();
    let mut total_files = 0;

    match item.kind {
        ClipboardKind::Text => {
            text = Some(preview_text(&item, redact_sensitive));
        }
        ClipboardKind::Image => {
            validate_image_file_name(&item.content)?;
            let path = image_store.origin_path(&item.content);
            image_exists = path.exists();
            image_path = Some(path_to_string(&path, "image path")?);
        }
        ClipboardKind::Files => {
            total_files = count_file_paths(&item.content);
            files = build_preview_file_entries(pool, file_icon_store, &item).await?;
        }
    }
    let preview_sub_kind = preview_sub_kind(&item, redact_sensitive);

    Ok(ClipboardPreviewPayload {
        id: item.id,
        kind: item.kind,
        sub_kind: preview_sub_kind,
        updated_at: item.updated_at,
        text,
        image_path,
        image_width: item.width,
        image_height: item.height,
        size: item.size,
        is_sensitive: item.is_sensitive,
        image_exists,
        files,
        total_files,
    })
}

/// 解析 files 类型记录中的路径列表，最多返回前 64 项以控制 IPC 与 icon 抽取成本。
async fn build_preview_file_entries(
    pool: &SqlitePool,
    store: &FileIconStore,
    item: &ClipboardItem,
) -> Result<Vec<ClipboardPreviewFileEntry>> {
    let paths: Vec<&str> = item
        .content
        .split('\n')
        .filter(|path| !path.is_empty())
        .take(PREVIEW_FILE_ENTRY_LIMIT)
        .collect();

    let mut entries = Vec::with_capacity(paths.len());
    for (index, path) in paths.iter().enumerate() {
        let (icon_path, exists) =
            resolve_file_icon_path(pool, store, path, item.file_types.as_deref(), index).await?;
        let path_obj = Path::new(path);
        let is_dir = resolve_preview_file_is_dir(path_obj, item.file_types.as_deref(), index);
        let is_image = !is_dir && is_image_path(path);
        let size = resolve_preview_file_size(path_obj, is_dir);
        let name = path_obj
            .file_name()
            .and_then(|n| n.to_str())
            .map(str::to_owned)
            .unwrap_or_else(|| (*path).to_owned());

        entries.push(ClipboardPreviewFileEntry {
            path: (*path).to_owned(),
            name,
            is_dir,
            is_image,
            exists,
            size,
            icon_path,
        });
    }

    Ok(entries)
}

/// 统计 files payload 中的有效路径数量。
fn count_file_paths(content: &str) -> usize {
    content.split('\n').filter(|path| !path.is_empty()).count()
}

/// 解析文件是否为目录：存在时信任实时 metadata，缺失时回退到入库时保存的 file_types。
fn resolve_preview_file_is_dir(path: &Path, file_types: Option<&str>, index: usize) -> bool {
    if let Ok(metadata) = path.metadata() {
        return metadata.is_dir();
    }

    file_types
        .and_then(|types| types.split(',').nth(index))
        .map(|file_type| file_type == "d")
        .unwrap_or(false)
}

/// 解析普通文件大小；目录或缺失路径不显示 size。
fn resolve_preview_file_size(path: &Path, is_dir: bool) -> Option<i64> {
    if is_dir {
        return None;
    }

    path.metadata().ok().map(|metadata| metadata.len() as i64)
}

/// 将本地路径转为 UTF-8 字符串，失败时返回面向用户的 clipboard 错误。
fn path_to_string(path: &Path, label: &str) -> Result<String> {
    path.to_str()
        .map(str::to_owned)
        .ok_or_else(|| AppError::Clipboard(format!("{label} is not valid utf-8")))
}

/// 与前端 `utils/is.ts` 的 `isImage` 同义：按扩展名（大小写不敏感）判断常见图片格式。
fn is_image_path(path: &str) -> bool {
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

/// 解析文件 icon 路径：优先命中缓存，未命中时在路径存在的前提下抽取并落盘缓存。
/// 返回 `(icon_path, exists)`，其中 `icon_path` 可能为 `None`（抽取失败或已删除且无缓存）。
async fn resolve_file_icon_path(
    pool: &SqlitePool,
    file_icon_store: &FileIconStore,
    path: &str,
    file_types: Option<&str>,
    index: usize,
) -> Result<(Option<String>, bool)> {
    let path_obj = Path::new(path);
    let exists = path_obj.exists();
    let platform = if cfg!(target_os = "macos") {
        Platform::Macos
    } else {
        Platform::Windows
    };

    let is_directory = file_types
        .and_then(|types| types.split(',').nth(index))
        .map(|t| t == "d");

    let cache_key = if is_directory == Some(true) {
        crate::clipboard::DIR_CACHE_KEY.to_string()
    } else {
        // 路径存在时实时判断（覆盖入库后类型变化的情况）；已删除时按扩展名推断。
        crate::clipboard::get_icon_cache_key(path_obj)
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
    let png_bytes = tauri::async_runtime::spawn_blocking(move || {
        crate::clipboard::icon_png(&path_for_extract, None)
    })
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

/// 按 id 列表批量取来源应用——前端渲染卡片时一次性补齐图标/名称。
#[tauri::command]
pub async fn list_clipboard_apps(
    db: State<'_, DatabaseState>,
    ids: Vec<String>,
) -> Result<Vec<crate::db::models::ClipboardApp>> {
    let pool = db.pool().await;
    crate::db::apps::list_apps_by_ids(&pool, &ids).await
}

/// 列出可过滤应用：DB 已知应用加上本次运行中枚举到的临时应用。
#[tauri::command]
pub async fn list_all_apps(
    app_icon_store: State<'_, AppIconStore>,
    db: State<'_, DatabaseState>,
    registry: State<'_, AppsRegistry>,
) -> Result<Vec<ClipboardAppView>> {
    let pool = db.pool().await;
    let running_apps = refresh_running_apps(registry.inner().clone()).await?;
    let known_apps = crate::db::apps::list_all_apps(&pool).await?;
    let mut apps = merge_clipboard_apps(known_apps, running_apps);

    sort_clipboard_apps(&mut apps);
    Ok(apps
        .into_iter()
        .map(|app| build_clipboard_app_view(&app_icon_store, app))
        .collect())
}

/// 从用户手动选择的应用路径写入来源应用注册表。
#[tauri::command]
pub async fn add_clipboard_app_from_path(
    app_icon_store: State<'_, AppIconStore>,
    registry: State<'_, AppsRegistry>,
    path: String,
) -> Result<ClipboardAppView> {
    let app = add_app_from_path(registry.inner().clone(), path).await?;

    Ok(build_clipboard_app_view(&app_icon_store, app))
}

/// 删除未被历史记录引用的来源应用，通常用于从忽略列表移除手动/默认物化应用。
#[tauri::command]
pub async fn delete_unreferenced_clipboard_apps(
    registry: State<'_, AppsRegistry>,
    ids: Vec<String>,
) -> Result<Vec<String>> {
    delete_unreferenced_apps(registry.inner().clone(), ids).await
}

/// 合并 DB 已知应用与运行中临时应用，同 id 时保留 DB 行。
fn merge_clipboard_apps(
    known_apps: Vec<ClipboardApp>,
    running_apps: Vec<ClipboardApp>,
) -> Vec<ClipboardApp> {
    let mut merged = HashMap::with_capacity(known_apps.len() + running_apps.len());

    for app in running_apps {
        merged.insert(app.id.clone(), app);
    }
    for mut app in known_apps {
        if app.icon_file.is_none() {
            app.icon_file = merged
                .get(&app.id)
                .and_then(|running| running.icon_file.clone());
        }
        merged.insert(app.id.clone(), app);
    }

    merged.into_values().collect()
}

/// 按名称和 id 稳定排序来源应用列表。
fn sort_clipboard_apps(apps: &mut [ClipboardApp]) {
    apps.sort_by(|left, right| {
        left.name
            .to_lowercase()
            .cmp(&right.name.to_lowercase())
            .then_with(|| left.id.cmp(&right.id))
    });
}

/// 归一化分组名称，保证持久化前去除首尾空白并限制长度。
fn normalize_group_name(name: &str) -> Result<String> {
    let normalized = name.trim();
    if normalized.is_empty() {
        return Err(AppError::Clipboard("分组名称不能为空".to_owned()));
    }

    if normalized.chars().count() > MAX_GROUP_NAME_CHARS {
        return Err(AppError::Clipboard("分组名称不能超过 32 个字符".to_owned()));
    }

    Ok(normalized.to_owned())
}

/// 归一化分组图标；空值回退到默认预设图标，自定义 SVG 做基础校验。
fn normalize_group_icon(icon: &str) -> Result<String> {
    let normalized = icon.trim();
    if normalized.is_empty() {
        return Ok(DEFAULT_CLIPBOARD_GROUP_ICON.to_owned());
    }

    if normalized.len() > MAX_GROUP_ICON_BYTES {
        return Err(AppError::Clipboard("SVG 图标不能超过 256 KB".to_owned()));
    }

    if normalized.starts_with("<svg") {
        normalize_group_svg(normalized)?;
    }

    Ok(normalized.to_owned())
}

/// 校验自定义 SVG 图标的基础形态。
fn normalize_group_svg(icon: &str) -> Result<()> {
    let normalized = icon.trim_start();
    if !normalized.starts_with("<svg") {
        return Err(AppError::Clipboard("请选择有效的 SVG 图标".to_owned()));
    }

    let lower = normalized.to_ascii_lowercase();
    if lower.contains("<script") || lower.contains("<foreignobject") {
        return Err(AppError::Clipboard(
            "SVG 图标不能包含脚本或 foreignObject".to_owned(),
        ));
    }

    Ok(())
}

/// 广播自定义分组变更，打开的前端窗口据此刷新分组列表。
fn emit_clipboard_groups_updated(app: &AppHandle) {
    if let Err(err) = app.emit(CLIPBOARD_GROUPS_UPDATED_EVENT, ()) {
        log::warn!("emit clipboard groups updated failed: {err}");
    }
}

/// 列出全部分组（薄封装），供前端构建分组 tab 栏。
/// 按 `sort_order` 升序，同序按 `created_at` 兜底。
#[tauri::command]
pub async fn list_clipboard_groups(db: State<'_, DatabaseState>) -> Result<Vec<ClipboardGroup>> {
    let pool = db.pool().await;
    crate::db::groups::list_groups(&pool).await
}

/// 新建自定义剪贴板分组。
#[tauri::command]
pub async fn create_clipboard_group(
    app: AppHandle,
    db: State<'_, DatabaseState>,
    input: ClipboardGroupInput,
) -> Result<ClipboardGroup> {
    let pool = db.pool().await;
    let name = normalize_group_name(&input.name)?;
    let icon = normalize_group_icon(&input.icon)?;
    let now = Utc::now();
    let group = ClipboardGroup {
        id: uuid::Uuid::new_v4().to_string(),
        name,
        icon,
        is_hidden: input.is_hidden,
        sort_order: crate::db::groups::next_group_sort_order(&pool).await?,
        created_at: now,
        updated_at: now,
    };

    crate::db::groups::insert_group(&pool, &group).await?;
    emit_clipboard_groups_updated(&app);

    Ok(group)
}

/// 更新自定义剪贴板分组名称、图标和显示状态。
#[tauri::command]
pub async fn update_clipboard_group(
    app: AppHandle,
    db: State<'_, DatabaseState>,
    id: String,
    input: ClipboardGroupInput,
) -> Result<()> {
    let pool = db.pool().await;
    let name = normalize_group_name(&input.name)?;
    let icon = normalize_group_icon(&input.icon)?;

    crate::db::groups::update_group(&pool, &id, &name, &icon, input.is_hidden).await?;
    emit_clipboard_groups_updated(&app);

    Ok(())
}

/// 批量更新自定义剪贴板分组排序和显示状态。
#[tauri::command]
pub async fn update_clipboard_groups_layout(
    app: AppHandle,
    db: State<'_, DatabaseState>,
    input: ClipboardGroupLayoutInput,
) -> Result<()> {
    let pool = db.pool().await;

    crate::db::groups::update_group_layout(&pool, &input.order, &input.visible_ids).await?;
    emit_clipboard_groups_updated(&app);

    Ok(())
}

/// 删除自定义剪贴板分组。
#[tauri::command]
pub async fn delete_clipboard_group(
    app: AppHandle,
    db: State<'_, DatabaseState>,
    id: String,
) -> Result<()> {
    let pool = db.pool().await;

    crate::db::groups::delete_group(&pool, &id).await?;
    emit_clipboard_groups_updated(&app);

    Ok(())
}

/// 读取用户通过 Tauri dialog 选择的 SVG 文件内容，供前端作为自定义分组图标保存。
#[tauri::command]
pub async fn import_clipboard_group_svg(path: String) -> Result<String> {
    let path = Path::new(&path);
    let extension = path
        .extension()
        .and_then(|value| value.to_str())
        .unwrap_or_default()
        .to_ascii_lowercase();
    if extension != "svg" {
        return Err(AppError::Clipboard("请选择 SVG 文件".to_owned()));
    }

    let bytes = std::fs::read(path)
        .map_err(|err| AppError::Clipboard(format!("无法读取 SVG 文件：{err}")))?;
    if bytes.len() > MAX_GROUP_ICON_BYTES {
        return Err(AppError::Clipboard("SVG 文件不能超过 256 KB".to_owned()));
    }

    let icon = String::from_utf8(bytes)
        .map_err(|_| AppError::Clipboard("SVG 文件必须是 UTF-8 文本".to_owned()))?;
    normalize_group_svg(&icon)?;

    Ok(icon)
}

/// 翻转收藏状态（薄封装）。前端在调用前/后做乐观更新；这里不返回新状态，
/// 失败时前端按需回滚 / 重拉单条。
#[tauri::command]
pub async fn toggle_clipboard_item_favorite(
    db: State<'_, DatabaseState>,
    id: String,
) -> Result<bool> {
    let pool = db.pool().await;
    crate::db::items::toggle_item_favorite(&pool, &id).await
}

/// 翻转置顶状态（薄封装）。不更新 `updated_at`，避免污染最近使用排序。
#[tauri::command]
pub async fn toggle_clipboard_item_pinned(
    db: State<'_, DatabaseState>,
    id: String,
) -> Result<bool> {
    let pool = db.pool().await;
    crate::db::items::toggle_item_pinned(&pool, &id).await
}

/// 删除单条记录（薄封装）。若删的是图片记录，连带删除其落盘文件（原图 + 缩略图）。
/// 行已删成功，删文件失败仅记日志、不回滚——残留文件最坏占用磁盘，不影响功能。
#[tauri::command]
pub async fn delete_clipboard_item(
    db: State<'_, DatabaseState>,
    store: State<'_, ImageStore>,
    id: String,
) -> Result<()> {
    let pool = db.pool().await;
    if let Some(file_name) = crate::db::items::delete_item(&pool, &id).await? {
        if let Err(err) = store.remove(&file_name) {
            log::warn!("remove deleted image {file_name} failed: {err}");
        }
    }
    Ok(())
}

/// 清空全部历史记录，并删除对应图片资源。完成后广播列表刷新事件。
#[tauri::command]
pub async fn clear_clipboard_items(
    app: AppHandle,
    db: State<'_, DatabaseState>,
    store: State<'_, ImageStore>,
    delete_favorites: bool,
    delete_pinned: bool,
) -> Result<u64> {
    let pool = db.pool().await;
    let outcome = clear_items(&pool, delete_favorites, delete_pinned).await?;

    for file_name in &outcome.image_files {
        if let Err(err) = store.remove(file_name) {
            log::warn!("remove cleared image {file_name} failed: {err}");
        }
    }

    if let Err(err) = app.emit(
        CLIPBOARD_UPDATED_EVENT,
        serde_json::json!({
            "cleanup": outcome.removed,
        }),
    ) {
        log::warn!("emit {CLIPBOARD_UPDATED_EVENT} after clear failed: {err}");
    }

    Ok(outcome.removed)
}

/// 更新备注（薄封装）。`note = None` 或空串清空备注；空串归一化为 None，
/// 保证「无备注」在库里只有一种表示（NULL），避免后续筛选/展示判别两套逻辑。
///
/// 备注更新结果：归一化后的备注串（去前后空白；纯空白返回 `None`）+ 是否触发 auto-favorite。
/// 前端用 `note` 直接回填本地镜像，避免「输入 `   ` 时前端镜像非空但 DB 为 NULL」的漂移。
#[derive(serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub struct UpdateNoteResult {
    pub note: Option<String>,
    pub auto_favorited: bool,
}

/// 写入备注；`note` 由 Rust 统一 trim + 空串归一化为 NULL。
/// auto-favorite：写入非空备注时，若 `settings.clipboard.content.autoFavorite` 开启，
/// 顺带把 `is_favorite` 置为 true（已收藏的无变化；清空备注不触发）。
#[tauri::command]
pub async fn update_clipboard_item_note(
    app: AppHandle,
    db: State<'_, DatabaseState>,
    id: String,
    note: Option<String>,
) -> Result<UpdateNoteResult> {
    let pool = db.pool().await;
    let normalized = note.as_deref().and_then(|s| {
        let trimmed = s.trim();
        if trimmed.is_empty() {
            None
        } else {
            Some(trimmed)
        }
    });
    crate::db::items::update_item_note(&pool, &id, normalized).await?;

    let mut auto_favorited = false;
    if normalized.is_some() {
        let auto_favorite = app
            .try_state::<crate::settings::SettingsStore>()
            .map(|s| s.snapshot().clipboard.content.auto_favorite)
            .unwrap_or(false);
        if auto_favorite {
            crate::db::items::mark_item_favorite(&pool, &id).await?;
            auto_favorited = true;
        }
    }
    Ok(UpdateNoteResult {
        note: normalized.map(str::to_owned),
        auto_favorited,
    })
}

/// 更新条目所属分组；分组 id 来自右键菜单的实时分组列表。
#[tauri::command]
pub async fn update_clipboard_item_group(
    db: State<'_, DatabaseState>,
    id: String,
    group_id: String,
) -> Result<()> {
    let pool = db.pool().await;
    let exists = crate::db::groups::list_groups(&pool)
        .await?
        .iter()
        .any(|group| group.id == group_id);
    if !exists {
        return Err(AppError::Clipboard("分组不存在".to_owned()));
    }

    crate::db::items::update_item_group(&pool, &id, Some(&group_id)).await
}

/// 打开条目 URL：按 `id` 取完整 `content`，trim 后用系统默认浏览器/邮件 client 打开。
/// `mailto = true` 时自动补 `mailto:` 前缀，供右键菜单「发送邮件」复用。
/// 仅适用于 text 类条目；files 类调用方应改用 `reveal_clipboard_item`。
#[tauri::command]
pub async fn open_clipboard_item_link(
    app: AppHandle,
    db: State<'_, DatabaseState>,
    id: String,
    mailto: bool,
) -> Result<()> {
    use tauri_plugin_opener::OpenerExt;

    let pool = db.pool().await;
    let item = find_item_by_id(&pool, &id)
        .await?
        .ok_or_else(|| AppError::Clipboard(format!("clipboard item not found: {id}")))?;

    let value = item.content.trim();
    if value.is_empty() {
        return Ok(());
    }

    let url = if mailto {
        format!("mailto:{value}")
    } else if value.starts_with("www.") {
        format!("https://{value}")
    } else {
        value.to_owned()
    };

    app.opener()
        .open_url(url, None::<&str>)
        .map_err(|err| AppError::Clipboard(err.to_string()))
}

/// 在系统文件管理器中显示条目对应文件：files 类取第一条路径，text 类（subKind=path）按 trim 后的字面量。
#[tauri::command]
pub async fn reveal_clipboard_item(
    app: AppHandle,
    db: State<'_, DatabaseState>,
    id: String,
) -> Result<()> {
    use tauri_plugin_opener::OpenerExt;

    let pool = db.pool().await;
    let item = find_item_by_id(&pool, &id)
        .await?
        .ok_or_else(|| AppError::Clipboard(format!("clipboard item not found: {id}")))?;

    let target = if item.kind == ClipboardKind::Files {
        item.content
            .split('\n')
            .find(|s| !s.is_empty())
            .unwrap_or("")
            .to_owned()
    } else {
        item.content.trim().to_owned()
    };

    if target.is_empty() {
        return Ok(());
    }

    app.opener()
        .reveal_item_in_dir(&target)
        .map_err(|err| AppError::Clipboard(err.to_string()))
}

/// 校验图片文件名：必须是单层 `<name>.png`，不含路径分隔符 / 父目录引用。
fn validate_image_file_name(file_name: &str) -> Result<()> {
    let invalid = file_name.is_empty()
        || file_name.contains('/')
        || file_name.contains('\\')
        || file_name.contains("..")
        || !file_name.ends_with(".png");

    if invalid {
        return Err(AppError::Clipboard(format!(
            "invalid image file name: {file_name:?}"
        )));
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::db::items::content_hash;
    use crate::db::models::Platform;
    use chrono::Utc;

    fn text_item(sub_kind: Option<ClipboardSubKind>, is_sensitive: bool) -> ClipboardItem {
        let content = "<b>secret</b>".to_owned();

        ClipboardItem {
            id: "item".to_owned(),
            kind: ClipboardKind::Text,
            sub_kind,
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
            is_sensitive,
            platform: Platform::Macos,
            note: None,
            created_at: Utc::now(),
            updated_at: Utc::now(),
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

    #[test]
    fn copy_plain_default_only_affects_text_items() {
        assert!(should_write_plain_for_copy(
            false,
            ClipboardKind::Text,
            true
        ));
        assert!(!should_write_plain_for_copy(
            false,
            ClipboardKind::Files,
            true
        ));
        assert!(!should_write_plain_for_copy(
            false,
            ClipboardKind::Image,
            true
        ));
        assert!(!should_write_plain_for_copy(
            false,
            ClipboardKind::Text,
            false
        ));
    }

    #[test]
    fn force_plain_copy_overrides_item_kind() {
        assert!(should_write_plain_for_copy(
            true,
            ClipboardKind::Files,
            false
        ));
        assert!(should_write_plain_for_copy(
            true,
            ClipboardKind::Image,
            false
        ));
    }

    #[test]
    fn paste_plain_defaults_follow_item_kind() {
        assert!(should_write_plain_for_paste(
            false,
            ClipboardKind::Text,
            true,
            false,
        ));
        assert!(should_write_plain_for_paste(
            false,
            ClipboardKind::Files,
            false,
            true,
        ));
        assert!(!should_write_plain_for_paste(
            false,
            ClipboardKind::Files,
            true,
            false,
        ));
        assert!(!should_write_plain_for_paste(
            false,
            ClipboardKind::Image,
            true,
            true,
        ));
    }

    #[test]
    fn force_plain_paste_overrides_item_kind() {
        assert!(should_write_plain_for_paste(
            true,
            ClipboardKind::Image,
            false,
            false,
        ));
    }

    #[test]
    fn mask_sensitive_text_replaces_middle_with_stars() {
        assert_eq!(
            mask_sensitive_text("sk-abcdefghijklmnopqrstuvwxyzABCDE1234567890"),
            "sk-a********7890"
        );
    }

    #[test]
    fn mask_sensitive_text_replaces_short_lines_fully() {
        assert_eq!(mask_sensitive_text("secret"), "******");
    }

    #[test]
    fn mask_sensitive_text_masks_each_line() {
        assert_eq!(
            mask_sensitive_text("abcd1234xyz\nshort"),
            "abcd***4xyz\n*****"
        );
    }

    #[test]
    fn redact_sensitive_list_item_masks_summary_when_enabled() {
        let mut item = text_item(None, true);
        item.summary = Some("sk-abcdefghijklmnopqrstuvwxyzABCDE1234567890".to_owned());

        redact_sensitive_list_item(&mut item, true);

        assert_eq!(item.summary.as_deref(), Some("sk-a********7890"));
    }

    #[test]
    fn redact_sensitive_list_item_keeps_summary_when_disabled() {
        let mut item = text_item(None, true);
        item.summary = Some("sk-abcdefghijklmnopqrstuvwxyzABCDE1234567890".to_owned());

        redact_sensitive_list_item(&mut item, false);

        assert_eq!(
            item.summary.as_deref(),
            Some("sk-abcdefghijklmnopqrstuvwxyzABCDE1234567890")
        );
    }

    #[test]
    fn preview_sub_kind_drops_sensitive_text_subtype_when_redacted() {
        let item = text_item(Some(ClipboardSubKind::Html), true);

        assert_eq!(preview_sub_kind(&item, true), None);
    }

    #[test]
    fn preview_sub_kind_keeps_sensitive_text_subtype_when_not_redacted() {
        let item = text_item(Some(ClipboardSubKind::Html), true);

        assert_eq!(preview_sub_kind(&item, false), Some(ClipboardSubKind::Html));
    }

    #[test]
    fn preview_sub_kind_keeps_regular_text_subtype() {
        let item = text_item(Some(ClipboardSubKind::Html), false);

        assert_eq!(preview_sub_kind(&item, true), Some(ClipboardSubKind::Html));
    }

    #[test]
    fn preview_text_keeps_plain_text_content() {
        let mut item = text_item(None, false);
        item.content = "plain text".to_owned();

        assert_eq!(preview_text(&item, false), "plain text");
    }

    #[test]
    fn preview_text_uses_search_text_for_html() {
        let mut item = text_item(Some(ClipboardSubKind::Html), false);
        item.content = "<b>Hello</b> World".to_owned();
        item.search_text = Some("Hello World".to_owned());

        assert_eq!(preview_text(&item, false), "Hello World");
    }

    #[test]
    fn preview_text_uses_search_text_for_rtf() {
        let mut item = text_item(Some(ClipboardSubKind::Rtf), false);
        item.content = r"{\rtf1 Hello World}".to_owned();
        item.search_text = Some("Hello World".to_owned());

        assert_eq!(preview_text(&item, false), "Hello World");
    }

    #[test]
    fn preview_text_masks_sensitive_plain_source() {
        let mut item = text_item(Some(ClipboardSubKind::Html), true);
        item.content = "<b>sk-abcdefghijklmnopqrstuvwxyzABCDE1234567890</b>".to_owned();
        item.search_text = Some("sk-abcdefghijklmnopqrstuvwxyzABCDE1234567890".to_owned());

        assert_eq!(preview_text(&item, true), "sk-a********7890");
    }

    #[test]
    fn accepts_plain_png_file_name() {
        assert!(validate_image_file_name("abcdef0123.png").is_ok());
    }

    #[test]
    fn rejects_traversal_and_subpaths() {
        for bad in [
            "",
            "evil.txt",
            "../secret.png",
            "..\\secret.png",
            "sub/dir.png",
            "a/b.png",
            "/abs.png",
            "name..png", // 含 ".." 序列，保守拒绝
        ] {
            assert!(
                validate_image_file_name(bad).is_err(),
                "should reject: {bad:?}"
            );
        }
    }
}

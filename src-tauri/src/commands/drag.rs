//! 拖拽相关命令：把条目作为 OS 级 drag-out 拖出主窗口到外部应用。

use std::path::PathBuf;

use sqlx::SqlitePool;
use tauri::{AppHandle, State};

use crate::clipboard::{icon_png, FileIconStore, ImageStore};
use crate::core::{AppError, Result};
use crate::db::items::find_item_by_id;
use crate::db::models::{ClipboardItem, ClipboardKind, ClipboardSubKind, Platform};
use crate::drag_out;
use crate::settings::Language;
use crate::window::{self, MAIN_WINDOW_LABEL};

/// 拖拽载荷：按 kind 解析成统一表示，再分派到对应的 drag-out 实现。
enum DragPayload {
    /// 文件路径列表（Files / Image kind）。
    Files(Vec<PathBuf>),
    /// 文本类型。`plain` 总有；`html` / `rtf` 二选一或都无，由 sub_kind 决定（见 ingest）：
    /// - sub_kind=Html → content=HTML 源，search_text=plain。
    /// - sub_kind=Rtf  → content=RTF 源，search_text=plain。
    /// - 其它          → content=plain。
    Text {
        plain: String,
        html: Option<String>,
        rtf: Option<String>,
    },
}

/// 把指定条目作为 OS drag 拖出主窗口。
///
/// 支持 kind：
/// - Files：从 `content` 按 `\n` 切出路径列表，过滤已删除项。
/// - Image：从 `ImageStore` 解析 `<hash>.png` 绝对路径，作为单文件 drag。
/// - Text：纯文本（`public.utf8-plain-text` / `CF_UNICODETEXT`）。
///
/// 平台差异（共用）：
/// - macOS：`beginDraggingSession` 必须主线程；用 `run_on_main_thread` fire-and-forget。
/// - Windows：`DoDragDrop` 必须在拥有窗口的线程（= Tauri 主线程）跑，否则 `QueryContinueDrag`
///   立刻 `DRAGDROP_S_CANCEL`；这里用 `run_on_main_thread` + `mpsc` 同步等待。
#[tauri::command]
pub async fn start_drag_clipboard_item(
    app: AppHandle,
    pool: State<'_, SqlitePool>,
    store: State<'_, ImageStore>,
    file_icon_store: State<'_, FileIconStore>,
    id: String,
) -> Result<()> {
    let item = find_item_by_id(&pool, &id)
        .await?
        .ok_or_else(|| AppError::Clipboard(format!("clipboard item not found: {id}")))?;

    let lang = crate::i18n::current_language(&app);
    let payload = resolve_drag_payload(&item, &store, lang)?;

    let window = window::get_window(&app, MAIN_WINDOW_LABEL)?;

    let preview = build_preview(&item, &payload, &store, &file_icon_store, &pool).await;

    #[cfg(target_os = "macos")]
    {
        let window = window.clone();
        app.run_on_main_thread(move || {
            if let Err(err) = dispatch_drag(&window, payload, preview) {
                log::error!("start drag (macos) failed: {err}");
            }
        })
        .map_err(|err| AppError::Clipboard(format!("dispatch to main thread failed: {err}")))?;
    }

    #[cfg(target_os = "windows")]
    {
        let (tx, rx) = std::sync::mpsc::channel();
        app.run_on_main_thread(move || {
            let _ = tx.send(dispatch_drag(&window, payload, preview));
        })
        .map_err(|err| AppError::Clipboard(format!("dispatch to main thread failed: {err}")))?;

        rx.recv()
            .map_err(|err| AppError::Clipboard(format!("drag result channel closed: {err}")))??;
    }

    Ok(())
}

/// 把 DragPayload 派发到对应的 drag_out::* 实现。
fn dispatch_drag(
    window: &tauri::WebviewWindow,
    payload: DragPayload,
    preview: Option<Vec<u8>>,
) -> Result<()> {
    match payload {
        DragPayload::Files(paths) => drag_out::start_drag_files(window, paths, preview),
        DragPayload::Text { plain, html, rtf } => {
            drag_out::start_drag_text(window, plain, html, rtf, preview)
        }
    }
}

/// 解析为统一的拖拽载荷。
fn resolve_drag_payload(
    item: &ClipboardItem,
    store: &ImageStore,
    lang: Language,
) -> Result<DragPayload> {
    use crate::i18n::commands::Key;

    match item.kind {
        ClipboardKind::Files => {
            let paths: Vec<PathBuf> = item
                .content
                .split('\n')
                .filter(|s| !s.is_empty())
                .map(PathBuf::from)
                .filter(|p| p.exists())
                .collect();

            if paths.is_empty() {
                return Err(AppError::Clipboard(
                    crate::i18n::commands::label(lang, Key::DragSourceFilesMissing).to_string(),
                ));
            }
            Ok(DragPayload::Files(paths))
        }
        ClipboardKind::Image => {
            let path = store.origin_path(&item.content);
            if !path.exists() {
                return Err(AppError::Clipboard(
                    crate::i18n::commands::label(lang, Key::DragImageMissing).to_string(),
                ));
            }
            Ok(DragPayload::Files(vec![path]))
        }
        ClipboardKind::Text => {
            if item.content.is_empty() {
                return Err(AppError::Clipboard(
                    crate::i18n::commands::label(lang, Key::DragTextEmpty).to_string(),
                ));
            }
            // sub_kind 语义（与 write.rs / ingest.rs 一致）：Html/Rtf 时 content 是富格式源，
            // search_text 才是 OS 提供的 plain；其他 sub_kind（url/email/color/path/None）content 即 plain。
            let (plain, html, rtf) = match item.sub_kind {
                Some(ClipboardSubKind::Html) => (
                    item.search_text
                        .clone()
                        .unwrap_or_else(|| item.content.clone()),
                    Some(item.content.clone()),
                    None,
                ),
                Some(ClipboardSubKind::Rtf) => (
                    item.search_text
                        .clone()
                        .unwrap_or_else(|| item.content.clone()),
                    None,
                    Some(item.content.clone()),
                ),
                _ => (item.content.clone(), None, None),
            };
            Ok(DragPayload::Text { plain, html, rtf })
        }
    }
}

/// 取拖拽预览图（拖拽必须秒响应，所有分支只读缓存文件，不做现场解码 / 编码）：
/// - Image：读 [`ImageStore`] 已落盘的缩略图（最长边 300px，前端首次预览时已生成），
///   命中就是「这张图本身的缩略图」，比通用 PNG 文件图标有辨识度。
/// - Files：按首个路径的 [`crate::clipboard::get_icon_cache_key`] 查 `file_type_icons` 表，
///   命中读 [`FileIconStore`] 的缓存 PNG；未命中（罕见，仅冷启动后未浏览过该扩展名时）
///   退回 [`icon_png`] 做一次性 OS 图标抽取（默认 256px，与缓存一致），不写回 DB。
/// - Text：返回 None，由 drag_out 平台层基于文本内容现场渲染（保证有辨识度），
///   避免用「来源 app 图标」这种所有文本长一样的兜底。
async fn build_preview(
    item: &ClipboardItem,
    payload: &DragPayload,
    image_store: &ImageStore,
    file_icon_store: &FileIconStore,
    pool: &SqlitePool,
) -> Option<Vec<u8>> {
    match payload {
        DragPayload::Text { .. } => None,
        DragPayload::Files(paths) => {
            if matches!(item.kind, ClipboardKind::Image) {
                if let Some(bytes) = read_image_thumbnail(image_store, &item.content) {
                    return Some(bytes);
                }
                // 缩略图缺失（首次拖拽 + 前端尚未浏览过）：回退 OS 图标，不阻塞拖拽启动。
            }
            read_cached_file_icon(pool, file_icon_store, &paths[0])
                .await
                .or_else(|| icon_png(&paths[0], None))
        }
    }
}

/// 读 `<thumbnails>/<hash[..2]>/<hash>.png` 字节。首次拖拽且前端未浏览过时缩略图可能不存在，
/// 此时返回 None 让上层走 icon 回退——不要在这里同步生成（解码大图会卡住拖拽启动）。
fn read_image_thumbnail(store: &ImageStore, file_name: &str) -> Option<Vec<u8>> {
    let path = store.thumbnail_path(file_name);
    match std::fs::read(&path) {
        Ok(b) => Some(b),
        Err(err) => {
            log::debug!("thumbnail miss for drag preview {}: {err}", path.display());
            None
        }
    }
}

/// 按文件路径的 cache_key + 当前平台查 `file_type_icons`，命中则读对应 PNG。
async fn read_cached_file_icon(
    pool: &SqlitePool,
    store: &FileIconStore,
    path: &std::path::Path,
) -> Option<Vec<u8>> {
    let cache_key = crate::clipboard::get_icon_cache_key(path);
    let platform = current_platform();
    let icon_file = match crate::db::file_icons::get_icon(pool, &cache_key, platform).await {
        Ok(Some(name)) => name,
        Ok(None) => return None,
        Err(err) => {
            log::warn!("query file_type_icons failed for drag preview: {err}");
            return None;
        }
    };
    let icon_path = store.icon_path(&icon_file);
    match std::fs::read(&icon_path) {
        Ok(b) => Some(b),
        Err(err) => {
            log::warn!(
                "read cached file icon failed {}: {err}",
                icon_path.display()
            );
            None
        }
    }
}

fn current_platform() -> Platform {
    #[cfg(target_os = "macos")]
    {
        Platform::Macos
    }
    #[cfg(target_os = "windows")]
    {
        Platform::Windows
    }
}

//! 拖拽相关命令：把条目作为 OS 级 drag-out 拖出主窗口到外部应用。

use std::path::PathBuf;

use sqlx::SqlitePool;
use tauri::{AppHandle, State};

use crate::clipboard::{icon_png, ImageStore};
use crate::core::{AppError, Result};
use crate::db::items::find_item_by_id;
use crate::db::models::{ClipboardItem, ClipboardKind, ClipboardSubKind};
use crate::drag_out;
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
    id: String,
) -> Result<()> {
    let item = find_item_by_id(&pool, &id)
        .await?
        .ok_or_else(|| AppError::Clipboard(format!("clipboard item not found: {id}")))?;

    let payload = resolve_drag_payload(&item, &store)?;

    let window = window::get_window(&app, MAIN_WINDOW_LABEL)?;

    // 平台差异：
    // - macOS：固定 256px 源 → drag_out 内 NSImage::setSize(128pt)，借 Retina backing
    //   提供 128pt 显示 + 256px 像素的清晰度。
    // - Windows：SHDRAGIMAGE.sizeDragImage 直接用 bitmap 物理像素作显示尺寸（无 backing scale
    //   概念），按窗口 DPI 算 128 * scale 物理像素。
    #[cfg(target_os = "macos")]
    let preview_size: u32 = 256;
    #[cfg(target_os = "windows")]
    let preview_size: u32 = {
        let scale = window.scale_factor().unwrap_or(1.0);
        (128.0 * scale).round() as u32
    };

    let preview = build_preview(&payload, preview_size);

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
fn resolve_drag_payload(item: &ClipboardItem, store: &ImageStore) -> Result<DragPayload> {
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
                return Err(AppError::Clipboard("拖拽源文件已不存在".to_string()));
            }
            Ok(DragPayload::Files(paths))
        }
        ClipboardKind::Image => {
            let path = store.origin_path(&item.content);
            if !path.exists() {
                return Err(AppError::Clipboard("图片文件已不存在".to_string()));
            }
            Ok(DragPayload::Files(vec![path]))
        }
        ClipboardKind::Text => {
            if item.content.is_empty() {
                return Err(AppError::Clipboard("文本内容为空".to_string()));
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

/// 取拖拽预览图：
/// - Files / Image：用首个路径抽 OS 文件 / 图片图标。
/// - Text：返回 None，由 drag_out 平台层基于文本内容现场渲染（保证有辨识度），
///   避免用「来源 app 图标」这种所有文本长一样的兜底。
fn build_preview(payload: &DragPayload, size: u32) -> Option<Vec<u8>> {
    match payload {
        DragPayload::Files(paths) => icon_png(&paths[0], Some(size)),
        DragPayload::Text { .. } => None,
    }
}

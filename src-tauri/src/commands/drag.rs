//! 拖拽相关命令：把条目作为文件 drag 出主窗口到外部应用。

use std::path::PathBuf;

use sqlx::SqlitePool;
use tauri::{AppHandle, State};

use crate::clipboard::{icon_png, ImageStore};
use crate::core::{AppError, Result};
use crate::db::items::find_item_by_id;
use crate::db::models::{ClipboardItem, ClipboardKind};
use crate::drag_out;
use crate::window::{self, MAIN_WINDOW_LABEL};

/// 把指定条目作为文件拖出主窗口。
///
/// 当前仅支持 `kind = Files / Image`：
/// - Files：从 `content` 按 `\n` 切出路径列表，过滤已删除项；空 → 报错。
/// - Image：从 `ImageStore` 解析 `<hash>.png` 的绝对路径，作为单文件 drag。
/// - Text：直接报错（drag crate 的 Data 在 Windows 是 dummy，需要后续 vendor 平台代码扩展）。
///
/// 平台差异：
/// - macOS：`beginDraggingSession` 必须主线程；用 `run_on_main_thread` 派发后立即返回，
///   drop 结果走 OS 回调（在 `drag_out` 内仅 log）。
/// - Windows：`DoDragDrop` 是阻塞调用直到 drop 完成；用 `spawn_blocking` 下沉到独立线程，
///   命令本身仍 await 至 drop 完成（保持「拖拽中前端可见到状态」的一致体验）。
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

    let paths = resolve_drag_paths(&item, &store)?;

    // 预览图：用首个路径抽 OS 文件图标；抽不到走 None（macOS 会 fallback 到 NSImage 自带识别）。
    // 该调用在 macOS 走 NSWorkspace，spawn_blocking 阻塞活，但耗时通常 < 5ms，
    // 这里同步取一次足够，不必落到 blocking pool。
    // 像素 256 = 显示 128pt @2x：macos 端 `drag_out` 固定把 NSImage size 设为 128pt，
    // 高分辨率源 PNG 作 Retina backing 提供清晰度（128pt 显示 + 256px 像素）。
    let preview = icon_png(&paths[0], Some(256));

    let window = window::get_window(&app, MAIN_WINDOW_LABEL)?;

    #[cfg(target_os = "macos")]
    {
        // run_on_main_thread 是 fire-and-forget；start_drag 本身仅注册 dragging session 后立即返回，
        // 后续光标跟随 / drop 全由 AppKit 接管。这里 await 也没意义。
        let window = window.clone();
        app.run_on_main_thread(move || {
            if let Err(err) = drag_out::start_drag_files(&window, paths, preview) {
                log::error!("start drag (macos) failed: {err}");
            }
        })
        .map_err(|err| AppError::Clipboard(format!("dispatch to main thread failed: {err}")))?;
    }

    #[cfg(target_os = "windows")]
    {
        // DoDragDrop 阻塞至 drop 完成，必须脱离 tokio worker；spawn_blocking 内部跑独立线程。
        // STA + OleInitialize 由 drag crate 内部处理。
        tauri::async_runtime::spawn_blocking(move || {
            if let Err(err) = drag_out::start_drag_files(&window, paths, preview) {
                log::error!("start drag (windows) failed: {err}");
            }
        })
        .await
        .map_err(|err| AppError::Clipboard(format!("drag blocking task join failed: {err}")))?;
    }

    Ok(())
}

/// 把条目解析成可拖拽的本地文件路径列表。
fn resolve_drag_paths(item: &ClipboardItem, store: &ImageStore) -> Result<Vec<PathBuf>> {
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
            Ok(paths)
        }
        ClipboardKind::Image => {
            let path = store.origin_path(&item.content);
            if !path.exists() {
                return Err(AppError::Clipboard("图片文件已不存在".to_string()));
            }
            Ok(vec![path])
        }
        ClipboardKind::Text => Err(AppError::Clipboard("文本拖拽暂未支持".to_string())),
    }
}

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
/// - Windows：`DoDragDrop` 是 OLE STA 调用，**必须在拥有窗口的线程**（= Tauri 主线程）上执行，
///   否则 `IDropSource::QueryContinueDrag` 立刻返回 `DRAGDROP_S_CANCEL` → 拖拽秒取消。
///   `DoDragDrop` 内部会泵消息，期间主线程被阻塞但 UI 仍可响应；同步等回调完成即可。
///   参考：drag-rs 官方 `tauri-plugin-drag` 也是这么写的。
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

    let window = window::get_window(&app, MAIN_WINDOW_LABEL)?;

    // 预览图：用首个路径抽 OS 文件图标；抽不到走 None（macOS 会 fallback 到 NSImage 自带识别）。
    // 平台差异：
    // - macOS：固定 256px 源 → `drag_out` 内 `NSImage::setSize(128pt)`，借 Retina backing
    //   提供 128pt 显示 + 256px 像素的清晰度。
    // - Windows：`SHDRAGIMAGE.sizeDragImage` 直接用 bitmap 物理像素作显示尺寸（无 backing scale
    //   概念），按窗口 DPI 算 `128 * scale` 物理像素：100% DPI 显示 128px，200% 显示 256px——
    //   既不会过大遮住窗口，也保留高 DPI 屏的清晰度。
    #[cfg(target_os = "macos")]
    let preview_size: u32 = 256;
    #[cfg(target_os = "windows")]
    let preview_size: u32 = {
        let scale = window.scale_factor().unwrap_or(1.0);
        (128.0 * scale).round() as u32
    };

    let preview = icon_png(&paths[0], Some(preview_size));

    #[cfg(target_os = "macos")]
    {
        // run_on_main_thread 是 fire-and-forget；start_drag 仅注册 dragging session 后立即返回，
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
        // 必须主线程：DoDragDrop 走 OLE STA，且要求线程拥有窗口（= Tauri 主线程）。
        // 闭包通过 mpsc 把结果回传当前 async 上下文，命令同步 await 直到 drop 完成。
        let (tx, rx) = std::sync::mpsc::channel();
        app.run_on_main_thread(move || {
            let result = drag_out::start_drag_files(&window, paths, preview);
            let _ = tx.send(result);
        })
        .map_err(|err| AppError::Clipboard(format!("dispatch to main thread failed: {err}")))?;

        rx.recv()
            .map_err(|err| AppError::Clipboard(format!("drag result channel closed: {err}")))??;
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

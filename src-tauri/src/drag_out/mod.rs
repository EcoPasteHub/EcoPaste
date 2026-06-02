//! OS 级 drag-out（把条目从列表拖到外部应用）。
//!
//! 当前只支持「文件类型」（Files / Image）：
//! - macOS：自实现（[`macos::start_drag_files`]），vendor 自 `drag` v2.1.1 但解耦了
//!   预览图像素与显示尺寸——`drag` crate 用 `NSImage::size()` 原样作显示尺寸，
//!   高分辨率 PNG 飞起来就有那么大；我们固定 `NSImage::setSize(POINT_SIZE)`。
//! - Windows：复用 `drag` crate（`IDataObject(CF_HDROP)` + `DoDragDrop`）。
//!
//! 文本 / HTML 暂不支持——`drag` crate 的 `DragItem::Data` 在 Windows 上是 dummy 实现，
//! mac 端也只声明单一 type。后续在 [`macos`] 内扩 `NSPasteboardItem` 多类型即可
//! （`public.html` / `public.utf8-plain-text` / `public.rtf`）。
//!
//! 线程模型：
//! - macOS：`beginDraggingSession` 必须在主线程。命令层用 `app.run_on_main_thread`
//!   派发，本函数本身不再切线程，由调用方保证。
//! - Windows：`DoDragDrop` 阻塞至 drop 完成，且需要 STA + `OleInitialize`。命令层
//!   用 `tauri::async_runtime::spawn_blocking` 把整段下沉到独立线程，本函数同步执行。

use std::path::PathBuf;

use tauri::WebviewWindow;

use crate::core::{AppError, Result};

#[cfg(target_os = "macos")]
pub mod macos;

/// 启动一次文件类型的 drag-out。
///
/// `preview_png` 是拖拽过程中跟随光标的预览图（PNG 字节）；为 `None` 时
/// 退回用首个路径自身解析（macOS 上 NSImage 能识别图片 / PDF，其余文件类型可能为空）。
/// 调用方应优先传入由 [`crate::clipboard::icon::icon_png`] 抽取的文件类型图标，避免触发 fallback。
pub fn start_drag_files(
    window: &WebviewWindow,
    paths: Vec<PathBuf>,
    preview_png: Option<Vec<u8>>,
) -> Result<()> {
    if paths.is_empty() {
        return Err(AppError::Clipboard("drag-out: empty path list".to_string()));
    }

    #[cfg(target_os = "macos")]
    {
        macos::start_drag_files(window, paths, preview_png, |result| {
            log::debug!("drag-out finished: {result:?}");
        })
    }

    #[cfg(target_os = "windows")]
    {
        use drag::{DragItem, Image, Options};

        let image = match preview_png {
            Some(bytes) => Image::Raw(bytes),
            None => Image::File(paths[0].clone()),
        };

        drag::start_drag(
            window,
            DragItem::Files(paths),
            image,
            |result, _cursor| {
                log::debug!("drag-out finished: {result:?}");
            },
            Options::default(),
        )
        .map_err(|err| AppError::Clipboard(format!("drag-out failed: {err}")))?;

        Ok(())
    }
}

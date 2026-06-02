//! OS 级 drag-out（把条目从列表拖到外部应用）。
//!
//! 支持类型：
//! - Files / Image：拖出本地文件路径。
//! - Text：纯文本（`public.utf8-plain-text` / `CF_UNICODETEXT`）。
//!
//! 平台实现：
//! - macOS：自实现于 [`macos`]，vendor 自 `drag` v2.1.1 但解耦了预览图像素与显示尺寸——
//!   `drag` crate 用 `NSImage::size()` 原样作显示尺寸，高分辨率 PNG 飞起来就有那么大；
//!   我们固定 `NSImage::setSize(POINT_SIZE)`。
//! - Windows：文件复用 `drag` crate（`IDataObject(CF_HDROP)` + `DoDragDrop`）；
//!   文本自实现于 [`windows`]（`drag` 的 `DragItem::Data` 在 Windows 是 dummy 实现，
//!   永远拖一个 "./" 目录而不是真实数据）。
//!
//! 线程模型：
//! - macOS：`beginDraggingSession` 必须在主线程。命令层用 `app.run_on_main_thread` 派发。
//! - Windows：`DoDragDrop` 必须在拥有窗口的线程（= Tauri 主线程）跑，且会阻塞至 drop 完成；
//!   命令层用 `run_on_main_thread` + `mpsc` 同步等待。

use std::path::PathBuf;

use tauri::WebviewWindow;

use crate::core::{AppError, Result};

#[cfg(target_os = "macos")]
pub mod macos;

#[cfg(target_os = "windows")]
pub mod windows;

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
        windows::start_drag_files(window, paths, preview_png)
    }
}

/// 启动一次文本 drag-out。
///
/// 必传 `plain`（`public.utf8-plain-text` / `CF_UNICODETEXT`），`html` / `rtf` 可选，
/// 为 `Some` 时同步声明对应的 pasteboard 类型——接收方按自身偏好选最合适的格式
/// （Word / Pages 优先 RTF；浏览器 / 富文本编辑器优先 HTML；纯文本 app 退回 plain）。
///
/// `preview_png` 用作跟随光标的预览图；mac 缺省时退回空 NSImage，Windows 暂时忽略。
pub fn start_drag_text(
    window: &WebviewWindow,
    plain: String,
    html: Option<String>,
    rtf: Option<String>,
    preview_png: Option<Vec<u8>>,
) -> Result<()> {
    if plain.is_empty() {
        return Err(AppError::Clipboard("drag-out: empty text".to_string()));
    }

    #[cfg(target_os = "macos")]
    {
        macos::start_drag_text(window, plain, html, rtf, preview_png, |result| {
            log::debug!("drag-out finished: {result:?}");
        })
    }

    #[cfg(target_os = "windows")]
    {
        let _ = window;
        windows::start_drag_text(&plain, html.as_deref(), rtf.as_deref(), preview_png)
    }
}

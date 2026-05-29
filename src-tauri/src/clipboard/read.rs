//! 剪贴板读取：基于 [`clipboard_rs`] 的薄封装，产出带类型标记的 [`ClipboardPayload`]。
//!
//! [`clipboard_rs`] 已内置 macOS（`NSPasteboard`）/ Windows 的平台读取，
//! 这里只负责按 files > image > text 的优先级归类，并过滤掉空内容。

use clipboard_rs::common::RustImage;
use clipboard_rs::{Clipboard, ClipboardContext, ContentFormat};

use super::payload::{ClipboardPayload, ImagePayload, TextPayload};
use crate::core::{AppError, Result};

/// 持有一个 [`ClipboardContext`] 的读取器。轻量，可按需创建；
/// 监听线程（2.2）会持有一个长生命周期实例复用。
pub struct ClipboardReader {
    ctx: ClipboardContext,
}

impl ClipboardReader {
    pub fn new() -> Result<Self> {
        let ctx = ClipboardContext::new().map_err(clip_err)?;
        Ok(Self { ctx })
    }

    /// 读取当前剪贴板，按 files > image > text 优先级归类为单一载荷。
    ///
    /// 返回 `None` 表示剪贴板为空或无可识别内容（监听回调应直接跳过，不入库）。
    pub fn read_all(&self) -> Result<Option<ClipboardPayload>> {
        // files 优先：复制文件时系统往往同时附带文件路径的文本表示，先判文件避免误判为 text。
        if self.ctx.has(ContentFormat::Files) {
            let files: Vec<String> = self
                .ctx
                .get_files()
                .map_err(clip_err)?
                .into_iter()
                .filter(|path| !path.is_empty())
                .collect();
            if !files.is_empty() {
                return Ok(Some(ClipboardPayload::Files(files)));
            }
        }

        // image 次之：从浏览器复制图片时常同时带 HTML，这里以图片为准。
        if self.ctx.has(ContentFormat::Image) {
            if let Some(image) = self.read_image()? {
                return Ok(Some(ClipboardPayload::Image(image)));
            }
        }

        let has_text = self.ctx.has(ContentFormat::Text);
        let has_html = self.ctx.has(ContentFormat::Html);
        let has_rtf = self.ctx.has(ContentFormat::Rtf);
        if has_text || has_html || has_rtf {
            let text = if has_text {
                self.ctx.get_text().map_err(clip_err)?
            } else {
                String::new()
            };
            let html = read_optional(has_html, || self.ctx.get_html());
            let rtf = read_optional(has_rtf, || self.ctx.get_rich_text());

            if !text.is_empty() || html.is_some() || rtf.is_some() {
                return Ok(Some(ClipboardPayload::Text(TextPayload {
                    text,
                    html,
                    rtf,
                })));
            }
        }

        Ok(None)
    }

    fn read_image(&self) -> Result<Option<ImagePayload>> {
        let image = self.ctx.get_image().map_err(clip_err)?;
        let (width, height) = image.get_size();
        if width == 0 || height == 0 {
            return Ok(None);
        }

        let bytes = image.to_png().map_err(clip_err)?.get_bytes().to_vec();
        if bytes.is_empty() {
            return Ok(None);
        }

        Ok(Some(ImagePayload {
            bytes,
            width,
            height,
        }))
    }
}

/// 仅当 `available` 时读取，读取失败或空串都归并为 `None`，
/// 让「格式存在但内容为空」与「格式不存在」对下游表现一致。
fn read_optional(
    available: bool,
    read: impl FnOnce() -> clipboard_rs::common::Result<String>,
) -> Option<String> {
    if !available {
        return None;
    }
    read().ok().filter(|value| !value.is_empty())
}

fn clip_err<E: std::fmt::Display>(err: E) -> AppError {
    AppError::Clipboard(err.to_string())
}

#[cfg(test)]
mod tests {
    use super::*;

    // 这些测试会触碰真实系统剪贴板，依赖桌面会话，CI 无头环境下会失败，
    // 故默认 ignore；本机验证用 `cargo test -- --ignored`。
    use clipboard_rs::{Clipboard, ClipboardContext};

    #[test]
    #[ignore = "touches the real system clipboard; run with --ignored on a desktop session"]
    fn round_trip_text() {
        let _guard = crate::clipboard::test_lock::serial();
        let ctx = ClipboardContext::new().unwrap();
        ctx.set_text("hello ecopaste".to_string()).unwrap();

        let payload = ClipboardReader::new()
            .unwrap()
            .read_all()
            .unwrap()
            .expect("clipboard should contain text");

        match payload {
            ClipboardPayload::Text(text) => assert_eq!(text.text, "hello ecopaste"),
            other => panic!("expected text payload, got {other:?}"),
        }
    }
}

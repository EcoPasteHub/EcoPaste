//! 剪贴板读取：基于 [`clipboard_rs`] 的薄封装，产出带类型标记的 [`ClipboardPayload`]。
//!
//! [`clipboard_rs`] 已内置 macOS（`NSPasteboard`）/ Windows 的平台读取，
//! 这里只负责按 files > image > text 的优先级归类，并过滤掉空内容。
//!
//! 图片读取走「PNG 直通优先」：源本身是 PNG 时直取原始字节、从头部解析尺寸（零解码/编码）；
//! 仅当源是 TIFF/DIB 等非 PNG 时才回退到库的解码 + 重编码 PNG。

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

    /// 读取图片为 PNG 载荷。
    ///
    /// 快路径：剪贴板原生就带 PNG（浏览器复制图片等）时，直接取原始 PNG 字节，
    /// 宽高从 PNG 头（IHDR）解析——全程零图像解码/编码。
    ///
    /// 慢路径回退：源是 TIFF（macOS 截图/多数 App）/ DIB（Windows）等非 PNG 时，
    /// 交给 [`clipboard_rs`] 解码再编码为 PNG（存储格式恒为 PNG，这次转码无法避免）。
    fn read_image(&self) -> Result<Option<ImagePayload>> {
        if let Ok(bytes) = self.ctx.get_buffer(PNG_FORMAT) {
            if let Some((width, height)) = png_dimensions(&bytes) {
                return Ok(Some(ImagePayload {
                    bytes,
                    width,
                    height,
                }));
            }
            // 拿到了 PNG 格式字节但解析不出尺寸（异常/截断）：落到下方回退，由库重新解码。
        }

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

/// 平台剪贴板里 PNG 的原始格式标识符，用于 `get_buffer` 直取原始字节。
#[cfg(target_os = "macos")]
const PNG_FORMAT: &str = "public.png";
#[cfg(target_os = "windows")]
const PNG_FORMAT: &str = "PNG";

/// 从 PNG 字节解析宽高（不解码像素）。
///
/// PNG 布局固定：8 字节签名 + 4 字节 IHDR 长度 + 4 字节 "IHDR" + 宽(大端 u32) + 高(大端 u32)。
/// 宽高位于偏移 16..24。校验签名与 IHDR 标记，任一不符返回 `None`（交回退路径处理）。
fn png_dimensions(bytes: &[u8]) -> Option<(u32, u32)> {
    const SIGNATURE: [u8; 8] = [0x89, b'P', b'N', b'G', 0x0d, 0x0a, 0x1a, 0x0a];
    if bytes.len() < 24 || bytes[..8] != SIGNATURE || &bytes[12..16] != b"IHDR" {
        return None;
    }
    let width = u32::from_be_bytes([bytes[16], bytes[17], bytes[18], bytes[19]]);
    let height = u32::from_be_bytes([bytes[20], bytes[21], bytes[22], bytes[23]]);
    if width == 0 || height == 0 {
        return None;
    }
    Some((width, height))
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

    /// 用 image crate 生成一张纯色 PNG，取其真实头部字节验证解析。
    fn sample_png(w: u32, h: u32) -> Vec<u8> {
        use std::io::Cursor;
        let buf = image::RgbaImage::from_pixel(w, h, image::Rgba([1, 2, 3, 255]));
        let mut out = Cursor::new(Vec::new());
        image::DynamicImage::ImageRgba8(buf)
            .write_to(&mut out, image::ImageFormat::Png)
            .unwrap();
        out.into_inner()
    }

    #[test]
    fn png_dimensions_reads_real_png_header() {
        assert_eq!(png_dimensions(&sample_png(123, 45)), Some((123, 45)));
    }

    #[test]
    fn png_dimensions_rejects_non_png_and_truncated() {
        // 非 PNG 签名。
        assert_eq!(png_dimensions(b"not a png at all really"), None);
        // 截断到 24 字节以下。
        assert_eq!(png_dimensions(&sample_png(8, 8)[..20]), None);
        // 空。
        assert_eq!(png_dimensions(&[]), None);
    }

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

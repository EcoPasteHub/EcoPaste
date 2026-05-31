//! 跨平台从「任意文件路径」抽取图标，落地为 PNG 字节。
//! macOS 走 NSWorkspace.iconForFile，Windows 走 SHGetFileInfo，由 `file_icon_provider` 封装。

use std::path::Path;

use image::{codecs::png::PngEncoder, ColorType, ImageEncoder};

/// App 图标 / 文件类型图标统一默认像素尺寸。
/// 128 覆盖 retina 下 ~64pt 显示，文件 ~10–20KB；过小（64）放大会糊，过大（256+）落盘冗余。
const DEFAULT_ICON_PIXEL_SIZE: u32 = 128;

/// 目录的 icon 缓存 key。`<>` 是 macOS/Windows 文件名非法字符，不会与真实路径撞 key。
pub const DIR_CACHE_KEY: &str = "<dir>";

/// 生成文件 icon 的缓存 key，用于 `file_type_icons` 表查询。
///
/// 规则：
/// - 目录：[`DIR_CACHE_KEY`]
/// - `.app` / `.exe`：完整路径（每个 bundle 单独缓存）
/// - 有扩展名：`.pdf`、`.txt`（小写、带点）
/// - 无扩展名：文件名本身（`Makefile`、`README` 等由 OS 决定是否有专属 icon）
pub fn get_icon_cache_key(path: &Path) -> String {
    if path.is_dir() {
        return DIR_CACHE_KEY.to_string();
    }

    let extname = path.extension().and_then(|s| s.to_str()).unwrap_or("");

    // .app / .exe 用完整路径
    if (cfg!(target_os = "macos") && extname == "app")
        || (cfg!(target_os = "windows") && extname == "exe")
    {
        return path.to_string_lossy().to_string();
    }

    // 无扩展名：用文件名本身（OS 会按需给 Makefile 等返回专属 icon，其余返回通用文件 icon）
    if extname.is_empty() {
        return path
            .file_name()
            .and_then(|s| s.to_str())
            .unwrap_or("")
            .to_string();
    }

    // 普通扩展名：小写、带点
    format!(".{}", extname.to_lowercase())
}

/// 抽取指定路径的图标 PNG 字节。`size` 为 `None` 时用内置默认值。
/// 失败一律返回 `None`，由调用方决定回退。
pub fn icon_png(path: &Path, size: Option<u32>) -> Option<Vec<u8>> {
    let size = size.unwrap_or(DEFAULT_ICON_PIXEL_SIZE);
    let icon = match file_icon_provider::get_file_icon(path, size as u16) {
        Ok(i) => i,
        Err(err) => {
            log::warn!(
                "icon_png: get_file_icon failed for {}: {err:?}",
                path.display()
            );
            return None;
        }
    };
    let mut out = Vec::with_capacity((size * size * 4) as usize / 2);
    let encoder = PngEncoder::new(&mut out);
    if let Err(err) = encoder.write_image(
        &icon.pixels,
        icon.width,
        icon.height,
        ColorType::Rgba8.into(),
    ) {
        log::warn!("icon_png: encode PNG failed for {}: {err}", path.display());
        return None;
    }
    Some(out)
}

#[cfg(all(test, target_os = "macos"))]
mod tests {
    use super::*;
    use std::path::PathBuf;

    #[test]
    fn icon_png_for_finder_returns_bytes() {
        let path = PathBuf::from("/System/Library/CoreServices/Finder.app");
        let png = icon_png(&path, None).expect("expected PNG bytes");
        assert!(png.len() > 100, "PNG too small: {}", png.len());
        // PNG 签名
        assert_eq!(&png[..8], b"\x89PNG\r\n\x1a\n");
        println!("Finder.app icon PNG size: {} bytes", png.len());
    }
}

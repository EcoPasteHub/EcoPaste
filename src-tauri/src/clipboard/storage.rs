//! 图片落盘：按内容哈希分片存储原图与缩略图。
//!
//! 目录布局（`content` 字段存文件名 `<sha256>.png`，分片目录由此函数推导，不入库）：
//! ```text
//! <app_local_data>/resources/images/
//!   origin/<hash[..2]>/<sha256>.png       原图（PNG）
//!   thumbnails/<hash[..2]>/<sha256>.png   缩略图（PNG，最长边 <= THUMBNAIL_MAX）
//! ```
//! 文件名取「PNG 字节的 sha256」：同一张图重复复制 → 同字节 → 同文件名，落盘幂等，
//! 且与阶段 1.4 的去重指纹同源（image 的 `content_hash` 即对 PNG 字节哈希）。
//! 按 hash 前 2 位 hex 分 256 个子目录，避免重度使用下单目录文件爆量。

use std::path::{Path, PathBuf};

use anyhow::Context;
use clipboard_rs::common::{RustImage, RustImageData};
use sha2::{Digest, Sha256};
use tauri::{AppHandle, Manager};

use super::payload::ImagePayload;
use crate::core::{AppError, Result};

/// 缩略图最长边像素。仅用于列表预览，够清晰即可。
const THUMBNAIL_MAX: u32 = 300;

const RESOURCES_DIR: &str = "resources";
const IMAGES_DIR: &str = "images";
const ORIGIN_DIR: &str = "origin";
const THUMBNAILS_DIR: &str = "thumbnails";

/// 一次图片落盘的结果，交给 ingest 写入 `ClipboardItem`。
pub struct StoredImage {
    /// 入库 `content`：图片文件名 `<sha256>.png`（不含分片目录）。
    pub file_name: String,
    /// 去重指纹来源：PNG 字节的 sha256（十六进制）。
    pub sha256: String,
    pub width: i64,
    pub height: i64,
    /// 原图字节数。
    pub size: i64,
}

/// 图片存储器：持有 app data 下的 `resources/images` 根目录。
/// 放入 Tauri `State`，监听线程与命令共用。
#[derive(Clone)]
pub struct ImageStore {
    images_root: PathBuf,
}

impl ImageStore {
    /// 从 `AppHandle` 解析 `<app_local_data>/resources/images` 作为根。
    pub fn new(app: &AppHandle) -> Result<Self> {
        let base = app
            .path()
            .app_local_data_dir()
            .context("failed to resolve app local data dir")?;
        let images_root = base.join(RESOURCES_DIR).join(IMAGES_DIR);
        Ok(Self { images_root })
    }

    #[cfg(test)]
    pub(crate) fn for_test(images_root: PathBuf) -> Self {
        Self { images_root }
    }

    /// 落盘原图 + 缩略图，返回 [`StoredImage`]。已存在的文件跳过写入（幂等）。
    pub fn store(&self, image: &ImagePayload) -> Result<StoredImage> {
        let sha256 = sha256_hex(&image.bytes);
        let file_name = format!("{sha256}.png");

        let origin_path = self.shard_path(ORIGIN_DIR, &sha256, &file_name);
        write_if_absent(&origin_path, &image.bytes)?;

        let thumb_path = self.shard_path(THUMBNAILS_DIR, &sha256, &file_name);
        if !thumb_path.exists() {
            let thumb_bytes = encode_thumbnail(&image.bytes)?;
            write_if_absent(&thumb_path, &thumb_bytes)?;
        }

        Ok(StoredImage {
            file_name,
            sha256,
            width: i64::from(image.width),
            height: i64::from(image.height),
            size: image.bytes.len() as i64,
        })
    }

    /// 由文件名解析原图绝对路径（分片目录从文件名前 2 位推导）。供写回/粘贴（阶段 4）使用。
    pub fn origin_path(&self, file_name: &str) -> PathBuf {
        self.shard_path(ORIGIN_DIR, shard_key(file_name), file_name)
    }

    /// 由文件名解析缩略图绝对路径。供前端预览取图。
    pub fn thumbnail_path(&self, file_name: &str) -> PathBuf {
        self.shard_path(THUMBNAILS_DIR, shard_key(file_name), file_name)
    }

    fn shard_path(&self, kind_dir: &str, shard_src: &str, file_name: &str) -> PathBuf {
        self.images_root
            .join(kind_dir)
            .join(shard_dir(shard_src))
            .join(file_name)
    }
}

/// 分片子目录名：取来源串前 2 个字符（sha256 恒为 hex，必有 2 位）；
/// 异常短串兜底为 `00`，保证始终有一层分片。
fn shard_dir(src: &str) -> &str {
    if src.len() >= 2 {
        &src[..2]
    } else {
        "00"
    }
}

/// 从文件名 `<sha256>.png` 取分片来源（即 sha256 本身）。
fn shard_key(file_name: &str) -> &str {
    file_name.split('.').next().unwrap_or(file_name)
}

fn sha256_hex(bytes: &[u8]) -> String {
    let mut hasher = Sha256::new();
    hasher.update(bytes);
    format!("{:x}", hasher.finalize())
}

/// 写文件（自动建分片目录）；目标已存在则跳过，保证幂等且不重复 IO。
fn write_if_absent(path: &Path, bytes: &[u8]) -> Result<()> {
    if path.exists() {
        return Ok(());
    }
    if let Some(parent) = path.parent() {
        std::fs::create_dir_all(parent)
            .with_context(|| format!("failed to create image dir {parent:?}"))?;
    }
    std::fs::write(path, bytes).with_context(|| format!("failed to write image {path:?}"))?;
    Ok(())
}

/// 把原图 PNG 字节解码 → 生成缩略图（最长边 <= [`THUMBNAIL_MAX`]，保持比例）→ 重新编码 PNG。
fn encode_thumbnail(png_bytes: &[u8]) -> Result<Vec<u8>> {
    let image = RustImageData::from_bytes(png_bytes).map_err(clip_err)?;
    let thumb = image
        .thumbnail(THUMBNAIL_MAX, THUMBNAIL_MAX)
        .map_err(clip_err)?;
    Ok(thumb.to_png().map_err(clip_err)?.get_bytes().to_vec())
}

fn clip_err<E: std::fmt::Display>(err: E) -> AppError {
    AppError::Clipboard(err.to_string())
}

#[cfg(test)]
mod tests {
    use super::*;

    /// 用 image crate 生成一张纯色 PNG 作测试输入。
    fn sample_png(w: u32, h: u32) -> Vec<u8> {
        use std::io::Cursor;
        let buf = image::RgbaImage::from_pixel(w, h, image::Rgba([10, 20, 30, 255]));
        let mut out = Cursor::new(Vec::new());
        image::DynamicImage::ImageRgba8(buf)
            .write_to(&mut out, image::ImageFormat::Png)
            .unwrap();
        out.into_inner()
    }

    fn temp_store() -> (tempdir_guard::TempDir, ImageStore) {
        let dir = tempdir_guard::TempDir::new();
        let store = ImageStore::for_test(dir.path().join("resources").join("images"));
        (dir, store)
    }

    #[test]
    fn stores_origin_and_thumbnail_under_hash_shard() {
        let (_dir, store) = temp_store();
        let payload = ImagePayload {
            bytes: sample_png(64, 48),
            width: 64,
            height: 48,
        };

        let stored = store.store(&payload).unwrap();
        assert!(stored.file_name.ends_with(".png"));
        assert_eq!(stored.file_name, format!("{}.png", stored.sha256));
        assert_eq!(stored.width, 64);
        assert_eq!(stored.height, 48);
        assert!(stored.size > 0);

        // 原图与缩略图都落在 <前2位>/<file_name>。
        let origin = store.origin_path(&stored.file_name);
        let thumb = store.thumbnail_path(&stored.file_name);
        assert!(origin.exists(), "origin should exist: {origin:?}");
        assert!(thumb.exists(), "thumbnail should exist: {thumb:?}");
        assert_eq!(
            origin.parent().unwrap().file_name().unwrap().to_str(),
            Some(&stored.sha256[..2])
        );
        // 原图字节与输入一致（未改动）。
        assert_eq!(std::fs::read(&origin).unwrap(), payload.bytes);
    }

    #[test]
    fn store_is_idempotent_for_same_bytes() {
        let (_dir, store) = temp_store();
        let payload = ImagePayload {
            bytes: sample_png(32, 32),
            width: 32,
            height: 32,
        };

        let a = store.store(&payload).unwrap();
        let b = store.store(&payload).unwrap();
        // 同字节 → 同文件名 / 同 sha256，幂等。
        assert_eq!(a.file_name, b.file_name);
        assert_eq!(a.sha256, b.sha256);
    }

    #[test]
    fn path_resolution_matches_store_layout() {
        let (_dir, store) = temp_store();
        let file_name = "abcdef0123456789.png";
        assert_eq!(
            store.origin_path(file_name),
            store.images_root.join("origin").join("ab").join(file_name)
        );
        assert_eq!(
            store.thumbnail_path(file_name),
            store
                .images_root
                .join("thumbnails")
                .join("ab")
                .join(file_name)
        );
    }

    /// 极简自清理临时目录（避免引第三方 tempfile 依赖）。
    mod tempdir_guard {
        use std::path::{Path, PathBuf};

        pub struct TempDir(PathBuf);

        impl TempDir {
            pub fn new() -> Self {
                let path = std::env::temp_dir()
                    .join(format!("ecopaste-imgstore-{}", uuid::Uuid::new_v4()));
                std::fs::create_dir_all(&path).unwrap();
                Self(path)
            }
            pub fn path(&self) -> &Path {
                &self.0
            }
        }

        impl Drop for TempDir {
            fn drop(&mut self) {
                std::fs::remove_dir_all(&self.0).ok();
            }
        }
    }
}

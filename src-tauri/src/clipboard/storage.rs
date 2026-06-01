//! 图片落盘：按内容哈希分片存储原图与缩略图。
//!
//! 目录布局（`content` 字段存文件名 `<hash>.png`，分片目录由此函数推导，不入库）：
//! ```text
//! <app_local_data>/resources/clipboard-images/
//!   origin/<hash[..2]>/<hash>.png       原图（PNG），复制时落盘
//!   thumbnails/<hash[..2]>/<hash>.png   缩略图（PNG，最长边 <= THUMBNAIL_MAX），首次预览时按需生成
//! ```
//! 文件名取「PNG 字节的 blake3」：同一张图重复复制 → 同字节 → 同文件名，落盘幂等，
//! 且与去重指纹同源（image 的 `content_hash` 即对 PNG 字节哈希）。
//! 按 hash 前 2 位 hex 分 256 个子目录，避免重度使用下单目录文件爆量。
//!
//! 缩略图的解码/缩放/编码不在复制热路径上——`store` 只写原图，缩略图由
//! [`ImageStore::ensure_thumbnail`] 在前端首次取图时懒生成并缓存。

use std::path::{Path, PathBuf};

use anyhow::Context;
use blake3::Hasher;
use clipboard_rs::common::{RustImage, RustImageData};
use tauri::AppHandle;

use super::payload::ImagePayload;
use crate::core::{AppError, Result};

/// 缩略图最长边像素。仅用于列表预览，够清晰即可。
const THUMBNAIL_MAX: u32 = 300;

/// 剪贴板图片目录名，挂在 `core::paths::resources_dir` 下（与 `app-icons` 并列）。
const IMAGES_DIR: &str = "clipboard-images";
const ORIGIN_DIR: &str = "origin";
const THUMBNAILS_DIR: &str = "thumbnails";

/// 一次图片落盘的结果，交给 ingest 写入 `ClipboardItem`。
pub struct StoredImage {
    /// 入库 `content`：图片文件名 `<hash>.png`（不含分片目录）。
    pub file_name: String,
    /// 去重指纹来源：PNG 字节的 blake3（十六进制）。
    #[allow(dead_code)]
    pub content_digest: String,
    pub width: i64,
    pub height: i64,
    /// 原图字节数。
    pub size: i64,
}

/// 图片存储器：持有 app data 下的 `resources/clipboard-images` 根目录。
/// 放入 Tauri `State`，监听线程与命令共用。
#[derive(Clone)]
pub struct ImageStore {
    images_root: PathBuf,
}

impl ImageStore {
    /// 从 `AppHandle` 解析 `<app_local_data>/resources/clipboard-images` 作为根。
    pub fn new(app: &AppHandle) -> Result<Self> {
        let images_root = crate::core::paths::resources_dir(app)?.join(IMAGES_DIR);
        Ok(Self { images_root })
    }

    #[cfg(test)]
    pub(crate) fn for_test(images_root: PathBuf) -> Self {
        Self { images_root }
    }

    /// 落盘原图，返回 [`StoredImage`]。已存在的文件跳过写入（幂等）。
    ///
    /// 缩略图**不在此生成**——它已移出复制热路径，改由 [`Self::ensure_thumbnail`] 在
    /// 前端首次预览时按需生成并缓存。复制路径只做「哈希 + 写原图」，避免大图编解码卡顿。
    pub fn store(&self, image: &ImagePayload) -> Result<StoredImage> {
        let content_digest = blake3_hex(&image.bytes);
        let file_name = format!("{content_digest}.png");

        let origin_path = self.shard_path(ORIGIN_DIR, &content_digest, &file_name);
        write_if_absent(&origin_path, &image.bytes)?;

        Ok(StoredImage {
            file_name,
            content_digest,
            width: i64::from(image.width),
            height: i64::from(image.height),
            size: image.bytes.len() as i64,
        })
    }

    /// 确保缩略图存在并返回其绝对路径：已存在直接返回；否则读原图 → 解码 → 缩放 → 编码 PNG → 落盘。
    ///
    /// 供 `get_clipboard_image_path(thumbnail=true)` 调用。把生成放在「读」而非「写」侧，
    /// 既将解码/编码移出复制热路径，又因「返回前文件已确保存在」天然避免前端加载到半成品文件。
    pub fn ensure_thumbnail(&self, file_name: &str) -> Result<PathBuf> {
        let thumb_path = self.thumbnail_path(file_name);
        if thumb_path.exists() {
            return Ok(thumb_path);
        }

        let origin_path = self.origin_path(file_name);
        let origin_bytes = std::fs::read(&origin_path)
            .with_context(|| format!("failed to read origin image {origin_path:?}"))?;
        let thumb_bytes = encode_thumbnail(&origin_bytes)?;
        write_if_absent(&thumb_path, &thumb_bytes)?;
        Ok(thumb_path)
    }

    /// 删除一张图片的原图与缩略图。缩略图懒生成、可能不存在，缺失文件视作成功（幂等）。
    /// 删后顺手清理变空的分片目录（`origin/<ab>`、`thumbnails/<ab>`）。
    ///
    /// 调用前提：库里该图至多一行（image 去重指纹源自 PNG 字节，落盘文件名即字节哈希），
    /// 故删行后该文件必为孤儿，可直接删，无需引用计数。其余 IO 错误上抛由调用方记日志。
    pub fn remove(&self, file_name: &str) -> Result<()> {
        let origin = self.origin_path(file_name);
        let thumb = self.thumbnail_path(file_name);
        remove_if_present(&origin)?;
        remove_if_present(&thumb)?;
        // 分片目录可能被同前缀的其他图共享，非空时保留——remove_dir 只删空目录，
        // 非空 / 不存在都返回 Err，一并忽略；目录清理是尽力而为，不影响删图结果。
        remove_dir_if_empty(origin.parent());
        remove_dir_if_empty(thumb.parent());
        Ok(())
    }

    /// 由文件名解析原图绝对路径（分片目录从文件名前 2 位推导）。供写回/粘贴使用。
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

/// 分片子目录名：取来源串前 2 个字符（hash 恒为 hex，必有 2 位）；
/// 异常短串兜底为 `00`，保证始终有一层分片。
fn shard_dir(src: &str) -> &str {
    if src.len() >= 2 {
        &src[..2]
    } else {
        "00"
    }
}

/// 从文件名 `<hash>.png` 取分片来源（即 hash 本身）。
fn shard_key(file_name: &str) -> &str {
    file_name.split('.').next().unwrap_or(file_name)
}

fn blake3_hex(bytes: &[u8]) -> String {
    let mut hasher = Hasher::new();
    hasher.update(bytes);
    hasher.finalize().to_hex().to_string()
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

/// 删文件；文件不存在时静默成功（幂等），其余 IO 错误上抛。
fn remove_if_present(path: &Path) -> Result<()> {
    match std::fs::remove_file(path) {
        Ok(()) => Ok(()),
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => Ok(()),
        Err(err) => Err(AppError::from(
            anyhow::Error::new(err).context(format!("failed to remove image {path:?}")),
        )),
    }
}

/// 尽力删除空目录：`remove_dir` 仅在目录为空时成功，非空（仍有同分片的其他图）/
/// 不存在都返回 `Err`，一律忽略——目录清理不影响删图结果，无需上抛。
fn remove_dir_if_empty(dir: Option<&Path>) {
    if let Some(dir) = dir {
        let _ = std::fs::remove_dir(dir);
    }
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
        let store = ImageStore::for_test(dir.path().join("resources").join("clipboard-images"));
        (dir, store)
    }

    #[test]
    fn stores_origin_under_hash_shard_without_thumbnail() {
        let (_dir, store) = temp_store();
        let payload = ImagePayload {
            bytes: sample_png(64, 48),
            width: 64,
            height: 48,
        };

        let stored = store.store(&payload).unwrap();
        assert!(stored.file_name.ends_with(".png"));
        assert_eq!(stored.file_name, format!("{}.png", stored.content_digest));
        assert_eq!(stored.width, 64);
        assert_eq!(stored.height, 48);
        assert!(stored.size > 0);

        // 原图落在 <前2位>/<file_name>；缩略图此刻尚未生成（移出热路径）。
        let origin = store.origin_path(&stored.file_name);
        let thumb = store.thumbnail_path(&stored.file_name);
        assert!(origin.exists(), "origin should exist: {origin:?}");
        assert!(
            !thumb.exists(),
            "thumbnail should NOT exist before ensure_thumbnail: {thumb:?}"
        );
        assert_eq!(
            origin.parent().unwrap().file_name().unwrap().to_str(),
            Some(&stored.content_digest[..2])
        );
        // 原图字节与输入一致（未改动）。
        assert_eq!(std::fs::read(&origin).unwrap(), payload.bytes);
    }

    #[test]
    fn ensure_thumbnail_generates_then_caches() {
        let (_dir, store) = temp_store();
        let payload = ImagePayload {
            bytes: sample_png(64, 48),
            width: 64,
            height: 48,
        };
        let stored = store.store(&payload).unwrap();

        // 首次：从原图生成缩略图并返回其路径。
        let thumb = store.ensure_thumbnail(&stored.file_name).unwrap();
        assert!(thumb.exists(), "thumbnail should be generated: {thumb:?}");
        assert_eq!(thumb, store.thumbnail_path(&stored.file_name));
        let first_bytes = std::fs::read(&thumb).unwrap();
        assert!(!first_bytes.is_empty());

        // 再次：命中缓存，路径一致、内容不变（幂等，不重复编码）。
        let thumb2 = store.ensure_thumbnail(&stored.file_name).unwrap();
        assert_eq!(thumb, thumb2);
        assert_eq!(std::fs::read(&thumb2).unwrap(), first_bytes);
    }

    #[test]
    fn ensure_thumbnail_errors_when_origin_missing() {
        let (_dir, store) = temp_store();
        // 原图从未落盘：ensure_thumbnail 读不到原图，报错而非 panic。
        let result = store.ensure_thumbnail("0000000000000000.png");
        assert!(result.is_err());
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
        // 同字节 → 同文件名 / 同 digest，幂等。
        assert_eq!(a.file_name, b.file_name);
        assert_eq!(a.content_digest, b.content_digest);
    }

    #[test]
    fn remove_deletes_origin_and_thumbnail_idempotently() {
        let (_dir, store) = temp_store();
        let payload = ImagePayload {
            bytes: sample_png(40, 30),
            width: 40,
            height: 30,
        };
        let stored = store.store(&payload).unwrap();
        store.ensure_thumbnail(&stored.file_name).unwrap();

        let origin = store.origin_path(&stored.file_name);
        let thumb = store.thumbnail_path(&stored.file_name);
        assert!(origin.exists() && thumb.exists());

        store.remove(&stored.file_name).unwrap();
        assert!(!origin.exists(), "origin should be removed");
        assert!(!thumb.exists(), "thumbnail should be removed");
        // 分片目录已空 → 一并清理。
        assert!(
            !origin.parent().unwrap().exists(),
            "empty origin shard dir should be removed"
        );
        assert!(
            !thumb.parent().unwrap().exists(),
            "empty thumbnail shard dir should be removed"
        );

        // 再次删除：文件已不存在，仍成功（幂等）。
        store.remove(&stored.file_name).unwrap();
    }

    #[test]
    fn remove_keeps_shard_dir_when_other_image_shares_prefix() {
        let (_dir, store) = temp_store();
        let payload = ImagePayload {
            bytes: sample_png(40, 30),
            width: 40,
            height: 30,
        };
        let stored = store.store(&payload).unwrap();
        let shard = store
            .origin_path(&stored.file_name)
            .parent()
            .unwrap()
            .to_path_buf();

        // 模拟同前缀的另一张图占用同一分片目录。
        let sibling = shard.join("sibling.png");
        std::fs::write(&sibling, b"x").unwrap();

        store.remove(&stored.file_name).unwrap();
        // 目标图已删，但分片目录非空 → 必须保留，sibling 不受影响。
        assert!(!store.origin_path(&stored.file_name).exists());
        assert!(shard.exists(), "non-empty shard dir must be kept");
        assert!(sibling.exists(), "sibling image must survive");
    }

    #[test]
    fn remove_succeeds_when_thumbnail_never_generated() {
        let (_dir, store) = temp_store();
        let payload = ImagePayload {
            bytes: sample_png(16, 16),
            width: 16,
            height: 16,
        };
        let stored = store.store(&payload).unwrap();
        // 只落了原图，缩略图从未生成；remove 不应因缩略图缺失而失败。
        assert!(!store.thumbnail_path(&stored.file_name).exists());
        store.remove(&stored.file_name).unwrap();
        assert!(!store.origin_path(&stored.file_name).exists());
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

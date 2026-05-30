//! 应用 icon 落盘：按 PNG 字节 sha256 命名，与 [`super::storage::ImageStore`] 同套分片布局。
//!
//! 目录：`<app_local_data>/resources/app-icons/<hash[..2]>/<sha256>.png`。
//! 同 icon（同字节）天然去重：不同应用若指向相同 icon 二进制只占一份。
//! `clipboard_apps.icon_file` 存「`<sha256>.png`」纯文件名，分片目录从文件名推导，不入库。

use std::path::{Path, PathBuf};

use anyhow::Context;
use sha2::{Digest, Sha256};
use tauri::{AppHandle, Manager};

use crate::core::Result;

const RESOURCES_DIR: &str = "resources";
const APP_ICONS_DIR: &str = "app-icons";

#[derive(Clone)]
pub struct AppIconStore {
    root: PathBuf,
}

impl AppIconStore {
    pub fn new(app: &AppHandle) -> Result<Self> {
        let base = app
            .path()
            .app_local_data_dir()
            .context("failed to resolve app local data dir")?;
        Ok(Self {
            root: base.join(RESOURCES_DIR).join(APP_ICONS_DIR),
        })
    }

    #[cfg(test)]
    pub(crate) fn for_test(root: PathBuf) -> Self {
        Self { root }
    }

    /// 落盘 PNG 字节，返回入库用文件名 `<sha256>.png`。已存在则跳过写入（幂等）。
    pub fn store(&self, png_bytes: &[u8]) -> Result<String> {
        let sha = sha256_hex(png_bytes);
        let file_name = format!("{sha}.png");
        let path = self.shard_path(&sha, &file_name);
        write_if_absent(&path, png_bytes)?;
        Ok(file_name)
    }

    pub fn icon_path(&self, file_name: &str) -> PathBuf {
        self.shard_path(shard_key(file_name), file_name)
    }

    fn shard_path(&self, shard_src: &str, file_name: &str) -> PathBuf {
        self.root.join(shard_dir(shard_src)).join(file_name)
    }
}

fn shard_dir(src: &str) -> &str {
    if src.len() >= 2 {
        &src[..2]
    } else {
        "00"
    }
}

fn shard_key(file_name: &str) -> &str {
    file_name.split('.').next().unwrap_or(file_name)
}

fn sha256_hex(bytes: &[u8]) -> String {
    let mut hasher = Sha256::new();
    hasher.update(bytes);
    format!("{:x}", hasher.finalize())
}

fn write_if_absent(path: &Path, bytes: &[u8]) -> Result<()> {
    if path.exists() {
        return Ok(());
    }
    if let Some(parent) = path.parent() {
        std::fs::create_dir_all(parent)
            .with_context(|| format!("failed to create app-icons dir {parent:?}"))?;
    }
    std::fs::write(path, bytes).with_context(|| format!("failed to write app icon {path:?}"))?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    fn temp_store() -> (TempDir, AppIconStore) {
        let dir = TempDir::new();
        let store = AppIconStore::for_test(dir.path().join("app-icons"));
        (dir, store)
    }

    #[test]
    fn stores_under_hash_shard_and_is_idempotent() {
        let (_d, store) = temp_store();
        let bytes = b"fake-png-bytes-for-test".to_vec();

        let a = store.store(&bytes).unwrap();
        let b = store.store(&bytes).unwrap();
        assert_eq!(a, b, "same bytes -> same file name");
        assert!(a.ends_with(".png"));

        let path = store.icon_path(&a);
        assert!(path.exists());
        assert_eq!(std::fs::read(&path).unwrap(), bytes);
        // 分片目录 = 文件名前 2 位 hex。
        assert_eq!(
            path.parent()
                .unwrap()
                .file_name()
                .unwrap()
                .to_str()
                .unwrap(),
            &a[..2]
        );
    }

    struct TempDir(PathBuf);
    impl TempDir {
        fn new() -> Self {
            let p = std::env::temp_dir().join(format!("ecopaste-appicon-{}", uuid::Uuid::new_v4()));
            std::fs::create_dir_all(&p).unwrap();
            Self(p)
        }
        fn path(&self) -> &Path {
            &self.0
        }
    }
    impl Drop for TempDir {
        fn drop(&mut self) {
            std::fs::remove_dir_all(&self.0).ok();
        }
    }
}

//! 文件类型 icon 落盘：按 PNG 字节 blake3 命名，平铺在单层目录下。
//!
//! 目录：`<app_local_data>/resources/file-icons/<hash>.png`。
//! 与 `app-icons/` 分开，避免应用 icon 和文件 icon 混在一起难以管理。
//! `file_type_icons.icon_file` 存「`<hash>.png`」纯文件名，不入库目录前缀。

use std::path::{Path, PathBuf};

use anyhow::Context;
use blake3::Hasher;
use tauri::AppHandle;

use crate::core::Result;

const FILE_ICONS_DIR: &str = "file-icons";

#[derive(Clone)]
pub struct FileIconStore {
    root: PathBuf,
}

impl FileIconStore {
    pub fn new(app: &AppHandle) -> Result<Self> {
        Ok(Self {
            root: crate::core::paths::resources_dir(app)?.join(FILE_ICONS_DIR),
        })
    }

    #[cfg(test)]
    pub(crate) fn for_test(root: PathBuf) -> Self {
        Self { root }
    }

    /// 落盘 PNG 字节，返回入库用文件名 `<hash>.png`。已存在则跳过写入（幂等）。
    pub fn store(&self, png_bytes: &[u8]) -> Result<String> {
        let digest = blake3_hex(png_bytes);
        let file_name = format!("{digest}.png");
        write_if_absent(&self.root.join(&file_name), png_bytes)?;
        Ok(file_name)
    }

    pub fn icon_path(&self, file_name: &str) -> PathBuf {
        self.root.join(file_name)
    }
}

fn blake3_hex(bytes: &[u8]) -> String {
    let mut hasher = Hasher::new();
    hasher.update(bytes);
    hasher.finalize().to_hex().to_string()
}

fn write_if_absent(path: &Path, bytes: &[u8]) -> Result<()> {
    if path.exists() {
        return Ok(());
    }
    if let Some(parent) = path.parent() {
        std::fs::create_dir_all(parent)
            .with_context(|| format!("failed to create file-icons dir {parent:?}"))?;
    }
    std::fs::write(path, bytes).with_context(|| format!("failed to write file icon {path:?}"))?;
    Ok(())
}

//! 剪贴板相关命令：手动重新读取、解析图片路径。供前端按需触发。

use serde::Serialize;
use sqlx::SqlitePool;
use tauri::{AppHandle, State};

use crate::clipboard::{build_item, persist_and_notify, ClipboardReader, ImageStore};
use crate::core::{AppError, Result};
use crate::db::models::ClipboardItem;

/// `read_clipboard` 的返回：入库后的记录 + 是否命中去重（前端据此决定提示/滚动行为）。
#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ReadClipboardResult {
    pub item: ClipboardItem,
    pub deduplicated: bool,
    /// 剪贴板为空 / 无可识别内容时为 `false`，此时 `item` 为 `None`。
    pub captured: bool,
}

/// 手动读取当前剪贴板并入库（「重新读取」按钮）。
///
/// 复用监听管线：read_all → build_item（含图片落盘）→ persist_and_notify（去重入库 + emit）。
/// 与 OS 监听走同一条入库路径，语义一致；emit 同样触发前端列表刷新。
///
/// `ClipboardReader` 持有的 `ClipboardContext` 是 `!Send`，故读取与转换全部在
/// await 之前的同步块内完成并 drop，之后才进入异步入库——保证命令 future 满足 `Send`。
#[tauri::command]
pub async fn read_clipboard(
    app: AppHandle,
    pool: State<'_, SqlitePool>,
    store: State<'_, ImageStore>,
) -> Result<Option<ReadClipboardResult>> {
    let item: Option<ClipboardItem> = {
        let reader = ClipboardReader::new()?;
        match reader.read_all()? {
            Some(payload) => build_item(&store, &payload)?,
            None => None,
        }
        // reader 在此 drop，!Send 句柄不跨下方 await。
    };

    let Some(item) = item else {
        return Ok(None);
    };

    let result = persist_and_notify(&app, &pool, &item).await?;
    Ok(Some(ReadClipboardResult {
        item,
        deduplicated: result.deduplicated,
        captured: true,
    }))
}

/// 把入库的图片文件名解析为磁盘绝对路径，供前端预览取图。
/// `thumbnail = true` 取缩略图，否则取原图。
///
/// `file_name` 来自前端（即记录的 `content`），是唯一的外部输入，需防路径穿越：
/// 仅允许「单层、纯 `<hash>.png` 形态」的文件名，含分隔符 / `..` / 子目录一律拒绝。
#[tauri::command]
pub async fn get_clipboard_image_path(
    store: State<'_, ImageStore>,
    file_name: String,
    thumbnail: bool,
) -> Result<String> {
    validate_image_file_name(&file_name)?;

    let path = if thumbnail {
        store.thumbnail_path(&file_name)
    } else {
        store.origin_path(&file_name)
    };

    path.to_str()
        .map(str::to_owned)
        .ok_or_else(|| AppError::Clipboard("image path is not valid utf-8".to_owned()))
}

/// 校验图片文件名：必须是单层 `<name>.png`，不含路径分隔符 / 父目录引用。
fn validate_image_file_name(file_name: &str) -> Result<()> {
    let invalid = file_name.is_empty()
        || file_name.contains('/')
        || file_name.contains('\\')
        || file_name.contains("..")
        || !file_name.ends_with(".png");

    if invalid {
        return Err(AppError::Clipboard(format!(
            "invalid image file name: {file_name:?}"
        )));
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn accepts_plain_png_file_name() {
        assert!(validate_image_file_name("abcdef0123.png").is_ok());
    }

    #[test]
    fn rejects_traversal_and_subpaths() {
        for bad in [
            "",
            "evil.txt",
            "../secret.png",
            "..\\secret.png",
            "sub/dir.png",
            "a/b.png",
            "/abs.png",
            "name..png", // 含 ".." 序列，保守拒绝
        ] {
            assert!(
                validate_image_file_name(bad).is_err(),
                "should reject: {bad:?}"
            );
        }
    }
}

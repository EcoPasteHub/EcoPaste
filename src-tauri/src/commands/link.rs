use anyhow::Context;
use tauri::AppHandle;
use tauri_plugin_opener::OpenerExt;

use crate::core::Result;

/// 打开经过白名单校验的外部网页 URL。
#[tauri::command]
pub async fn open_external_url(app: AppHandle, url: String) -> Result<()> {
    let normalized = url.trim();
    if !(normalized.starts_with("https://") || normalized.starts_with("http://")) {
        return Err(anyhow::Error::msg("only http and https urls can be opened").into());
    }

    app.opener()
        .open_url(normalized, None::<&str>)
        .context("failed to open external url")?;

    Ok(())
}

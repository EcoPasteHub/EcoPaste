use anyhow::Context;
use tauri::AppHandle;
use tauri_plugin_opener::OpenerExt;

use crate::core::{AppError, Result};

/// 打开经过白名单校验的外部网页 URL。
#[tauri::command]
pub async fn open_external_url(app: AppHandle, url: String) -> Result<()> {
    let normalized = url.trim();
    if !(normalized.starts_with("https://") || normalized.starts_with("http://")) {
        let lang = crate::i18n::current_language(&app);
        let message =
            crate::i18n::commands::label(lang, crate::i18n::commands::Key::ExternalUrlUnsupported);

        return Err(AppError::Other(anyhow::anyhow!(message)));
    }

    app.opener()
        .open_url(normalized, None::<&str>)
        .context("failed to open external url")?;

    Ok(())
}

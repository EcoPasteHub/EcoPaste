//! Local, serial background indexing; the original clipboard rows are never rewritten.

pub mod native;

use std::sync::atomic::{AtomicU64, Ordering};
use std::time::Duration;

use serde::Serialize;
use tauri::{AppHandle, Emitter, Manager};
use tokio::sync::Mutex;

use crate::clipboard::{ImageStore, WatcherPause};
use crate::core::Result;
use crate::db::{ocr as repository, DatabaseState};
use crate::settings::SettingsStore;

const CLIPBOARD_UPDATED_EVENT: &str = "clipboard://updated";

/// Serializes settings/clear with publication, invalidating in-flight native work on edits.
#[derive(Default)]
pub struct OcrRuntime {
    pub gate: Mutex<()>,
    generation: AtomicU64,
}

impl OcrRuntime {
    pub fn invalidate(&self) {
        self.generation.fetch_add(1, Ordering::SeqCst);
    }

    fn generation(&self) -> u64 {
        self.generation.load(Ordering::SeqCst)
    }
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ImageOcrStatus {
    supported: bool,
    enabled: bool,
    paused: bool,
    #[serde(flatten)]
    counts: repository::Counts,
}

pub async fn status(app: &AppHandle) -> Result<ImageOcrStatus> {
    let settings = app.state::<SettingsStore>().snapshot().clipboard.ocr;
    let pool = app.state::<DatabaseState>().pool().await;
    Ok(ImageOcrStatus {
        supported: native::supported(),
        enabled: settings.enabled,
        paused: settings.paused,
        counts: repository::counts(&pool).await?,
    })
}

pub async fn queue_history(app: &AppHandle) -> Result<ImageOcrStatus> {
    let runtime = app.state::<OcrRuntime>();
    let _guard = runtime.gate.lock().await;
    if !native::supported()
        || !app
            .state::<SettingsStore>()
            .snapshot()
            .clipboard
            .ocr
            .enabled
    {
        return Err(anyhow::anyhow!("Image OCR is not enabled or supported").into());
    }
    let pool = app.state::<DatabaseState>().pool().await;
    repository::enqueue_history(&pool).await?;
    status(app).await
}

/// Pause before clearing so results from a running native call cannot recreate the index.
pub async fn clear(app: &AppHandle) -> Result<ImageOcrStatus> {
    let runtime = app.state::<OcrRuntime>();
    let _guard = runtime.gate.lock().await;
    runtime.invalidate();
    let settings = app
        .state::<SettingsStore>()
        .update(serde_json::json!({"clipboard":{"ocr":{"paused":true}}}))?;
    crate::commands::emit_settings_updated(app, &settings);
    let pool = app.state::<DatabaseState>().pool().await;
    repository::clear(&pool).await?;
    notify_search(app, true);
    status(app).await
}

/// Queue metadata only on the capture path; all recognition runs off-thread later.
pub async fn on_capture(app: &AppHandle, pool: &sqlx::SqlitePool, id: &str) -> Result<()> {
    let runtime = app.state::<OcrRuntime>();
    let _guard = runtime.gate.lock().await;
    if native::supported()
        && app
            .state::<SettingsStore>()
            .snapshot()
            .clipboard
            .ocr
            .enabled
    {
        repository::enqueue(pool, id).await?;
    }
    Ok(())
}

pub fn notify_search(app: &AppHandle, cleared: bool) {
    // No id intentionally: consumers must refetch filters/counts, not append an unfiltered item.
    if let Err(err) = app.emit(
        CLIPBOARD_UPDATED_EVENT,
        serde_json::json!({"ocr": if cleared { "cleared" } else { "updated" }}),
    ) {
        log::warn!("emit OCR search refresh failed: {err}");
    }
}

/// Prepare provider resources before clipboard capture can enqueue new images.
pub fn configure(app: &AppHandle) {
    match app.path().resource_dir() {
        Ok(root) => {
            if let Err(err) = native::configure(&root.join("assets/ocr/tessdata")) {
                log::warn!("local image OCR components unavailable: {err}");
            }
        }
        Err(err) => log::warn!("image OCR resource directory unavailable: {err}"),
    }
}

pub fn spawn(app: AppHandle) {
    if !native::supported() {
        return;
    }
    tauri::async_runtime::spawn(async move {
        loop {
            if let Err(err) = step(&app).await {
                log::warn!("image OCR queue step failed: {err}");
            }
            tokio::time::sleep(Duration::from_secs(1)).await;
        }
    });
}

/// Checks current policy both before native work and under the publication lock after it.
async fn step(app: &AppHandle) -> Result<()> {
    let runtime = app.state::<OcrRuntime>();
    let (pool, job, path, generation) = {
        let _guard = runtime.gate.lock().await;
        let settings = app.state::<SettingsStore>().snapshot().clipboard.ocr;
        if !settings.enabled || settings.paused || app.state::<WatcherPause>().is_paused() {
            return Ok(());
        }
        let pool = app.state::<DatabaseState>().pool().await;
        let Some(job) = repository::next_job(&pool).await? else {
            return Ok(());
        };
        if !valid_image_name(&job.content) {
            repository::finish(&pool, &job, None).await?;
            return Ok(());
        }
        let path = app.state::<ImageStore>().origin_path(&job.content);
        if app.state::<WatcherPause>().is_paused() {
            return Ok(());
        }
        (pool, job, path, runtime.generation())
    };

    let source_path = path.clone();
    let recognized = tauri::async_runtime::spawn_blocking(move || native::recognize(&path)).await;
    let _guard = runtime.gate.lock().await;
    let settings = app.state::<SettingsStore>().snapshot().clipboard.ocr;
    if !can_publish(
        settings.enabled,
        settings.paused,
        generation,
        runtime.generation(),
    ) || pool.is_closed()
        || app.state::<WatcherPause>().is_paused()
        || source_path != app.state::<ImageStore>().origin_path(&job.content)
    {
        return Ok(());
    }
    // Never log OCR contents or paths. Failed jobs remain visible in counters for explicit retry.
    let text = match recognized {
        Ok(Ok(text)) => Some(text),
        _ => None,
    };
    if repository::finish(&pool, &job, text.as_deref()).await? && text.is_some() {
        notify_search(app, false);
    }
    Ok(())
}

fn can_publish(enabled: bool, paused: bool, started: u64, current: u64) -> bool {
    enabled && !paused && started == current
}

fn valid_image_name(name: &str) -> bool {
    name.len() == 68
        && name.ends_with(".png")
        && name.as_bytes()[..64].iter().all(|b| b.is_ascii_hexdigit())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn changing_policy_discards_running_results() {
        assert!(can_publish(true, false, 1, 1));
        assert!(!can_publish(false, false, 1, 1));
        assert!(!can_publish(true, true, 1, 1));
        assert!(!can_publish(true, false, 1, 2));
    }

    #[test]
    fn image_names_cannot_escape_resource_store() {
        assert!(valid_image_name(&format!("{}.png", "a".repeat(64))));
        assert!(!valid_image_name("../../secret.png"));
        assert!(!valid_image_name(&format!("{}.png", "好".repeat(22))));
    }

    #[test]
    fn old_settings_default_to_ocr_off() {
        let settings: crate::settings::Settings = serde_json::from_str("{}").unwrap();
        assert!(!settings.clipboard.ocr.enabled);
        assert!(!settings.clipboard.ocr.paused);
    }
}

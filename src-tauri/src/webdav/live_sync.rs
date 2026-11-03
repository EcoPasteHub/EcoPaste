//! 双向无感剪贴板文本同步（电脑 <-> 手机）。
//! 在 WebDAV 远端维护极简的 `live_text.json`，实现手机复制电脑粘贴、电脑复制手机粘贴。

use std::sync::atomic::{AtomicBool, AtomicI64, Ordering};
use std::sync::{Arc, Mutex};
use std::time::Duration;

use anyhow::Context;
use chrono::Utc;
use clipboard_rs::{Clipboard, ClipboardContext};
use serde::{Deserialize, Serialize};
use tauri::{AppHandle, Emitter, Listener, Manager};
use tokio::sync::Notify;

use crate::clipboard::WritebackGuard;
use crate::core::Result;
use crate::db::items::content_hash;
use crate::db::models::{ClipboardKind, ClipboardSubKind, Platform};
use crate::settings::SettingsStore;
use crate::webdav::client::WebDavClient;
use crate::webdav::sync::normalize_dir;

pub const LIVE_TEXT_FILENAME: &str = "live_text.json";

/// 实时剪贴板数据传输载荷。
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LiveClipboardPayload {
    pub text: String,
    pub device: String,
    pub timestamp: i64,
}

/// 启动双向实时无感剪贴板同步服务。
pub fn spawn_live_sync(app: AppHandle) {
    let last_seen_remote_ts = Arc::new(AtomicI64::new(0));
    let last_pushed_hash = Arc::new(Mutex::new(String::new()));

    let push_notify = Arc::new(Notify::new());
    let push_pending = Arc::new(AtomicBool::new(false));

    // 1. 监听本地剪贴板更新事件，当电脑有新复制时，快速推送到 WebDAV live_text.json
    {
        let notify = push_notify.clone();
        let pending = push_pending.clone();
        let app_handle = app.clone();
        app.listen("clipboard://updated", move |_| {
            if is_live_sync_enabled(&app_handle) {
                pending.store(true, Ordering::SeqCst);
                notify.notify_one();
            }
        });
    }

    // 电脑推送后台防抖任务（500ms 快速响应）
    {
        let app = app.clone();
        let notify = push_notify.clone();
        let pending = push_pending.clone();
        let pushed_hash = last_pushed_hash.clone();
        tauri::async_runtime::spawn(async move {
            loop {
                notify.notified().await;
                tokio::time::sleep(Duration::from_millis(500)).await;
                if !pending.swap(false, Ordering::SeqCst) {
                    continue;
                }

                if !is_live_sync_enabled(&app) {
                    continue;
                }

                if let Err(err) = push_desktop_live_text(&app, &pushed_hash).await {
                    log::debug!("webdav live text push skipped or failed: {err}");
                }
            }
        });
    }

    // 2. 远端轮询拉取任务：定期检查 WebDAV 上是否有来自手机（HarmonyOS）的新文本
    {
        let app = app.clone();
        let last_seen_ts = last_seen_remote_ts.clone();
        let pushed_hash = last_pushed_hash.clone();
        tauri::async_runtime::spawn(async move {
            let mut interval = tokio::time::interval(Duration::from_millis(2500));
            interval.tick().await;

            loop {
                interval.tick().await;

                if !is_live_sync_enabled(&app) {
                    continue;
                }

                if let Err(err) = pull_remote_live_text(&app, &last_seen_ts, &pushed_hash).await {
                    log::trace!("webdav live text pull check: {err}");
                }
            }
        });
    }

    log::info!("webdav: live clipboard bidirectional synchronization spawned");
}

/// 检查 WebDAV 配置是否开启且具备连接参数。
fn is_live_sync_enabled(app: &AppHandle) -> bool {
    let settings = match app.try_state::<SettingsStore>() {
        Some(s) => s.snapshot(),
        None => return false,
    };
    let webdav = &settings.webdav;
    webdav.enabled && !webdav.endpoint.trim().is_empty() && webdav.sync_text
}

/// 电脑端将最新复制的纯文本快速上传至 `live_text.json`。
async fn push_desktop_live_text(
    app: &AppHandle,
    last_pushed_hash: &Arc<Mutex<String>>,
) -> Result<()> {
    let settings = match app.try_state::<SettingsStore>() {
        Some(s) => s.snapshot(),
        None => return Ok(()),
    };
    let webdav = &settings.webdav;
    let pool = app.state::<crate::db::DatabaseState>().pool().await;

    // 获取本地最新的一条文本类型条目
    let row: Option<(String, String)> = sqlx::query_as(
        "SELECT content, content_hash FROM clipboard_items \
         WHERE kind = 'text' ORDER BY created_at DESC LIMIT 1",
    )
    .fetch_optional(&pool)
    .await
    .unwrap_or(None);

    let (content, hash) = match row {
        Some(r) => r,
        None => return Ok(()),
    };

    if content.trim().is_empty() {
        return Ok(());
    }

    // 若内容未变动，则跳过上传
    {
        let mut last_hash = last_pushed_hash.lock().unwrap_or_else(|e| e.into_inner());
        if *last_hash == hash {
            return Ok(());
        }
        *last_hash = hash;
    }

    let client = WebDavClient::new(&webdav.endpoint, &webdav.username, &webdav.password)?;
    let remote_dir = normalize_dir(&webdav.remote_dir);
    let remote_file_path = format!("{remote_dir}/{LIVE_TEXT_FILENAME}");

    let payload = LiveClipboardPayload {
        text: content,
        device: "desktop".to_owned(),
        timestamp: Utc::now().timestamp_millis(),
    };

    let json_bytes =
        serde_json::to_vec(&payload).context("failed to serialize live clipboard payload")?;
    client
        .put(&remote_file_path, json_bytes, "application/json")
        .await?;
    log::info!("webdav: live clipboard text pushed from desktop to WebDAV");

    Ok(())
}

/// 从 WebDAV 拉取远端文本（例如来自 HarmonyOS 鸿蒙手机），并写入电脑剪贴板。
async fn pull_remote_live_text(
    app: &AppHandle,
    last_seen_ts: &Arc<AtomicI64>,
    last_pushed_hash: &Arc<Mutex<String>>,
) -> Result<()> {
    let settings = match app.try_state::<SettingsStore>() {
        Some(s) => s.snapshot(),
        None => return Ok(()),
    };
    let webdav = &settings.webdav;

    let client = WebDavClient::new(&webdav.endpoint, &webdav.username, &webdav.password)?;
    let remote_dir = normalize_dir(&webdav.remote_dir);
    let remote_file_path = format!("{remote_dir}/{LIVE_TEXT_FILENAME}");

    let bytes = match client.get(&remote_file_path).await? {
        Some(b) => b,
        None => return Ok(()),
    };

    let payload: LiveClipboardPayload = match serde_json::from_slice(&bytes) {
        Ok(p) => p,
        Err(_) => return Ok(()),
    };

    // 仅处理来自非桌面设备（如 harmony 手机端）且时间戳较新的条目
    if payload.device == "desktop" || payload.text.trim().is_empty() {
        return Ok(());
    }

    let prev_ts = last_seen_ts.load(Ordering::Relaxed);
    if payload.timestamp <= prev_ts {
        return Ok(());
    }
    last_seen_ts.store(payload.timestamp, Ordering::Relaxed);

    let text_hash = content_hash(ClipboardKind::Text, &payload.text);
    {
        let mut last_hash = last_pushed_hash.lock().unwrap_or_else(|e| e.into_inner());
        *last_hash = text_hash.clone();
    }

    // 1. 抑制回环并写入电脑剪贴板
    if let Some(guard) = app.try_state::<Arc<WritebackGuard>>() {
        guard.suppress(text_hash.clone());
    }

    if let Ok(ctx) = ClipboardContext::new() {
        if let Err(err) = ctx.set_text(payload.text.clone()) {
            log::warn!("failed to set clipboard text from webdav: {err}");
        }
    }

    // 2. 将来自手机的条目插入本地历史数据库并通知前端界面刷新
    let pool = app.state::<crate::db::DatabaseState>().pool().await;
    let now = Utc::now();
    let now_str = now.to_rfc3339();
    let id = uuid::Uuid::new_v4().to_string();
    let summary: String = payload.text.chars().take(256).collect();

    let _ = sqlx::query(
        "INSERT OR IGNORE INTO clipboard_items \
         (id, kind, sub_kind, group_id, source_app_id, content, content_hash, search_text, \
          summary, file_types, size, width, height, use_count, is_favorite, is_pinned, is_sensitive, platform, note, \
          created_at, updated_at) \
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
    )
    .bind(&id)
    .bind(ClipboardKind::Text)
    .bind(None::<ClipboardSubKind>)
    .bind(None::<String>)
    .bind(None::<String>)
    .bind(&payload.text)
    .bind(&text_hash)
    .bind(&payload.text)
    .bind(&summary)
    .bind(None::<String>)
    .bind(payload.text.len() as i64)
    .bind(None::<i64>)
    .bind(None::<i64>)
    .bind(1)
    .bind(false)
    .bind(false)
    .bind(false)
    .bind(Platform::Windows)
    .bind(Some("来自 HarmonyOS 鸿蒙手机".to_owned()))
    .bind(&now_str)
    .bind(&now_str)
    .execute(&pool)
    .await;

    let _ = app.emit("clipboard://updated", serde_json::json!({ "live": true }));
    log::info!(
        "webdav: live clipboard text from HarmonyOS applied to desktop (len: {})",
        payload.text.len()
    );

    Ok(())
}

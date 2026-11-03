//! WebDAV 备份与增量同步核心实现。

use std::fs;
use std::path::PathBuf;

use anyhow::{anyhow, Context};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::SqlitePool;
use tauri::{AppHandle, Emitter, Manager};

use super::client::WebDavClient;
use super::model::{
    WebDavManifest, WebDavRestoreResult, WebDavSyncResult, WebDavSyncState, WebDavTestResult,
};
use crate::clipboard::ImageStore;
use crate::core::{AppError, Result};
use crate::db::models::{ClipboardApp, ClipboardGroup, ClipboardItem, ClipboardKind};
use crate::settings::{Settings, SettingsStore};

pub const WEBDAV_STATUS_UPDATED_EVENT: &str = "webdav://status_updated";
const SYNC_STATE_FILENAME: &str = "webdav_sync_state.json";
const MANIFEST_FILENAME: &str = "manifest.json";
const SETTINGS_FILENAME: &str = "settings.json";

/// WebDAV 上传的增量批次文件结构。
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct WebDavBatch {
    pub batch_id: String,
    pub created_at: DateTime<Utc>,
    pub items: Vec<ClipboardItem>,
    pub groups: Vec<ClipboardGroup>,
    pub apps: Vec<ClipboardApp>,
}

/// 测试 WebDAV 连接。
pub async fn test_connection(app: &AppHandle) -> Result<WebDavTestResult> {
    let settings = app
        .try_state::<SettingsStore>()
        .map(|s| s.snapshot())
        .unwrap_or_default();

    let webdav = &settings.webdav;
    if webdav.endpoint.trim().is_empty() {
        return Err(AppError::Other(anyhow!("请先填写 WebDAV 服务器地址")));
    }

    let client = WebDavClient::new(&webdav.endpoint, &webdav.username, &webdav.password)?;
    client.test_connection().await?;

    Ok(WebDavTestResult {
        success: true,
        message: "WebDAV 连接成功".to_owned(),
    })
}

/// 立即执行一次 WebDAV 同步与备份。
pub async fn sync_now(app: &AppHandle) -> Result<WebDavSyncResult> {
    let settings_store = app
        .try_state::<SettingsStore>()
        .ok_or_else(|| AppError::Other(anyhow!("设置存储不可用")))?;
    let settings = settings_store.snapshot();
    let webdav = &settings.webdav;

    if webdav.endpoint.trim().is_empty() {
        return Err(AppError::Other(anyhow!("请先配置 WebDAV 服务器地址")));
    }

    let result = execute_sync(app, &settings).await;

    // 更新同步状态并保存到设置与持久化
    let now_str = Utc::now().to_rfc3339();
    match &result {
        Ok(res) => {
            let patch = serde_json::json!({
                "webdav": {
                    "lastSyncAt": now_str,
                    "lastSyncStatus": "success",
                    "lastSyncError": null,
                }
            });
            if let Ok(next) = settings_store.update(patch) {
                let _ = app.emit("settings://updated", &next);
                let _ = app.emit(WEBDAV_STATUS_UPDATED_EVENT, &next.webdav);
            }
            Ok(res.clone())
        }
        Err(err) => {
            let err_msg = err.to_string();
            let patch = serde_json::json!({
                "webdav": {
                    "lastSyncAt": now_str,
                    "lastSyncStatus": "failed",
                    "lastSyncError": err_msg.clone(),
                }
            });
            if let Ok(next) = settings_store.update(patch) {
                let _ = app.emit("settings://updated", &next);
                let _ = app.emit(WEBDAV_STATUS_UPDATED_EVENT, &next.webdav);
            }
            Err(AppError::Other(anyhow!("WebDAV 同步失败: {err_msg}")))
        }
    }
}

/// 执行同步内部逻辑。
async fn execute_sync(app: &AppHandle, settings: &Settings) -> Result<WebDavSyncResult> {
    let webdav = &settings.webdav;
    let client = WebDavClient::new(&webdav.endpoint, &webdav.username, &webdav.password)?;

    let remote_dir = normalize_dir(&webdav.remote_dir);
    client.ensure_dir(&remote_dir).await?;
    let items_dir = format!("{remote_dir}/items");
    client.ensure_dir(&items_dir).await?;
    let images_dir = format!("{remote_dir}/resources/clipboard-images");
    client.ensure_dir(&images_dir).await?;

    let mut state = load_sync_state(app)?;
    let now = Utc::now();
    let now_str = now.to_rfc3339();

    let mut backed_up_settings = false;
    if webdav.backup_settings {
        let config_dir = crate::core::paths::config_dir(app)?;
        let settings_file = config_dir.join(SETTINGS_FILENAME);
        if settings_file.exists() {
            // 安全起见，上传备份配置时将密码打码
            let mut safe_settings = settings.clone();
            safe_settings.webdav.password = String::new();
            let json_bytes = serde_json::to_vec_pretty(&safe_settings)
                .context("failed to serialize settings for backup")?;
            let remote_settings_path = format!("{remote_dir}/{SETTINGS_FILENAME}");
            client
                .put(&remote_settings_path, json_bytes, "application/json")
                .await?;
            backed_up_settings = true;
        }
    }

    let mut uploaded_items_count = 0;
    let mut uploaded_images_count = 0;

    if webdav.backup_clipboard {
        let pool = app.state::<crate::db::DatabaseState>().pool().await;
        let image_store = app.state::<ImageStore>();

        // 增量获取条目
        let items = if webdav.incremental {
            if let Some(last_sync_time) = &state.last_synced_at {
                fetch_items_since(&pool, last_sync_time).await?
            } else {
                fetch_all_items(&pool).await?
            }
        } else {
            fetch_all_items(&pool).await?
        };

        let groups = fetch_all_groups(&pool).await?;
        let apps = fetch_all_apps(&pool).await?;

        // 关联图片增量上传
        for item in &items {
            if item.kind == ClipboardKind::Image {
                let file_name = &item.content;
                if !state.synced_image_hashes.contains(file_name) {
                    let local_image_path = image_store.origin_path(file_name);
                    if local_image_path.exists() {
                        if let Ok(bytes) = fs::read(&local_image_path) {
                            let remote_image_path = format!("{images_dir}/{file_name}");
                            if client
                                .put(&remote_image_path, bytes, "image/png")
                                .await
                                .is_ok()
                            {
                                state.synced_image_hashes.push(file_name.clone());
                                uploaded_images_count += 1;
                            }
                        }
                    }
                }
            }
        }

        // 若有新条目需要备份，生成增量批次文件
        if !items.is_empty() || state.last_synced_at.is_none() {
            let batch_id = format!("batch_{}", now.format("%Y%m%d_%H%M%S"));
            let batch_filename = format!("{batch_id}.json");
            let batch = WebDavBatch {
                batch_id: batch_id.clone(),
                created_at: now,
                items,
                groups,
                apps,
            };

            uploaded_items_count = batch.items.len();

            let batch_bytes =
                serde_json::to_vec(&batch).context("failed to serialize webdav batch")?;
            let remote_batch_path = format!("{items_dir}/{batch_filename}");
            client
                .put(&remote_batch_path, batch_bytes, "application/json")
                .await?;

            // 获取或创建远程清单文件
            let remote_manifest_path = format!("{remote_dir}/{MANIFEST_FILENAME}");
            let mut manifest: WebDavManifest = match client.get(&remote_manifest_path).await? {
                Some(bytes) => serde_json::from_slice(&bytes).unwrap_or_else(|_| WebDavManifest {
                    version: 1,
                    app_name: "EcoPaste".to_owned(),
                    app_version: app.package_info().version.to_string(),
                    last_synced_at: now_str.clone(),
                    total_items: 0,
                    batches: Vec::new(),
                }),
                None => WebDavManifest {
                    version: 1,
                    app_name: "EcoPaste".to_owned(),
                    app_version: app.package_info().version.to_string(),
                    last_synced_at: now_str.clone(),
                    total_items: 0,
                    batches: Vec::new(),
                },
            };

            if !manifest.batches.contains(&batch_filename) {
                manifest.batches.push(batch_filename);
            }
            manifest.last_synced_at = now_str.clone();
            manifest.total_items += uploaded_items_count;

            let manifest_bytes =
                serde_json::to_vec_pretty(&manifest).context("failed to serialize manifest")?;
            client
                .put(&remote_manifest_path, manifest_bytes, "application/json")
                .await?;
        }
    }

    state.last_synced_at = Some(now_str.clone());
    save_sync_state(app, &state)?;

    Ok(WebDavSyncResult {
        success: true,
        uploaded_items: uploaded_items_count,
        uploaded_images: uploaded_images_count,
        backed_up_settings,
        error: None,
        synced_at: now_str,
    })
}

/// 从 WebDAV 恢复备份数据。
pub async fn restore_from_webdav(app: &AppHandle) -> Result<WebDavRestoreResult> {
    let settings_store = app
        .try_state::<SettingsStore>()
        .ok_or_else(|| AppError::Other(anyhow!("设置存储不可用")))?;
    let settings = settings_store.snapshot();
    let webdav = &settings.webdav;

    if webdav.endpoint.trim().is_empty() {
        return Err(AppError::Other(anyhow!("请先配置 WebDAV 服务器地址")));
    }

    let client = WebDavClient::new(&webdav.endpoint, &webdav.username, &webdav.password)?;
    let remote_dir = normalize_dir(&webdav.remote_dir);

    let remote_manifest_path = format!("{remote_dir}/{MANIFEST_FILENAME}");
    let manifest_bytes = client
        .get(&remote_manifest_path)
        .await?
        .ok_or_else(|| AppError::Other(anyhow!("未在 WebDAV 服务器找到备份数据")))?;

    let manifest: WebDavManifest =
        serde_json::from_slice(&manifest_bytes).context("解析远端备份清单失败")?;

    let pool = app.state::<crate::db::DatabaseState>().pool().await;
    let image_store = app.state::<ImageStore>();

    let mut restored_items_count = 0;
    let mut restored_images_count = 0;

    let items_dir = format!("{remote_dir}/items");
    let images_dir = format!("{remote_dir}/resources/clipboard-images");

    for batch_file in manifest.batches {
        let remote_batch_path = format!("{items_dir}/{batch_file}");
        if let Some(bytes) = client.get(&remote_batch_path).await? {
            if let Ok(batch) = serde_json::from_slice::<WebDavBatch>(&bytes) {
                // 恢复条目和图片
                for item in batch.items {
                    if item.kind == ClipboardKind::Image {
                        let file_name = &item.content;
                        let local_image_path = image_store.origin_path(file_name);
                        if !local_image_path.exists() {
                            let remote_image_path = format!("{images_dir}/{file_name}");
                            if let Ok(Some(img_bytes)) = client.get(&remote_image_path).await {
                                if let Some(parent) = local_image_path.parent() {
                                    let _ = fs::create_dir_all(parent);
                                }
                                if fs::write(&local_image_path, img_bytes).is_ok() {
                                    restored_images_count += 1;
                                }
                            }
                        }
                    }

                    // 入库去重判定
                    let exists: Option<i64> = sqlx::query_scalar(
                        "SELECT 1 FROM clipboard_items WHERE kind = ? AND content_hash = ? LIMIT 1",
                    )
                    .bind(item.kind)
                    .bind(&item.content_hash)
                    .fetch_optional(&pool)
                    .await
                    .unwrap_or(None);

                    if exists.is_none() {
                        let _ = sqlx::query(
                            "INSERT OR IGNORE INTO clipboard_items \
                             (id, kind, sub_kind, group_id, source_app_id, content, content_hash, search_text, \
                              summary, file_types, size, width, height, use_count, is_favorite, is_pinned, is_sensitive, platform, note, \
                              created_at, updated_at) \
                             VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
                        )
                        .bind(&item.id)
                        .bind(item.kind)
                        .bind(item.sub_kind)
                        .bind(&item.group_id)
                        .bind(&item.source_app_id)
                        .bind(&item.content)
                        .bind(&item.content_hash)
                        .bind(&item.search_text)
                        .bind(&item.summary)
                        .bind(&item.file_types)
                        .bind(item.size)
                        .bind(item.width)
                        .bind(item.height)
                        .bind(item.use_count)
                        .bind(item.is_favorite)
                        .bind(item.is_pinned)
                        .bind(item.is_sensitive)
                        .bind(item.platform)
                        .bind(&item.note)
                        .bind(item.created_at)
                        .bind(item.updated_at)
                        .execute(&pool)
                        .await;

                        restored_items_count += 1;
                    }
                }

                // 恢复分组
                for group in batch.groups {
                    let _ = sqlx::query(
                        "INSERT OR IGNORE INTO clipboard_groups \
                         (id, name, icon, is_hidden, sort_order, created_at, updated_at) \
                         VALUES (?, ?, ?, ?, ?, ?, ?)",
                    )
                    .bind(group.id)
                    .bind(group.name)
                    .bind(group.icon)
                    .bind(group.is_hidden)
                    .bind(group.sort_order)
                    .bind(group.created_at)
                    .bind(group.updated_at)
                    .execute(&pool)
                    .await;
                }

                // 恢复应用
                for app_item in batch.apps {
                    let _ = sqlx::query(
                        "INSERT OR IGNORE INTO clipboard_apps \
                         (id, name, icon_file, platform, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
                    )
                    .bind(app_item.id)
                    .bind(app_item.name)
                    .bind(app_item.icon_file)
                    .bind(app_item.platform)
                    .bind(app_item.created_at)
                    .bind(app_item.updated_at)
                    .execute(&pool)
                    .await;
                }
            }
        }
    }

    // 检查并恢复设置
    let mut restored_settings = false;
    let remote_settings_path = format!("{remote_dir}/{SETTINGS_FILENAME}");
    if let Ok(Some(bytes)) = client.get(&remote_settings_path).await {
        if let Ok(mut remote_settings_json) = serde_json::from_slice::<serde_json::Value>(&bytes) {
            // 保留当前客户端的 WebDAV 配置，避免被远端覆盖
            if let Some(obj) = remote_settings_json.as_object_mut() {
                obj.remove("webdav");
            }
            if let Ok(next) = settings_store.update(remote_settings_json) {
                let _ = app.emit("settings://updated", &next);
                restored_settings = true;
            }
        }
    }

    let _ = app.emit(
        "clipboard://updated",
        serde_json::json!({ "restored": true }),
    );

    Ok(WebDavRestoreResult {
        success: true,
        restored_items: restored_items_count,
        restored_images: restored_images_count,
        restored_settings,
        error: None,
    })
}

fn normalize_dir(dir: &str) -> String {
    let trimmed = dir.trim().trim_matches('/');
    if trimmed.is_empty() {
        "EcoPaste".to_owned()
    } else {
        trimmed.to_owned()
    }
}

fn sync_state_path(app: &AppHandle) -> Result<PathBuf> {
    let dir = crate::core::paths::config_dir(app)?;
    Ok(dir.join(SYNC_STATE_FILENAME))
}

fn load_sync_state(app: &AppHandle) -> Result<WebDavSyncState> {
    let path = sync_state_path(app)?;
    if !path.exists() {
        return Ok(WebDavSyncState::default());
    }

    let content = fs::read_to_string(&path)
        .with_context(|| format!("failed to read sync state at {path:?}"))?;
    let state = serde_json::from_str(&content).unwrap_or_default();
    Ok(state)
}

fn save_sync_state(app: &AppHandle, state: &WebDavSyncState) -> Result<()> {
    let path = sync_state_path(app)?;
    let content =
        serde_json::to_string_pretty(state).context("failed to serialize webdav sync state")?;
    fs::write(&path, content).with_context(|| format!("failed to write sync state to {path:?}"))?;
    Ok(())
}

async fn fetch_all_items(pool: &SqlitePool) -> Result<Vec<ClipboardItem>> {
    let rows = sqlx::query_as::<_, ClipboardItem>(
        "SELECT id, kind, sub_kind, group_id, source_app_id, content, \
         content_hash, search_text, summary, file_types, size, width, height, use_count, is_favorite, is_pinned, \
         is_sensitive, platform, note, created_at, updated_at FROM clipboard_items ORDER BY created_at ASC",
    )
    .fetch_all(pool)
    .await
    .context("failed to query all clipboard items")?;

    Ok(rows)
}

async fn fetch_items_since(pool: &SqlitePool, since: &str) -> Result<Vec<ClipboardItem>> {
    let rows = sqlx::query_as::<_, ClipboardItem>(
        "SELECT id, kind, sub_kind, group_id, source_app_id, content, \
         content_hash, search_text, summary, file_types, size, width, height, use_count, is_favorite, is_pinned, \
         is_sensitive, platform, note, created_at, updated_at FROM clipboard_items \
         WHERE created_at > ? OR updated_at > ? ORDER BY created_at ASC",
    )
    .bind(since)
    .bind(since)
    .fetch_all(pool)
    .await
    .context("failed to query incremental clipboard items")?;

    Ok(rows)
}

async fn fetch_all_groups(pool: &SqlitePool) -> Result<Vec<ClipboardGroup>> {
    let rows = sqlx::query_as::<_, ClipboardGroup>(
        "SELECT id, name, icon, is_hidden, sort_order, created_at, updated_at FROM clipboard_groups",
    )
    .fetch_all(pool)
    .await
    .context("failed to query clipboard groups")?;

    Ok(rows)
}

async fn fetch_all_apps(pool: &SqlitePool) -> Result<Vec<ClipboardApp>> {
    let rows = sqlx::query_as::<_, ClipboardApp>(
        "SELECT id, name, icon_file, platform, created_at, updated_at FROM clipboard_apps",
    )
    .fetch_all(pool)
    .await
    .context("failed to query clipboard apps")?;

    Ok(rows)
}

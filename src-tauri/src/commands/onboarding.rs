use std::fs;
use std::path::{Path, PathBuf};

use anyhow::Context;
use chrono::{DateTime, NaiveDateTime, Utc};
use serde::{Deserialize, Serialize};
use serde_json::json;
use sqlx::sqlite::{SqliteConnectOptions, SqliteJournalMode, SqlitePoolOptions};
use sqlx::SqlitePool;
use tauri::{AppHandle, Emitter, Manager};
use uuid::Uuid;
use walkdir::WalkDir;

use crate::clipboard::{
    build_item_with_settings, persist_and_notify, ClipboardPayload, ImagePayload, ImageStore,
    TextPayload,
};
use crate::core::{AppError, Result};
use crate::db::DatabaseState;
use crate::settings::{Settings, SettingsStore};
use crate::window;

const MAX_ONBOARDING_STEP: u32 = 20;
const CLIPBOARD_UPDATED_EVENT: &str = "clipboard://updated";
const LEGACY_HISTORY_TABLE: &str = "history";

#[derive(Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct OnboardingLegacyDataDetection {
    found: bool,
    path: Option<String>,
    database_files: Vec<String>,
    total_bytes: u64,
    checked_at: String,
    importable_database: Option<String>,
    importable_item_count: u64,
    normal_item_count: u64,
    favorite_item_count: u64,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "camelCase")]
pub enum LegacyImportSelection {
    Normal,
    Favorite,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct OnboardingLegacyImportResult {
    imported: u64,
    skipped: u64,
    imported_normal: u64,
    imported_favorite: u64,
    selected_types: Vec<LegacyImportSelection>,
    imported_at: String,
}

#[derive(Debug, Clone)]
struct LegacyDataMatch {
    path: PathBuf,
    database_files: Vec<PathBuf>,
    total_bytes: u64,
    importable_database: Option<PathBuf>,
    importable_item_count: u64,
    normal_item_count: u64,
    favorite_item_count: u64,
}

#[derive(Debug, Clone, sqlx::FromRow)]
struct LegacyHistoryRow {
    #[sqlx(rename = "type")]
    item_type: Option<String>,
    value: Option<String>,
    favorite: Option<i64>,
    #[sqlx(rename = "createTime")]
    create_time: Option<String>,
    note: Option<String>,
    width: Option<i64>,
    height: Option<i64>,
}

#[tauri::command]
pub async fn open_onboarding(app: AppHandle) -> Result<()> {
    window::open_onboarding(&app)
}

#[tauri::command]
pub async fn set_onboarding_step(app: AppHandle, step: u32) -> Result<Settings> {
    update_onboarding_settings(
        &app,
        json!({
            "onboarding": {
                "lastStep": step.min(MAX_ONBOARDING_STEP),
            },
        }),
    )
}

#[tauri::command]
pub async fn finish_onboarding(app: AppHandle) -> Result<Settings> {
    let next = update_onboarding_settings(
        &app,
        json!({
            "onboarding": {
                "completed": true,
                "lastStep": 0,
            },
        }),
    )?;

    if let Err(err) = window::hide_window(&app, window::ONBOARDING_WINDOW_LABEL) {
        log::warn!("hide onboarding window after finish failed: {err}");
    }

    window::show_window(&app, window::MAIN_WINDOW_LABEL)?;

    Ok(next)
}

#[tauri::command]
pub async fn detect_legacy_data(app: AppHandle) -> Result<OnboardingLegacyDataDetection> {
    let detection = inspect_legacy_data().await?;

    update_onboarding_settings(
        &app,
        json!({
            "onboarding": {
                "legacyImport": {
                    "checked": true,
                },
            },
        }),
    )?;

    Ok(detection)
}

#[tauri::command]
pub async fn import_legacy_data(
    app: AppHandle,
    types: Vec<LegacyImportSelection>,
) -> Result<OnboardingLegacyImportResult> {
    if types.is_empty() {
        return Err(AppError::Clipboard("请至少选择一种导入类型".to_owned()));
    }

    let detection = inspect_legacy_data().await?;
    let db_path = detection
        .importable_database
        .clone()
        .ok_or_else(|| AppError::Clipboard("未找到可导入的旧版数据库".to_owned()))?;
    let legacy_db_path = PathBuf::from(&db_path);
    let legacy_pool = open_legacy_pool(&legacy_db_path).await?;
    let db = app.state::<DatabaseState>();
    let pool = db.pool().await;
    let image_store = app.state::<ImageStore>();
    let settings = app.state::<SettingsStore>().snapshot();
    let selected_types = normalize_import_types(types);

    let rows = sqlx::query_as::<_, LegacyHistoryRow>(
        "SELECT type, value, favorite, createTime, note, width, height \
         FROM history ORDER BY createTime ASC, id ASC",
    )
    .fetch_all(&legacy_pool)
    .await
    .context("failed to load legacy history rows")?;

    let mut imported = 0_u64;
    let mut skipped = 0_u64;
    let mut imported_normal = 0_u64;
    let mut imported_favorite = 0_u64;
    let legacy_images_dir = detection
        .path
        .as_ref()
        .map(|path| PathBuf::from(path).join("images"));

    for row in rows {
        let is_favorite = row.favorite.unwrap_or_default() != 0;
        let selection = if is_favorite {
            LegacyImportSelection::Favorite
        } else {
            LegacyImportSelection::Normal
        };

        if !selected_types.contains(&selection) {
            continue;
        }

        let Some(mut item) =
            map_legacy_row_to_item(&row, legacy_images_dir.as_deref(), &image_store, &settings)?
        else {
            skipped += 1;
            continue;
        };

        item.is_favorite = is_favorite;

        let result = persist_and_notify(&app, &pool, &item, None).await?;
        if result.deduplicated {
            skipped += 1;
            continue;
        }

        imported += 1;
        if is_favorite {
            imported_favorite += 1;
        } else {
            imported_normal += 1;
        }
    }

    let imported_at = Utc::now().to_rfc3339();
    update_onboarding_settings(
        &app,
        json!({
            "onboarding": {
                "legacyImport": {
                    "checked": true,
                    "imported": imported > 0,
                    "importTypes": selected_types,
                    "importedAt": imported_at,
                },
            },
        }),
    )?;

    if let Err(err) = app.emit(
        CLIPBOARD_UPDATED_EVENT,
        json!({
            "imported": imported > 0,
        }),
    ) {
        log::warn!("emit {CLIPBOARD_UPDATED_EVENT} after legacy import failed: {err}");
    }

    Ok(OnboardingLegacyImportResult {
        imported,
        skipped,
        imported_normal,
        imported_favorite,
        selected_types,
        imported_at,
    })
}

fn update_onboarding_settings(app: &AppHandle, patch: serde_json::Value) -> Result<Settings> {
    let next = app.state::<SettingsStore>().update(patch)?;

    super::settings::emit_settings_updated(app, &next);

    Ok(next)
}

async fn inspect_legacy_data() -> Result<OnboardingLegacyDataDetection> {
    let mut best_match: Option<LegacyDataMatch> = None;

    for dir in legacy_data_candidates() {
        let Some(candidate) = inspect_legacy_dir(&dir).await? else {
            continue;
        };

        if best_match
            .as_ref()
            .is_none_or(|current| candidate.total_bytes > current.total_bytes)
        {
            best_match = Some(candidate);
        }
    }

    let checked_at = Utc::now().to_rfc3339();
    let Some(match_data) = best_match else {
        return Ok(OnboardingLegacyDataDetection {
            found: false,
            path: None,
            database_files: Vec::new(),
            total_bytes: 0,
            checked_at,
            importable_database: None,
            importable_item_count: 0,
            normal_item_count: 0,
            favorite_item_count: 0,
        });
    };

    Ok(OnboardingLegacyDataDetection {
        found: true,
        path: Some(match_data.path.display().to_string()),
        database_files: match_data
            .database_files
            .into_iter()
            .map(|path| path.display().to_string())
            .collect(),
        total_bytes: match_data.total_bytes,
        checked_at,
        importable_database: match_data
            .importable_database
            .map(|path| path.display().to_string()),
        importable_item_count: match_data.importable_item_count,
        normal_item_count: match_data.normal_item_count,
        favorite_item_count: match_data.favorite_item_count,
    })
}

async fn inspect_legacy_dir(dir: &Path) -> Result<Option<LegacyDataMatch>> {
    if !dir.exists() {
        return Ok(None);
    }

    let mut database_files = Vec::new();
    let mut total_bytes = 0;

    for entry in WalkDir::new(dir).max_depth(4).into_iter().flatten() {
        let path = entry.path();
        if !path.is_file() {
            continue;
        }

        if let Ok(metadata) = fs::metadata(path) {
            total_bytes += metadata.len();
        }

        if is_legacy_database_file(path) {
            database_files.push(path.to_path_buf());
        }
    }

    if database_files.is_empty() && total_bytes == 0 {
        return Ok(None);
    }

    let mut importable_database = None;
    let mut importable_item_count = 0;
    let mut normal_item_count = 0;
    let mut favorite_item_count = 0;

    for db_path in &database_files {
        let Some(stats) = inspect_legacy_database(db_path).await? else {
            continue;
        };

        if stats.item_count > importable_item_count {
            importable_database = Some(db_path.clone());
            importable_item_count = stats.item_count;
            normal_item_count = stats.normal_item_count;
            favorite_item_count = stats.favorite_item_count;
        }
    }

    Ok(Some(LegacyDataMatch {
        path: dir.to_path_buf(),
        database_files,
        total_bytes,
        importable_database,
        importable_item_count,
        normal_item_count,
        favorite_item_count,
    }))
}

struct LegacyDatabaseStats {
    item_count: u64,
    normal_item_count: u64,
    favorite_item_count: u64,
}

async fn inspect_legacy_database(path: &Path) -> Result<Option<LegacyDatabaseStats>> {
    let pool = match open_legacy_pool(path).await {
        Ok(pool) => pool,
        Err(err) => {
            log::warn!(
                "inspect legacy database failed for {}: {err}",
                path.display()
            );
            return Ok(None);
        }
    };

    let table_exists: Option<String> = sqlx::query_scalar(
        "SELECT name FROM sqlite_master WHERE type = 'table' AND name = ? LIMIT 1",
    )
    .bind(LEGACY_HISTORY_TABLE)
    .fetch_optional(&pool)
    .await
    .context("failed to inspect legacy schema")?;

    if table_exists.is_none() {
        return Ok(None);
    }

    let row = sqlx::query_as::<_, (i64, i64, i64)>(
        "SELECT \
            COUNT(*) AS total, \
            SUM(CASE WHEN favorite = 0 THEN 1 ELSE 0 END) AS normal_count, \
            SUM(CASE WHEN favorite = 1 THEN 1 ELSE 0 END) AS favorite_count \
         FROM history",
    )
    .fetch_one(&pool)
    .await
    .context("failed to count legacy history rows")?;

    Ok(Some(LegacyDatabaseStats {
        item_count: row.0.max(0) as u64,
        normal_item_count: row.1.max(0) as u64,
        favorite_item_count: row.2.max(0) as u64,
    }))
}

async fn open_legacy_pool(path: &Path) -> Result<SqlitePool> {
    let options = SqliteConnectOptions::new()
        .filename(path)
        .read_only(true)
        .journal_mode(SqliteJournalMode::Wal)
        .create_if_missing(false);

    SqlitePoolOptions::new()
        .max_connections(1)
        .connect_with(options)
        .await
        .with_context(|| format!("failed to open legacy database {}", path.display()))
        .map_err(AppError::from)
}

fn is_legacy_database_file(path: &Path) -> bool {
    let Some(file_name) = path.file_name().and_then(|value| value.to_str()) else {
        return false;
    };

    matches!(
        file_name,
        ".store.db"
            | ".store.dev.db"
            | ".window-state.db"
            | ".window-state.dev.db"
            | "EcoPaste.db"
            | "EcoPaste.dev.db"
            | "EcoPaste.v2.db"
            | "EcoPaste.v2.dev.db"
    ) || file_name.ends_with(".sqlite")
        || file_name.ends_with(".sqlite3")
        || file_name.ends_with(".db")
}

fn legacy_data_candidates() -> Vec<PathBuf> {
    let mut candidates = Vec::new();

    #[cfg(target_os = "macos")]
    {
        if let Some(home) = std::env::var_os("HOME") {
            let base = PathBuf::from(home).join("Library/Application Support");
            candidates.push(base.join("com.ayangweb.EcoPaste"));
            candidates.push(base.join("EcoPaste"));
        }
    }

    #[cfg(target_os = "windows")]
    {
        if let Some(app_data) = std::env::var_os("APPDATA") {
            let base = PathBuf::from(app_data);
            candidates.push(base.join("com.ayangweb.EcoPaste"));
            candidates.push(base.join("EcoPaste"));
        }
    }

    candidates
}

fn normalize_import_types(types: Vec<LegacyImportSelection>) -> Vec<LegacyImportSelection> {
    let mut normalized = Vec::new();

    for import_type in types {
        if !normalized.contains(&import_type) {
            normalized.push(import_type);
        }
    }

    normalized
}

fn map_legacy_row_to_item(
    row: &LegacyHistoryRow,
    legacy_images_dir: Option<&Path>,
    image_store: &ImageStore,
    settings: &Settings,
) -> Result<Option<crate::db::models::ClipboardItem>> {
    let now = parse_legacy_datetime(row.create_time.as_deref()).unwrap_or_else(Utc::now);
    let note = row
        .note
        .as_ref()
        .map(|value| value.trim().to_owned())
        .filter(|value| !value.is_empty());
    let item_id = Uuid::new_v4().to_string();

    let payload = match row.item_type.as_deref() {
        Some("text") => {
            let text = row.value.clone().unwrap_or_default();
            if text.trim().is_empty() {
                return Ok(None);
            }

            ClipboardPayload::Text(TextPayload {
                text,
                html: None,
                rtf: None,
            })
        }
        Some("files") => {
            let raw = row.value.as_deref().unwrap_or_default();
            let paths: Vec<String> = serde_json::from_str(raw).unwrap_or_default();
            if paths.is_empty() {
                return Ok(None);
            }

            ClipboardPayload::Files(paths)
        }
        Some("image") => {
            let file_name = row.value.as_deref().unwrap_or_default();
            let Some(images_dir) = legacy_images_dir else {
                return Ok(None);
            };
            let path = images_dir.join(file_name);
            if !path.exists() {
                return Ok(None);
            }

            let bytes = fs::read(&path)
                .with_context(|| format!("failed to read legacy image {}", path.display()))?;
            let width = row.width.unwrap_or_default().max(1) as u32;
            let height = row.height.unwrap_or_default().max(1) as u32;

            ClipboardPayload::Image(ImagePayload {
                bytes,
                width,
                height,
            })
        }
        _ => {
            return Ok(None);
        }
    };

    let mut item = build_item_with_settings(
        image_store,
        &payload,
        &settings.clipboard.capture,
        &settings.clipboard.sensitive,
        false,
    )?
    .ok_or_else(|| AppError::Clipboard("旧版条目无法转换为当前记录".to_owned()))?;

    item.id = item_id;
    item.note = note;
    item.created_at = now;
    item.updated_at = now;
    item.is_pinned = false;
    item.source_app_id = None;

    Ok(Some(item))
}

fn parse_legacy_datetime(value: Option<&str>) -> Option<DateTime<Utc>> {
    let raw = value?.trim();
    if raw.is_empty() {
        return None;
    }

    NaiveDateTime::parse_from_str(raw, "%Y-%m-%d %H:%M:%S")
        .ok()
        .map(|time| DateTime::<Utc>::from_naive_utc_and_offset(time, Utc))
}

use std::fs;
use std::path::{Path, PathBuf};

use anyhow::Context;
use chrono::{DateTime, NaiveDateTime, Utc};
use serde::{Deserialize, Serialize};
use serde_json::json;
use sqlx::sqlite::{SqliteConnectOptions, SqlitePoolOptions};
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
    checked_at: String,
    importable_database: Option<String>,
    importable_item_count: u64,
    normal_item_count: u64,
    favorite_item_count: u64,
    scan_messages: Vec<String>,
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

    window::show_window(&app, window::CLIPBOARD_WINDOW_LABEL)?;

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
    let mut scan_messages = Vec::new();
    let candidates = legacy_data_candidates(&mut scan_messages);

    record_legacy_scan(
        &mut scan_messages,
        format!("checking {} legacy data candidate(s)", candidates.len()),
    );

    for dir in candidates {
        let Some(candidate) = inspect_legacy_dir(&dir, &mut scan_messages).await? else {
            continue;
        };

        if best_match
            .as_ref()
            .is_none_or(|current| is_better_legacy_match(&candidate, current))
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
            checked_at,
            importable_database: None,
            importable_item_count: 0,
            normal_item_count: 0,
            favorite_item_count: 0,
            scan_messages,
        });
    };

    record_legacy_scan(
        &mut scan_messages,
        format!(
            "selected legacy data candidate {} with {} importable item(s)",
            match_data.path.display(),
            match_data.importable_item_count
        ),
    );

    Ok(OnboardingLegacyDataDetection {
        found: true,
        path: Some(match_data.path.display().to_string()),
        database_files: match_data
            .database_files
            .into_iter()
            .map(|path| path.display().to_string())
            .collect(),
        checked_at,
        importable_database: match_data
            .importable_database
            .map(|path| path.display().to_string()),
        importable_item_count: match_data.importable_item_count,
        normal_item_count: match_data.normal_item_count,
        favorite_item_count: match_data.favorite_item_count,
        scan_messages,
    })
}

fn is_better_legacy_match(candidate: &LegacyDataMatch, current: &LegacyDataMatch) -> bool {
    candidate.importable_item_count > current.importable_item_count
}

fn record_legacy_scan(scan_messages: &mut Vec<String>, message: impl Into<String>) {
    let message = message.into();
    log::info!("legacy data detection: {message}");
    scan_messages.push(message);
}

fn record_legacy_scan_warn(scan_messages: &mut Vec<String>, message: impl Into<String>) {
    let message = message.into();
    log::warn!("legacy data detection: {message}");
    scan_messages.push(message);
}

async fn inspect_legacy_dir(
    dir: &Path,
    scan_messages: &mut Vec<String>,
) -> Result<Option<LegacyDataMatch>> {
    record_legacy_scan(
        scan_messages,
        format!("checking legacy data directory {}", dir.display()),
    );

    if !dir.exists() {
        record_legacy_scan(
            scan_messages,
            format!("legacy data directory missing: {}", dir.display()),
        );
        return Ok(None);
    }

    let mut database_files = Vec::new();
    for entry in WalkDir::new(dir).max_depth(4).into_iter().flatten() {
        let path = entry.path();
        if !path.is_file() {
            continue;
        }

        if is_legacy_database_file(path) {
            record_legacy_scan(
                scan_messages,
                format!("found legacy database candidate {}", path.display()),
            );
            database_files.push(path.to_path_buf());
        }
    }

    record_legacy_scan(
        scan_messages,
        format!(
            "scanned legacy data directory {}: {} database file(s)",
            dir.display(),
            database_files.len()
        ),
    );

    if database_files.is_empty() {
        return Ok(None);
    }

    let mut importable_database = None;
    let mut importable_item_count = 0;
    let mut normal_item_count = 0;
    let mut favorite_item_count = 0;

    for db_path in &database_files {
        let Some(stats) = inspect_legacy_database_with_messages(db_path, scan_messages).await?
        else {
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

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum LegacyEnvironment {
    Dev,
    Prod,
}

const fn current_legacy_environment() -> LegacyEnvironment {
    if cfg!(dev) {
        LegacyEnvironment::Dev
    } else {
        LegacyEnvironment::Prod
    }
}

async fn inspect_legacy_database_with_messages(
    path: &Path,
    scan_messages: &mut Vec<String>,
) -> Result<Option<LegacyDatabaseStats>> {
    record_legacy_scan(
        scan_messages,
        format!("opening legacy database candidate {}", path.display()),
    );

    let pool = match open_legacy_pool(path).await {
        Ok(pool) => pool,
        Err(err) => {
            record_legacy_scan_warn(
                scan_messages,
                format!(
                    "failed to open legacy database candidate {}: {err}",
                    path.display()
                ),
            );
            return Ok(None);
        }
    };

    let table_exists: Option<String> = match sqlx::query_scalar(
        "SELECT name FROM sqlite_master WHERE type = 'table' AND name = ? LIMIT 1",
    )
    .bind(LEGACY_HISTORY_TABLE)
    .fetch_optional(&pool)
    .await
    {
        Ok(table_exists) => table_exists,
        Err(err) => {
            record_legacy_scan_warn(
                scan_messages,
                format!(
                    "failed to inspect legacy database schema {}: {err}",
                    path.display()
                ),
            );
            return Ok(None);
        }
    };

    if table_exists.is_none() {
        record_legacy_scan(
            scan_messages,
            format!(
                "legacy database candidate {} has no history table",
                path.display()
            ),
        );
        return Ok(None);
    }

    let row = match sqlx::query_as::<_, (i64, i64, i64)>(
        "SELECT \
            COUNT(*) AS total, \
            COALESCE(SUM(CASE WHEN favorite IS NULL OR favorite = 0 THEN 1 ELSE 0 END), 0) AS normal_count, \
            COALESCE(SUM(CASE WHEN favorite = 1 THEN 1 ELSE 0 END), 0) AS favorite_count \
         FROM history",
    )
    .fetch_one(&pool)
    .await
    {
        Ok(row) => row,
        Err(err) => {
            record_legacy_scan_warn(
                scan_messages,
                format!(
                    "failed to count legacy history rows {}: {err}",
                    path.display()
                ),
            );
            return Ok(None);
        }
    };

    record_legacy_scan(
        scan_messages,
        format!(
            "legacy database candidate {} has {} item(s), {} normal, {} favorite",
            path.display(),
            row.0.max(0),
            row.1.max(0),
            row.2.max(0)
        ),
    );

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

    let env = current_legacy_environment();
    let expected_names = legacy_database_file_names(env);
    if expected_names.contains(&file_name) {
        return true;
    }

    if legacy_other_environment_database_file_names(env).contains(&file_name) {
        return false;
    }

    let is_sqlite = file_name.ends_with(".sqlite")
        || file_name.ends_with(".sqlite3")
        || file_name.ends_with(".db");
    if !is_sqlite {
        return false;
    }

    env == LegacyEnvironment::Prod
        && !file_name.contains(".dev.")
        && !file_name.ends_with(".dev.db")
}

fn legacy_database_file_names(env: LegacyEnvironment) -> &'static [&'static str] {
    match env {
        LegacyEnvironment::Dev => &[
            ".store.dev.db",
            ".window-state.dev.db",
            "EcoPaste.dev.db",
            "EcoPaste.v2.dev.db",
        ],
        LegacyEnvironment::Prod => &[
            ".store.db",
            ".window-state.db",
            "EcoPaste.db",
            "EcoPaste.v2.db",
        ],
    }
}

fn legacy_other_environment_database_file_names(env: LegacyEnvironment) -> &'static [&'static str] {
    match env {
        LegacyEnvironment::Dev => &[
            ".store.db",
            ".window-state.db",
            "EcoPaste.db",
            "EcoPaste.v2.db",
        ],
        LegacyEnvironment::Prod => &[
            ".store.dev.db",
            ".window-state.dev.db",
            "EcoPaste.dev.db",
            "EcoPaste.v2.dev.db",
        ],
    }
}

fn legacy_store_file_names(env: LegacyEnvironment) -> &'static [&'static str] {
    match env {
        LegacyEnvironment::Dev => &[".store.dev.json"],
        LegacyEnvironment::Prod => &[".store.json"],
    }
}

fn legacy_data_candidates(scan_messages: &mut Vec<String>) -> Vec<PathBuf> {
    legacy_data_candidates_from_defaults(default_legacy_data_candidates(), scan_messages)
}

fn legacy_data_candidates_from_defaults(
    defaults: Vec<PathBuf>,
    scan_messages: &mut Vec<String>,
) -> Vec<PathBuf> {
    let mut candidates = Vec::new();

    for dir in defaults {
        push_unique_path(&mut candidates, dir.clone());

        for custom_dir in custom_legacy_data_candidates(&dir, scan_messages) {
            push_unique_path(&mut candidates, custom_dir);
        }
    }

    candidates
}

fn default_legacy_data_candidates() -> Vec<PathBuf> {
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

fn custom_legacy_data_candidates(
    default_dir: &Path,
    scan_messages: &mut Vec<String>,
) -> Vec<PathBuf> {
    let mut candidates = Vec::new();

    for file_name in legacy_store_file_names(current_legacy_environment()) {
        let store_path = default_dir.join(file_name);
        if !store_path.exists() {
            continue;
        }

        let content = match fs::read_to_string(&store_path) {
            Ok(content) => content,
            Err(err) => {
                record_legacy_scan_warn(
                    scan_messages,
                    format!(
                        "failed to read legacy store {}: {err}",
                        store_path.display()
                    ),
                );
                continue;
            }
        };

        let value: serde_json::Value = match serde_json::from_str(&content) {
            Ok(value) => value,
            Err(err) => {
                record_legacy_scan_warn(
                    scan_messages,
                    format!(
                        "failed to parse legacy store {}: {err}",
                        store_path.display()
                    ),
                );
                continue;
            }
        };

        let Some(data_dir) = value
            .pointer("/globalStore/env/saveDataDir")
            .and_then(|value| value.as_str())
            .map(str::trim)
            .filter(|value| !value.is_empty())
        else {
            record_legacy_scan(
                scan_messages,
                format!(
                    "legacy store {} has no custom saveDataDir",
                    store_path.display()
                ),
            );
            continue;
        };

        let data_dir = PathBuf::from(data_dir);
        record_legacy_scan(
            scan_messages,
            format!(
                "legacy store {} points to data directory {}",
                store_path.display(),
                data_dir.display()
            ),
        );
        push_unique_path(&mut candidates, data_dir);
    }

    candidates
}

fn push_unique_path(paths: &mut Vec<PathBuf>, path: PathBuf) {
    if paths.contains(&path) {
        return;
    }

    paths.push(path);
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

    let Some(mut item) = build_item_with_settings(
        image_store,
        &payload,
        &settings.clipboard.capture,
        &settings.clipboard.sensitive,
        false,
    )?
    else {
        return Ok(None);
    };

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

#[cfg(test)]
mod tests {
    use super::*;

    use tempfile::TempDir;

    async fn create_legacy_history_db(path: &Path, rows: &[(i64, &str)]) {
        let pool = SqlitePoolOptions::new()
            .max_connections(1)
            .connect_with(
                SqliteConnectOptions::new()
                    .filename(path)
                    .create_if_missing(true),
            )
            .await
            .unwrap();

        sqlx::query(
            "CREATE TABLE history (
                id TEXT PRIMARY KEY,
                type TEXT,
                value TEXT,
                favorite INTEGER,
                createTime TEXT,
                note TEXT,
                width INTEGER,
                height INTEGER
            )",
        )
        .execute(&pool)
        .await
        .unwrap();

        for (index, (favorite, value)) in rows.iter().enumerate() {
            sqlx::query(
                "INSERT INTO history (id, type, value, favorite, createTime)
                 VALUES (?, 'text', ?, ?, '2026-01-01 00:00:00')",
            )
            .bind(format!("item-{index}"))
            .bind(value)
            .bind(favorite)
            .execute(&pool)
            .await
            .unwrap();
        }

        pool.close().await;
    }

    async fn create_sqlite_without_history(path: &Path) {
        let pool = SqlitePoolOptions::new()
            .max_connections(1)
            .connect_with(
                SqliteConnectOptions::new()
                    .filename(path)
                    .create_if_missing(true),
            )
            .await
            .unwrap();

        sqlx::query("CREATE TABLE window_state (id TEXT PRIMARY KEY)")
            .execute(&pool)
            .await
            .unwrap();

        pool.close().await;
    }

    fn write_legacy_store(default_dir: &Path, data_dir: &Path) {
        let content = json!({
            "globalStore": {
                "env": {
                    "saveDataDir": data_dir,
                },
            },
        });

        fs::write(
            default_dir.join(legacy_store_file_names(current_legacy_environment())[0]),
            serde_json::to_string_pretty(&content).unwrap(),
        )
        .unwrap();
    }

    #[tokio::test]
    async fn inspect_legacy_dir_counts_default_history_database() {
        let temp = TempDir::new().unwrap();
        let db_path = temp
            .path()
            .join(legacy_database_file_names(current_legacy_environment())[2]);
        create_legacy_history_db(&db_path, &[(0, "normal"), (1, "favorite")]).await;

        let mut messages = Vec::new();
        let result = inspect_legacy_dir(temp.path(), &mut messages)
            .await
            .unwrap()
            .unwrap();

        assert_eq!(result.importable_database, Some(db_path));
        assert_eq!(result.importable_item_count, 2);
        assert_eq!(result.normal_item_count, 1);
        assert_eq!(result.favorite_item_count, 1);
        assert!(messages
            .iter()
            .any(|message| message.contains("has 2 item(s)")));
    }

    #[test]
    fn legacy_text_over_capture_limit_is_skipped() {
        let temp = TempDir::new().unwrap();
        let image_store = ImageStore::for_test(temp.path().join("images"));
        let mut settings = Settings::default();
        settings.clipboard.capture.max_text_mb = 1;
        let row = LegacyHistoryRow {
            item_type: Some("text".to_owned()),
            value: Some("a".repeat(1024 * 1024 + 1)),
            favorite: Some(0),
            create_time: Some("2026-01-01 00:00:00".to_owned()),
            note: None,
            width: None,
            height: None,
        };

        let result = map_legacy_row_to_item(&row, None, &image_store, &settings).unwrap();

        assert!(result.is_none());
    }

    #[tokio::test]
    async fn inspect_legacy_dir_reports_found_directory_without_history() {
        let temp = TempDir::new().unwrap();
        fs::create_dir_all(temp.path().join("images")).unwrap();
        fs::write(temp.path().join("images").join("old.png"), [1_u8, 2, 3]).unwrap();
        create_sqlite_without_history(
            &temp
                .path()
                .join(legacy_database_file_names(current_legacy_environment())[1]),
        )
        .await;

        let mut messages = Vec::new();
        let result = inspect_legacy_dir(temp.path(), &mut messages)
            .await
            .unwrap()
            .unwrap();

        assert!(result.importable_database.is_none());
        assert_eq!(result.importable_item_count, 0);
        assert!(messages
            .iter()
            .any(|message| message.contains("has no history table")));
    }

    #[tokio::test]
    async fn custom_store_directory_is_added_as_candidate() {
        let temp = TempDir::new().unwrap();
        let default_dir = temp.path().join("default");
        let custom_dir = temp.path().join("custom");
        fs::create_dir_all(&default_dir).unwrap();
        fs::create_dir_all(&custom_dir).unwrap();
        write_legacy_store(&default_dir, &custom_dir);

        let mut messages = Vec::new();
        let candidates =
            legacy_data_candidates_from_defaults(vec![default_dir.clone()], &mut messages);

        assert_eq!(candidates, vec![default_dir, custom_dir]);
        assert!(messages
            .iter()
            .any(|message| message.contains("points to data directory")));
    }

    #[tokio::test]
    async fn custom_candidate_with_items_beats_larger_default_residue() {
        let temp = TempDir::new().unwrap();
        let default_dir = temp.path().join("default");
        let custom_dir = temp.path().join("custom");
        fs::create_dir_all(default_dir.join("images")).unwrap();
        fs::create_dir_all(&custom_dir).unwrap();
        fs::write(
            default_dir.join("images").join("large.bin"),
            vec![7_u8; 4096],
        )
        .unwrap();
        write_legacy_store(&default_dir, &custom_dir);
        create_legacy_history_db(
            &custom_dir.join(legacy_database_file_names(current_legacy_environment())[2]),
            &[(0, "custom")],
        )
        .await;

        let mut best_match: Option<LegacyDataMatch> = None;
        let mut messages = Vec::new();
        for dir in legacy_data_candidates_from_defaults(vec![default_dir], &mut messages) {
            let Some(candidate) = inspect_legacy_dir(&dir, &mut messages).await.unwrap() else {
                continue;
            };

            if best_match
                .as_ref()
                .is_none_or(|current| is_better_legacy_match(&candidate, current))
            {
                best_match = Some(candidate);
            }
        }

        let result = best_match.unwrap();
        assert_eq!(result.path, custom_dir);
        assert_eq!(result.importable_item_count, 1);
    }

    #[tokio::test]
    async fn broken_legacy_store_does_not_stop_default_candidate() {
        let temp = TempDir::new().unwrap();
        let default_dir = temp.path().join("default");
        fs::create_dir_all(&default_dir).unwrap();
        fs::write(
            default_dir.join(legacy_store_file_names(current_legacy_environment())[0]),
            "{bad json",
        )
        .unwrap();

        let mut messages = Vec::new();
        let candidates =
            legacy_data_candidates_from_defaults(vec![default_dir.clone()], &mut messages);

        assert_eq!(candidates, vec![default_dir]);
        assert!(messages
            .iter()
            .any(|message| message.contains("failed to parse legacy store")));
    }

    #[test]
    fn legacy_database_file_filter_matches_current_environment_only() {
        assert_eq!(
            legacy_database_file_names(LegacyEnvironment::Dev),
            [
                ".store.dev.db",
                ".window-state.dev.db",
                "EcoPaste.dev.db",
                "EcoPaste.v2.dev.db",
            ]
        );
        assert_eq!(
            legacy_database_file_names(LegacyEnvironment::Prod),
            [
                ".store.db",
                ".window-state.db",
                "EcoPaste.db",
                "EcoPaste.v2.db"
            ]
        );

        assert_eq!(
            legacy_other_environment_database_file_names(LegacyEnvironment::Dev),
            [
                ".store.db",
                ".window-state.db",
                "EcoPaste.db",
                "EcoPaste.v2.db"
            ]
        );
        assert_eq!(
            legacy_other_environment_database_file_names(LegacyEnvironment::Prod),
            [
                ".store.dev.db",
                ".window-state.dev.db",
                "EcoPaste.dev.db",
                "EcoPaste.v2.dev.db",
            ]
        );

        assert_eq!(
            is_legacy_database_file(Path::new("EcoPaste.dev.db")),
            current_legacy_environment() == LegacyEnvironment::Dev
        );
        assert_eq!(
            is_legacy_database_file(Path::new("EcoPaste.db")),
            current_legacy_environment() == LegacyEnvironment::Prod
        );
    }

    #[test]
    fn legacy_store_file_filter_matches_current_environment_only() {
        assert_eq!(
            legacy_store_file_names(LegacyEnvironment::Dev),
            [".store.dev.json"]
        );
        assert_eq!(
            legacy_store_file_names(LegacyEnvironment::Prod),
            [".store.json"]
        );

        let temp = TempDir::new().unwrap();
        let default_dir = temp.path().join("default");
        let custom_dir = temp.path().join("custom");
        fs::create_dir_all(&default_dir).unwrap();
        fs::create_dir_all(&custom_dir).unwrap();

        fs::write(
            default_dir.join(if cfg!(dev) {
                ".store.json"
            } else {
                ".store.dev.json"
            }),
            serde_json::to_string_pretty(&json!({
                "globalStore": {
                    "env": {
                        "saveDataDir": custom_dir,
                    },
                },
            }))
            .unwrap(),
        )
        .unwrap();

        let mut messages = Vec::new();
        let candidates =
            legacy_data_candidates_from_defaults(vec![default_dir.clone()], &mut messages);

        assert_eq!(candidates, vec![default_dir]);
    }
}

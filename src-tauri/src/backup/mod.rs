//! EcoPaste 历史备份包导出与接收壳识别。
//!
//! `.ecopastebak` 有两种格式：明文模式是标准 ZIP；加密模式是 EcoPaste 自有容器。

use std::fs::{self, File};
use std::io::{BufReader, BufWriter, Cursor, Read, Seek, Write};
use std::path::{Path, PathBuf};
use std::sync::{LazyLock, Mutex};

use anyhow::{anyhow, Context};
use argon2::{Algorithm, Argon2, Params, Version};
use chacha20poly1305::aead::{Aead, KeyInit};
use chacha20poly1305::{XChaCha20Poly1305, XNonce};
use chrono::{DateTime, Utc};
use rand::RngCore;
use serde::{Deserialize, Serialize};
use sqlx::sqlite::{SqliteConnectOptions, SqliteJournalMode, SqlitePoolOptions};
use sqlx::{Sqlite, SqlitePool};
use tauri::{AppHandle, Emitter, Manager};
use tempfile::{NamedTempFile, TempDir};
use walkdir::WalkDir;
use zeroize::Zeroizing;
use zip::write::SimpleFileOptions;
use zip::{CompressionMethod, ZipArchive, ZipWriter};

use crate::core::{AppError, Result};

pub const BACKUP_EXTENSION: &str = "ecopastebak";
pub const BACKUP_RECEIVED_EVENT: &str = "backup://received";

const MAGIC: &[u8; 12] = b"ECOPASTEBAK1";
const ZIP_MAGIC: &[u8; 2] = b"PK";
const HEADER_LEN_BYTES: usize = 4;
const FORMAT_VERSION: u16 = 1;
const ARGON2_MEMORY_KIB: u32 = 64 * 1024;
const ARGON2_TIME_COST: u32 = 3;
const ARGON2_PARALLELISM: u32 = 1;
const SALT_LEN: usize = 16;
const NONCE_LEN: usize = 24;
const KEY_LEN: usize = 32;
const SETTINGS_FILENAME: &str = "settings.json";
const DB_FILENAME: &str = "clipboard.db";
const DB_ARCHIVE_DIR: &str = "db";
const RESOURCES_ARCHIVE_DIR: &str = "resources";
const CONFIG_ARCHIVE_DIR: &str = "config";
const MANIFEST_FILENAME: &str = "manifest.json";

/// 偏好窗口被销毁时暂存的待处理备份接收事件。
///
/// `emit_received_backup` 在 preference 不存在（已空闲销毁）时无法 push 事件——重建是异步的，
/// 前端 listener 尚未挂载。改为存入此 slot，由前端重建后通过 `take_pending_backup` 主动拉取，
/// 两条路径（存活 push / 销毁 pull）互斥，避免事件丢失。
static PENDING_BACKUP: LazyLock<Mutex<Option<BackupReceivedPayload>>> =
    LazyLock::new(|| Mutex::new(None));

#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ExportHistoryBackupOptions {
    pub mode: BackupExportMode,
    pub password: Option<String>,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ImportHistoryBackupInput {
    pub path: String,
    pub password: Option<String>,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ImportHistoryBackupOptions {
    pub strategy: BackupImportStrategy,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum BackupImportStrategy {
    Merge,
    Overwrite,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum BackupExportMode {
    Encrypted,
    Plain,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ExportHistoryBackupResult {
    pub path: String,
    pub total_bytes: u64,
    pub item_count: i64,
    pub text_count: i64,
    pub image_count: i64,
    pub files_count: i64,
    pub resource_bytes: u64,
    pub exported_at: DateTime<Utc>,
    pub mode: BackupExportMode,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ImportHistoryBackupResult {
    pub strategy: BackupImportStrategy,
    pub imported_items: u64,
    pub skipped_items: u64,
    pub imported_resources: u64,
    pub imported_settings: bool,
    pub requires_restart: bool,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct BackupReceivedPayload {
    pub path: String,
    pub source: BackupReceiveSource,
    pub mode: BackupContainerMode,
}

#[derive(Debug, Clone, Copy, Serialize)]
#[serde(rename_all = "camelCase")]
pub enum BackupReceiveSource {
    OpenFile,
    DragDrop,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum BackupContainerMode {
    Encrypted,
    Plain,
}

#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
struct ContainerHeader {
    format_version: u16,
    mode: BackupContainerMode,
    kdf: Option<KdfHeader>,
    cipher: Option<CipherHeader>,
}

#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
struct KdfHeader {
    algorithm: String,
    memory_kib: u32,
    time_cost: u32,
    parallelism: u32,
    salt: Vec<u8>,
}

#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
struct CipherHeader {
    algorithm: String,
    nonce: Vec<u8>,
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
struct BackupManifest {
    format_version: u16,
    app_name: String,
    app_version: String,
    exported_at: DateTime<Utc>,
    platform: String,
    encryption: ManifestEncryption,
    item_count: i64,
    text_count: i64,
    image_count: i64,
    files_count: i64,
    resource_bytes: u64,
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
enum ManifestEncryption {
    None,
    Password,
}

#[derive(Debug, Clone, Copy)]
struct BackupCounts {
    item_count: i64,
    text_count: i64,
    image_count: i64,
    files_count: i64,
}

#[derive(Debug, Clone)]
struct BackupSourcePaths {
    db_path: PathBuf,
    resources_dir: PathBuf,
    settings_path: PathBuf,
}

/// 导出当前环境历史数据库、资源文件和设置为 `.ecopastebak` 备份包。
pub async fn export_history_backup(
    app: &AppHandle,
    pool: &SqlitePool,
    target_path: String,
    options: ExportHistoryBackupOptions,
) -> Result<ExportHistoryBackupResult> {
    let target = normalize_backup_path(PathBuf::from(target_path))?;
    let password = validate_password_options(&options)?;
    let exported_at = Utc::now();
    let counts = load_counts(pool).await?;
    let source_paths = backup_source_paths(app)?;
    let resource_bytes = dir_size(&source_paths.resources_dir)?;
    let manifest = build_manifest(app, exported_at, options.mode, counts, resource_bytes)?;

    checkpoint_database(pool).await?;

    let payload_file = NamedTempFile::new().context("failed to create temporary backup payload")?;
    write_payload_zip(&source_paths, payload_file.path(), &manifest, &target)?;
    let total_bytes = write_container(&target, payload_file.path(), options.mode, password)?;

    Ok(ExportHistoryBackupResult {
        path: target.to_string_lossy().into_owned(),
        total_bytes,
        item_count: counts.item_count,
        text_count: counts.text_count,
        image_count: counts.image_count,
        files_count: counts.files_count,
        resource_bytes,
        exported_at,
        mode: options.mode,
    })
}

/// 从 `.ecopastebak` 导入历史和设置；合并写入当前库，覆盖热替换当前数据。
pub async fn import_history_backup(
    app: &AppHandle,
    db: &crate::db::DatabaseState,
    input: ImportHistoryBackupInput,
    options: ImportHistoryBackupOptions,
) -> Result<ImportHistoryBackupResult> {
    validate_import_options(&input, &options)?;

    let path = PathBuf::from(input.path);
    ensure_backup_extension(&path)?;
    let payload = read_backup_payload(&path, input.password.as_deref())?;
    let temp = extract_payload_zip(&payload)?;
    validate_extracted_payload(temp.path())?;

    match options.strategy {
        BackupImportStrategy::Merge => {
            let pool = db.pool().await;
            merge_import(app, &pool, temp.path(), &options).await
        }
        BackupImportStrategy::Overwrite => overwrite_import(app, db, temp.path(), &options).await,
    }
}

/// 识别 `.ecopastebak` 文件头并返回容器模式；不解密、不导入。
pub fn inspect_backup_file(path: &Path) -> Result<BackupContainerMode> {
    ensure_backup_extension(path)?;

    let mut file = File::open(path).with_context(|| format!("failed to open backup {path:?}"))?;
    inspect_backup_reader(&mut file)
}

/// 将系统打开文件或拖入文件统一转成偏好页接收事件。
///
/// preference 已改为空闲可销毁窗口：若窗口仍存活，照常 show + push 事件；
/// 若已销毁，先把 payload 存入 [`PENDING_BACKUP`]，再 show 触发重建——
/// 前端重建后经 `take_pending_backup` 主动拉取，规避「重建异步、push 丢失」竞态。
pub fn emit_received_backup(
    app: &AppHandle,
    path: PathBuf,
    source: BackupReceiveSource,
) -> Result<()> {
    let mode = inspect_backup_file(&path)?;

    let payload = BackupReceivedPayload {
        path: path.to_string_lossy().into_owned(),
        source,
        mode,
    };

    let exists = app
        .get_webview_window(crate::window::PREFERENCE_WINDOW_LABEL)
        .is_some();

    if !exists {
        set_pending_backup(payload.clone());
    }

    crate::window::show_window(app, crate::window::PREFERENCE_WINDOW_LABEL)?;

    if exists {
        app.emit(BACKUP_RECEIVED_EVENT, payload)
            .context("failed to emit backup received event")?;
    }

    Ok(())
}

/// 存入待处理备份接收事件，覆盖旧值（仅保留最近一次）。
fn set_pending_backup(payload: BackupReceivedPayload) {
    let mut guard = PENDING_BACKUP.lock().unwrap_or_else(|poisoned| {
        log::error!("pending backup mutex poisoned on set, recovering");
        poisoned.into_inner()
    });
    *guard = Some(payload);
}

/// 取走并清空待处理备份接收事件，供偏好窗口重建后首屏拉取。
pub fn take_pending_backup() -> Option<BackupReceivedPayload> {
    let mut guard = PENDING_BACKUP.lock().unwrap_or_else(|poisoned| {
        log::error!("pending backup mutex poisoned on take, recovering");
        poisoned.into_inner()
    });
    guard.take()
}

/// 从进程参数中查找 `.ecopastebak` 路径，供 Windows 文件关联和第二实例回调使用。
pub fn backup_path_from_args(args: &[String]) -> Option<PathBuf> {
    args.iter()
        .map(PathBuf::from)
        .find(|path| is_backup_path(path))
}

/// 判断路径是否看起来是 EcoPaste 备份包。
pub fn is_backup_path(path: &Path) -> bool {
    path.extension()
        .and_then(|value| value.to_str())
        .is_some_and(|ext| ext.eq_ignore_ascii_case(BACKUP_EXTENSION))
}

fn validate_password_options(
    options: &ExportHistoryBackupOptions,
) -> Result<Option<Zeroizing<String>>> {
    match options.mode {
        BackupExportMode::Encrypted => {
            let Some(password) = options.password.as_ref() else {
                return app_error("请输入备份密码");
            };
            if password.chars().count() < 8 {
                return app_error("备份密码至少需要 8 个字符");
            }

            Ok(Some(Zeroizing::new(password.clone())))
        }
        BackupExportMode::Plain => {
            if options
                .password
                .as_deref()
                .is_some_and(|value| !value.is_empty())
            {
                return app_error("明文备份不应包含密码");
            }

            Ok(None)
        }
    }
}

fn validate_import_options(
    input: &ImportHistoryBackupInput,
    _options: &ImportHistoryBackupOptions,
) -> Result<()> {
    let path = PathBuf::from(input.path.as_str());
    match inspect_backup_file(&path)? {
        BackupContainerMode::Encrypted => {
            if input.password.as_deref().is_none_or(str::is_empty) {
                return app_error("请输入备份密码");
            }
        }
        BackupContainerMode::Plain => {
            if input
                .password
                .as_deref()
                .is_some_and(|value| !value.is_empty())
            {
                return app_error("明文备份不应包含密码");
            }
        }
    }

    Ok(())
}

fn normalize_backup_path(path: PathBuf) -> Result<PathBuf> {
    if path.as_os_str().is_empty() {
        return app_error("请选择备份保存位置");
    }

    match path.extension().and_then(|value| value.to_str()) {
        Some(ext) if ext.eq_ignore_ascii_case(BACKUP_EXTENSION) => Ok(path),
        Some(_) => app_error(format!("备份文件后缀必须是 .{BACKUP_EXTENSION}")),
        None => Ok(path.with_extension(BACKUP_EXTENSION)),
    }
}

fn ensure_backup_extension(path: &Path) -> Result<()> {
    if is_backup_path(path) {
        return Ok(());
    }

    app_error(format!("请选择 .{BACKUP_EXTENSION} 备份文件"))
}

async fn checkpoint_database(pool: &SqlitePool) -> Result<()> {
    sqlx::query("PRAGMA wal_checkpoint(TRUNCATE)")
        .execute(pool)
        .await
        .context("failed to checkpoint backup database")?;

    Ok(())
}

async fn load_counts(pool: &SqlitePool) -> Result<BackupCounts> {
    let item_count = count_items(pool, None).await?;
    let text_count = count_items(pool, Some("text")).await?;
    let image_count = count_items(pool, Some("image")).await?;
    let files_count = count_items(pool, Some("files")).await?;

    Ok(BackupCounts {
        item_count,
        text_count,
        image_count,
        files_count,
    })
}

async fn count_items(pool: &SqlitePool, kind: Option<&str>) -> Result<i64> {
    let count = if let Some(kind) = kind {
        sqlx::query_scalar::<_, i64>("SELECT COUNT(*) FROM clipboard_items WHERE kind = ?")
            .bind(kind)
            .fetch_one(pool)
            .await
    } else {
        sqlx::query_scalar::<_, i64>("SELECT COUNT(*) FROM clipboard_items")
            .fetch_one(pool)
            .await
    }
    .context("failed to count backup items")?;

    Ok(count)
}

fn build_manifest(
    app: &AppHandle,
    exported_at: DateTime<Utc>,
    mode: BackupExportMode,
    counts: BackupCounts,
    resource_bytes: u64,
) -> Result<BackupManifest> {
    Ok(BackupManifest {
        format_version: FORMAT_VERSION,
        app_name: app.package_info().name.to_string(),
        app_version: app.package_info().version.to_string(),
        exported_at,
        platform: current_platform().to_owned(),
        encryption: match mode {
            BackupExportMode::Encrypted => ManifestEncryption::Password,
            BackupExportMode::Plain => ManifestEncryption::None,
        },
        item_count: counts.item_count,
        text_count: counts.text_count,
        image_count: counts.image_count,
        files_count: counts.files_count,
        resource_bytes,
    })
}

fn current_platform() -> &'static str {
    if cfg!(target_os = "macos") {
        "macos"
    } else {
        "windows"
    }
}

/// 汇总备份允许写入包内的源路径，避免把日志、缓存、临时文件等环境杂项带进迁移包。
fn backup_source_paths(app: &AppHandle) -> Result<BackupSourcePaths> {
    Ok(BackupSourcePaths {
        db_path: crate::db::db_path(app)?,
        resources_dir: crate::core::paths::resources_dir(app)?,
        settings_path: crate::core::paths::config_dir(app)?.join(SETTINGS_FILENAME),
    })
}

fn write_payload_zip(
    source_paths: &BackupSourcePaths,
    path: &Path,
    manifest: &BackupManifest,
    target_path: &Path,
) -> Result<()> {
    let file = File::create(path).with_context(|| format!("failed to create payload {path:?}"))?;
    let mut zip = ZipWriter::new(BufWriter::new(file));
    let options = SimpleFileOptions::default().compression_method(CompressionMethod::Deflated);

    zip.start_file("manifest.json", options)
        .context("failed to write backup manifest entry")?;
    let manifest_bytes =
        serde_json::to_vec_pretty(manifest).context("failed to serialize backup manifest")?;
    zip.write_all(&manifest_bytes)
        .context("failed to write backup manifest")?;

    add_optional_file(
        &mut zip,
        &source_paths.db_path,
        &archive_path(DB_ARCHIVE_DIR, Path::new("clipboard.db"))?,
        options,
        target_path,
    )?;
    add_dir_contents(
        &mut zip,
        &source_paths.resources_dir,
        RESOURCES_ARCHIVE_DIR,
        options,
        target_path,
    )?;
    add_optional_file(
        &mut zip,
        &source_paths.settings_path,
        &archive_path(CONFIG_ARCHIVE_DIR, Path::new(SETTINGS_FILENAME))?,
        options,
        target_path,
    )?;

    zip.finish()
        .context("failed to finish backup payload archive")?;

    Ok(())
}

fn add_file<W: Write + Seek>(
    zip: &mut ZipWriter<W>,
    path: &Path,
    archive_name: &str,
    options: SimpleFileOptions,
) -> Result<()> {
    zip.start_file(archive_name, options)
        .with_context(|| format!("failed to start archive file {archive_name}"))?;
    let mut file = File::open(path).with_context(|| format!("failed to open {path:?}"))?;
    std::io::copy(&mut file, zip).with_context(|| format!("failed to archive {path:?}"))?;

    Ok(())
}

fn add_optional_file<W: Write + Seek>(
    zip: &mut ZipWriter<W>,
    path: &Path,
    archive_name: &str,
    options: SimpleFileOptions,
    target_path: &Path,
) -> Result<()> {
    if !path.exists() || same_path(path, target_path) {
        return Ok(());
    }

    add_file(zip, path, archive_name, options)
}

fn add_dir_contents<W: Write + Seek>(
    zip: &mut ZipWriter<W>,
    root: &Path,
    archive_root: &str,
    options: SimpleFileOptions,
    target_path: &Path,
) -> Result<()> {
    if !root.exists() {
        return Ok(());
    }

    for entry in WalkDir::new(root)
        .into_iter()
        .filter_map(std::result::Result::ok)
    {
        let path = entry.path();
        let metadata = entry
            .metadata()
            .with_context(|| format!("failed to read metadata at {path:?}"))?;
        if !metadata.is_file() {
            continue;
        }
        let Some(file_name) = path.file_name().and_then(|value| value.to_str()) else {
            continue;
        };
        if should_skip_backup_path(path, file_name, target_path) {
            continue;
        }

        let relative = path
            .strip_prefix(root)
            .with_context(|| format!("failed to strip root {root:?} from {path:?}"))?;
        let archive_name = archive_path(archive_root, relative)?;
        add_file(zip, path, &archive_name, options)?;
    }

    Ok(())
}

fn archive_path(prefix: &str, relative: &Path) -> Result<String> {
    let mut parts = Vec::new();
    if !prefix.is_empty() {
        parts.push(prefix.to_owned());
    }
    for part in relative.components() {
        let std::path::Component::Normal(value) = part else {
            return app_error("backup path contains unsupported component");
        };
        let value = value
            .to_str()
            .ok_or_else(|| anyhow!("backup path is not valid utf-8"))?;
        parts.push(value.to_owned());
    }

    Ok(parts.join("/"))
}

fn should_skip_backup_path(path: &Path, file_name: &str, target_path: &Path) -> bool {
    if same_path(path, target_path) {
        return true;
    }

    file_name.ends_with(".tmp")
        || file_name.ends_with(".temp")
        || file_name.starts_with(".tmp")
        || file_name == ".DS_Store"
}

fn same_path(left: &Path, right: &Path) -> bool {
    let left = fs::canonicalize(left).unwrap_or_else(|_| left.to_path_buf());
    let right = fs::canonicalize(right).unwrap_or_else(|_| right.to_path_buf());

    left == right
}

fn write_container(
    target: &Path,
    payload_path: &Path,
    mode: BackupExportMode,
    password: Option<Zeroizing<String>>,
) -> Result<u64> {
    let parent = target
        .parent()
        .filter(|path| !path.as_os_str().is_empty())
        .unwrap_or_else(|| Path::new("."));
    fs::create_dir_all(parent).with_context(|| format!("failed to create {parent:?}"))?;

    let mut temp = NamedTempFile::new_in(parent)
        .with_context(|| format!("failed to create temporary file under {parent:?}"))?;
    match mode {
        BackupExportMode::Plain => {
            copy_file_into_writer(payload_path, &mut temp)?;
        }
        BackupExportMode::Encrypted => {
            let password = password.ok_or_else(|| anyhow!("missing backup password"))?;
            let mut payload = Vec::new();
            File::open(payload_path)
                .with_context(|| format!("failed to open payload {payload_path:?}"))?
                .read_to_end(&mut payload)
                .context("failed to read backup payload")?;

            let encrypted = encrypt_payload(&payload, &password)?;
            write_header(&mut temp, &encrypted.header)?;
            temp.write_all(&encrypted.ciphertext)
                .context("failed to write encrypted backup payload")?;
        }
    }

    temp.flush().context("failed to flush backup file")?;
    temp.as_file()
        .sync_all()
        .context("failed to sync backup file")?;
    temp.persist(target)
        .map_err(|err| anyhow!(err))
        .with_context(|| format!("failed to persist backup to {target:?}"))?;

    let total_bytes = fs::metadata(target)
        .with_context(|| format!("failed to read backup metadata {target:?}"))?
        .len();

    Ok(total_bytes)
}

struct EncryptedPayload {
    header: ContainerHeader,
    ciphertext: Vec<u8>,
}

fn encrypt_payload(payload: &[u8], password: &str) -> Result<EncryptedPayload> {
    let mut salt = [0u8; SALT_LEN];
    let mut nonce = [0u8; NONCE_LEN];
    rand::rngs::OsRng.fill_bytes(&mut salt);
    rand::rngs::OsRng.fill_bytes(&mut nonce);

    let mut key = Zeroizing::new([0u8; KEY_LEN]);
    let params = Params::new(
        ARGON2_MEMORY_KIB,
        ARGON2_TIME_COST,
        ARGON2_PARALLELISM,
        Some(KEY_LEN),
    )
    .context("failed to build argon2 params")?;
    let argon2 = Argon2::new(Algorithm::Argon2id, Version::V0x13, params);
    argon2
        .hash_password_into(password.as_bytes(), &salt, key.as_mut())
        .context("failed to derive backup key")?;

    let cipher = XChaCha20Poly1305::new_from_slice(key.as_ref())
        .context("failed to initialize backup cipher")?;
    let nonce = XNonce::from(nonce);
    let ciphertext = cipher
        .encrypt(&nonce, payload)
        .map_err(|_| anyhow!("failed to encrypt backup payload"))?;

    Ok(EncryptedPayload {
        header: ContainerHeader {
            format_version: FORMAT_VERSION,
            mode: BackupContainerMode::Encrypted,
            kdf: Some(KdfHeader {
                algorithm: "argon2id".to_owned(),
                memory_kib: ARGON2_MEMORY_KIB,
                time_cost: ARGON2_TIME_COST,
                parallelism: ARGON2_PARALLELISM,
                salt: salt.to_vec(),
            }),
            cipher: Some(CipherHeader {
                algorithm: "xchacha20poly1305".to_owned(),
                nonce: nonce.to_vec(),
            }),
        },
        ciphertext,
    })
}

fn write_header<W: Write>(writer: &mut W, header: &ContainerHeader) -> Result<()> {
    let header_bytes = serde_json::to_vec(header).context("failed to serialize backup header")?;
    let header_len: u32 = header_bytes
        .len()
        .try_into()
        .context("backup header is too large")?;

    writer
        .write_all(MAGIC)
        .context("failed to write backup magic")?;
    writer
        .write_all(&header_len.to_le_bytes())
        .context("failed to write backup header length")?;
    writer
        .write_all(&header_bytes)
        .context("failed to write backup header")?;

    Ok(())
}

fn read_backup_payload(path: &Path, password: Option<&str>) -> Result<Vec<u8>> {
    let mut file = File::open(path).with_context(|| format!("failed to open backup {path:?}"))?;
    let mut bytes = Vec::new();
    file.read_to_end(&mut bytes)
        .with_context(|| format!("failed to read backup {path:?}"))?;

    if bytes.starts_with(ZIP_MAGIC) {
        return Ok(bytes);
    }
    if !bytes.starts_with(MAGIC) {
        return app_error("不是有效的 EcoPaste 备份文件");
    }

    let mut cursor = Cursor::new(bytes.as_slice());
    cursor.set_position(MAGIC.len() as u64);
    let header = read_container_header_after_magic(&mut cursor)?;
    let ciphertext_start = cursor.position() as usize;
    let Some(kdf) = header.kdf else {
        return app_error("加密备份文件头无效");
    };
    let Some(cipher) = header.cipher else {
        return app_error("加密备份文件头无效");
    };
    let Some(password) = password else {
        return app_error("请输入备份密码");
    };

    decrypt_payload(&bytes[ciphertext_start..], password, &kdf, &cipher)
}

fn decrypt_payload(
    ciphertext: &[u8],
    password: &str,
    kdf: &KdfHeader,
    cipher_header: &CipherHeader,
) -> Result<Vec<u8>> {
    if kdf.algorithm != "argon2id" || cipher_header.algorithm != "xchacha20poly1305" {
        return app_error("暂不支持该备份加密格式");
    }
    if kdf.salt.len() != SALT_LEN || cipher_header.nonce.len() != NONCE_LEN {
        return app_error("加密备份文件头无效");
    }

    let mut key = Zeroizing::new([0u8; KEY_LEN]);
    let params = Params::new(
        kdf.memory_kib,
        kdf.time_cost,
        kdf.parallelism,
        Some(KEY_LEN),
    )
    .context("failed to build argon2 params")?;
    let argon2 = Argon2::new(Algorithm::Argon2id, Version::V0x13, params);
    argon2
        .hash_password_into(password.as_bytes(), &kdf.salt, key.as_mut())
        .context("failed to derive backup key")?;

    let cipher =
        XChaCha20Poly1305::new_from_slice(key.as_ref()).context("failed to initialize cipher")?;
    let mut nonce = [0u8; NONCE_LEN];
    nonce.copy_from_slice(&cipher_header.nonce);
    let nonce = XNonce::from(nonce);
    cipher
        .decrypt(&nonce, ciphertext)
        .map_err(|_| AppError::Other(anyhow!("备份密码不正确或文件已损坏")))
}

fn extract_payload_zip(payload: &[u8]) -> Result<TempDir> {
    let temp = tempfile::tempdir().context("failed to create temporary import directory")?;
    let cursor = Cursor::new(payload);
    let mut archive = ZipArchive::new(cursor).context("failed to read backup zip payload")?;
    archive
        .extract(temp.path())
        .context("failed to extract backup payload")?;

    Ok(temp)
}

fn validate_extracted_payload(root: &Path) -> Result<()> {
    if !root.join(MANIFEST_FILENAME).exists() {
        return app_error("备份文件缺少 manifest.json");
    }
    if !root.join(DB_ARCHIVE_DIR).join(DB_FILENAME).exists() {
        return app_error("备份文件缺少历史数据库");
    }
    if !root
        .join(CONFIG_ARCHIVE_DIR)
        .join(SETTINGS_FILENAME)
        .exists()
    {
        return app_error("备份文件缺少设置文件");
    }

    Ok(())
}

fn inspect_backup_reader<R: Read>(reader: &mut R) -> Result<BackupContainerMode> {
    let mut magic = [0u8; MAGIC.len()];
    let read = reader
        .read(&mut magic)
        .context("failed to read backup magic")?;
    if read >= MAGIC.len() && &magic == MAGIC {
        let header = read_container_header_after_magic(reader)?;
        return Ok(header.mode);
    }
    if read >= ZIP_MAGIC.len() && &magic[..ZIP_MAGIC.len()] == ZIP_MAGIC {
        return Ok(BackupContainerMode::Plain);
    }

    app_error("不是有效的 EcoPaste 备份文件")
}

fn read_container_header_after_magic<R: Read>(reader: &mut R) -> Result<ContainerHeader> {
    let mut len = [0u8; HEADER_LEN_BYTES];
    reader
        .read_exact(&mut len)
        .context("failed to read backup header length")?;
    let header_len = u32::from_le_bytes(len) as usize;
    if header_len == 0 || header_len > 64 * 1024 {
        return app_error("备份文件头无效");
    }

    let mut header = vec![0u8; header_len];
    reader
        .read_exact(&mut header)
        .context("failed to read backup header")?;
    let header: ContainerHeader =
        serde_json::from_slice(&header).context("failed to parse backup header")?;
    if header.format_version != FORMAT_VERSION {
        return app_error("暂不支持该备份格式版本");
    }

    Ok(header)
}

async fn merge_import(
    app: &AppHandle,
    pool: &SqlitePool,
    root: &Path,
    _options: &ImportHistoryBackupOptions,
) -> Result<ImportHistoryBackupResult> {
    let db_path = root.join(DB_ARCHIVE_DIR).join(DB_FILENAME);
    let backup_pool = open_backup_db(&db_path).await?;
    let outcome = merge_history(pool, &backup_pool).await?;
    let resources = root.join(RESOURCES_ARCHIVE_DIR);
    let imported_resources =
        copy_dir_missing(&resources, &crate::core::paths::resources_dir(app)?)?;
    emit_clipboard_imported(app);

    let settings_path = root.join(CONFIG_ARCHIVE_DIR).join(SETTINGS_FILENAME);
    let patch = read_json_file(&settings_path)?;
    let next = app
        .state::<crate::settings::SettingsStore>()
        .update(patch)?;
    emit_settings_updated(app, &next);

    Ok(ImportHistoryBackupResult {
        strategy: BackupImportStrategy::Merge,
        imported_items: outcome.imported_items,
        skipped_items: outcome.skipped_items,
        imported_resources,
        imported_settings: true,
        requires_restart: false,
    })
}

async fn overwrite_import(
    app: &AppHandle,
    db: &crate::db::DatabaseState,
    root: &Path,
    _options: &ImportHistoryBackupOptions,
) -> Result<ImportHistoryBackupResult> {
    let _pause_guard = pause_watcher(app);

    let db_src = root.join(DB_ARCHIVE_DIR).join(DB_FILENAME);
    let app_for_db = app.clone();
    db.close_and_replace(|| {
        let app = app_for_db.clone();
        let db_src = db_src.clone();
        async move {
            replace_live_database(&app, &db_src)?;
            crate::db::init(&app).await
        }
    })
    .await?;

    let resources_src = root.join(RESOURCES_ARCHIVE_DIR);
    if resources_src.exists() {
        replace_dir(&resources_src, &crate::core::paths::resources_dir(app)?)?;
    }

    refresh_apps_registry(app).await;
    emit_clipboard_imported(app);

    let settings_path = root.join(CONFIG_ARCHIVE_DIR).join(SETTINGS_FILENAME);
    let next = app
        .state::<crate::settings::SettingsStore>()
        .replace_from_file(&settings_path)?;
    emit_settings_updated(app, &next);

    Ok(ImportHistoryBackupResult {
        strategy: BackupImportStrategy::Overwrite,
        imported_items: 0,
        skipped_items: 0,
        imported_resources: 0,
        imported_settings: true,
        requires_restart: false,
    })
}

struct MergeOutcome {
    imported_items: u64,
    skipped_items: u64,
}

async fn open_backup_db(path: &Path) -> Result<SqlitePool> {
    let options = SqliteConnectOptions::new()
        .filename(path)
        .read_only(true)
        .journal_mode(SqliteJournalMode::Wal)
        .foreign_keys(true);

    Ok(SqlitePoolOptions::new()
        .max_connections(1)
        .connect_with(options)
        .await
        .with_context(|| format!("failed to open backup database at {path:?}"))?)
}

async fn merge_history(current: &SqlitePool, backup: &SqlitePool) -> Result<MergeOutcome> {
    let mut tx = current.begin().await.context("failed to begin import")?;
    merge_groups(&mut tx, backup).await?;
    merge_apps(&mut tx, backup).await?;
    merge_file_type_icons(&mut tx, backup).await?;
    let outcome = merge_items(&mut tx, backup).await?;
    tx.commit().await.context("failed to commit import")?;

    Ok(outcome)
}

async fn merge_groups(tx: &mut sqlx::Transaction<'_, Sqlite>, backup: &SqlitePool) -> Result<()> {
    let rows = sqlx::query_as::<
        _,
        (
            String,
            String,
            String,
            bool,
            i64,
            DateTime<Utc>,
            DateTime<Utc>,
        ),
    >(
        "SELECT id, name, icon, is_hidden, sort_order, created_at, updated_at FROM clipboard_groups",
    )
    .fetch_all(backup)
    .await
    .context("failed to read backup groups")?;

    for row in rows {
        sqlx::query(
            "INSERT OR IGNORE INTO clipboard_groups \
             (id, name, icon, is_hidden, sort_order, created_at, updated_at) \
             VALUES (?, ?, ?, ?, ?, ?, ?)",
        )
        .bind(row.0)
        .bind(row.1)
        .bind(row.2)
        .bind(row.3)
        .bind(row.4)
        .bind(row.5)
        .bind(row.6)
        .execute(&mut **tx)
        .await
        .context("failed to import group")?;
    }

    Ok(())
}

async fn merge_apps(tx: &mut sqlx::Transaction<'_, Sqlite>, backup: &SqlitePool) -> Result<()> {
    let rows = sqlx::query_as::<
        _,
        (
            String,
            String,
            Option<String>,
            String,
            DateTime<Utc>,
            DateTime<Utc>,
        ),
    >(
        "SELECT id, name, icon_file, platform, created_at, updated_at FROM clipboard_apps"
    )
    .fetch_all(backup)
    .await
    .context("failed to read backup apps")?;

    for row in rows {
        sqlx::query(
            "INSERT OR IGNORE INTO clipboard_apps \
             (id, name, icon_file, platform, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
        )
        .bind(row.0)
        .bind(row.1)
        .bind(row.2)
        .bind(row.3)
        .bind(row.4)
        .bind(row.5)
        .execute(&mut **tx)
        .await
        .context("failed to import app")?;
    }

    Ok(())
}

async fn merge_file_type_icons(
    tx: &mut sqlx::Transaction<'_, Sqlite>,
    backup: &SqlitePool,
) -> Result<()> {
    let rows = sqlx::query_as::<_, (String, String, String, DateTime<Utc>, DateTime<Utc>)>(
        "SELECT cache_key, platform, icon_file, created_at, updated_at FROM file_type_icons",
    )
    .fetch_all(backup)
    .await
    .context("failed to read backup file type icons")?;

    for row in rows {
        sqlx::query(
            "INSERT OR IGNORE INTO file_type_icons \
             (cache_key, platform, icon_file, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
        )
        .bind(row.0)
        .bind(row.1)
        .bind(row.2)
        .bind(row.3)
        .bind(row.4)
        .execute(&mut **tx)
        .await
        .context("failed to import file type icon")?;
    }

    Ok(())
}

async fn merge_items(
    tx: &mut sqlx::Transaction<'_, Sqlite>,
    backup: &SqlitePool,
) -> Result<MergeOutcome> {
    let rows = sqlx::query_as::<_, BackupItemRow>(
        "SELECT id, kind, sub_kind, group_id, source_app_id, content, content_hash, search_text, \
         summary, file_types, size, width, height, use_count, is_favorite, is_pinned, platform, note, \
         created_at, updated_at FROM clipboard_items ORDER BY created_at ASC",
    )
    .fetch_all(backup)
    .await
    .context("failed to read backup items")?;

    let mut imported_items = 0;
    let mut skipped_items = 0;
    for row in rows {
        let exists: Option<i64> = sqlx::query_scalar(
            "SELECT 1 FROM clipboard_items WHERE kind = ? AND content_hash = ? LIMIT 1",
        )
        .bind(row.kind.as_str())
        .bind(row.content_hash.as_str())
        .fetch_optional(&mut **tx)
        .await
        .context("failed to check duplicate item")?;
        if exists.is_some() {
            skipped_items += 1;
            continue;
        }

        sqlx::query(
            "INSERT OR IGNORE INTO clipboard_items \
             (id, kind, sub_kind, group_id, source_app_id, content, content_hash, search_text, \
              summary, file_types, size, width, height, use_count, is_favorite, is_pinned, platform, note, \
              created_at, updated_at) \
             VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
        )
        .bind(row.id)
        .bind(row.kind)
        .bind(row.sub_kind)
        .bind(row.group_id)
        .bind(row.source_app_id)
        .bind(row.content)
        .bind(row.content_hash)
        .bind(row.search_text)
        .bind(row.summary)
        .bind(row.file_types)
        .bind(row.size)
        .bind(row.width)
        .bind(row.height)
        .bind(row.use_count)
        .bind(row.is_favorite)
        .bind(row.is_pinned)
        .bind(row.platform)
        .bind(row.note)
        .bind(row.created_at)
        .bind(row.updated_at)
        .execute(&mut **tx)
        .await
        .context("failed to import item")?;
        imported_items += 1;
    }

    Ok(MergeOutcome {
        imported_items,
        skipped_items,
    })
}

#[derive(sqlx::FromRow)]
struct BackupItemRow {
    id: String,
    kind: String,
    sub_kind: Option<String>,
    group_id: Option<String>,
    source_app_id: Option<String>,
    content: String,
    content_hash: String,
    search_text: Option<String>,
    summary: Option<String>,
    file_types: Option<String>,
    size: Option<i64>,
    width: Option<i64>,
    height: Option<i64>,
    use_count: i64,
    is_favorite: bool,
    is_pinned: bool,
    platform: String,
    note: Option<String>,
    created_at: DateTime<Utc>,
    updated_at: DateTime<Utc>,
}

struct WatcherPauseRestore {
    pause: Option<crate::clipboard::WatcherPause>,
    previous: bool,
}

impl Drop for WatcherPauseRestore {
    fn drop(&mut self) {
        if let Some(pause) = &self.pause {
            pause.set_paused(self.previous);
        }
    }
}

/// 暂停剪贴板监听，并在 guard drop 时恢复导入前的暂停状态。
fn pause_watcher(app: &AppHandle) -> WatcherPauseRestore {
    let pause = app.try_state::<crate::clipboard::WatcherPause>();
    let previous = pause.as_ref().is_some_and(|state| state.is_paused());
    if let Some(state) = pause.as_ref() {
        state.set_paused(true);
    }

    WatcherPauseRestore {
        pause: pause.map(|state| state.inner().clone()),
        previous,
    }
}

/// 用备份数据库替换当前主数据库，并移除旧 WAL / SHM sidecar。
fn replace_live_database(app: &AppHandle, src: &Path) -> Result<()> {
    let dst = crate::db::db_path(app)?;
    copy_file_to(src, &dst)?;

    for suffix in ["-wal", "-shm"] {
        let sidecar = PathBuf::from(format!("{}{suffix}", dst.display()));
        if sidecar.exists() {
            fs::remove_file(&sidecar).with_context(|| format!("failed to remove {sidecar:?}"))?;
        }
    }

    Ok(())
}

/// 覆盖导入后从新数据库重建来源应用内存缓存。
async fn refresh_apps_registry(app: &AppHandle) {
    let Some(registry) = app.try_state::<crate::clipboard::AppsRegistry>() else {
        return;
    };

    if let Err(err) = registry.load_from_db().await {
        log::warn!("refresh apps registry after backup overwrite failed: {err}");
    }
}

fn read_json_file(path: &Path) -> Result<serde_json::Value> {
    let content = fs::read_to_string(path).with_context(|| format!("failed to read {path:?}"))?;
    Ok(serde_json::from_str(&content).with_context(|| format!("failed to parse {path:?}"))?)
}

fn copy_file_to(src: &Path, dst: &Path) -> Result<()> {
    let parent = dst
        .parent()
        .ok_or_else(|| anyhow!("target path has no parent"))?;
    fs::create_dir_all(parent).with_context(|| format!("failed to create {parent:?}"))?;
    fs::copy(src, dst).with_context(|| format!("failed to copy {src:?} to {dst:?}"))?;

    Ok(())
}

fn copy_dir_all(src: &Path, dst: &Path) -> Result<u64> {
    if !src.exists() {
        return Ok(0);
    }

    let mut copied = 0;
    for entry in WalkDir::new(src)
        .into_iter()
        .filter_map(std::result::Result::ok)
    {
        let path = entry.path();
        let relative = path
            .strip_prefix(src)
            .with_context(|| format!("failed to strip {src:?} from {path:?}"))?;
        if relative.as_os_str().is_empty() {
            continue;
        }

        let target = dst.join(relative);
        if entry.file_type().is_dir() {
            fs::create_dir_all(&target)
                .with_context(|| format!("failed to create directory {target:?}"))?;
            continue;
        }

        copy_file_to(path, &target)?;
        copied += 1;
    }

    Ok(copied)
}

fn copy_dir_missing(src: &Path, dst: &Path) -> Result<u64> {
    if !src.exists() {
        return Ok(0);
    }

    let mut copied = 0;
    for entry in WalkDir::new(src)
        .into_iter()
        .filter_map(std::result::Result::ok)
    {
        let path = entry.path();
        if !entry.file_type().is_file() {
            continue;
        }

        let relative = path
            .strip_prefix(src)
            .with_context(|| format!("failed to strip {src:?} from {path:?}"))?;
        let target = dst.join(relative);
        if target.exists() {
            continue;
        }

        copy_file_to(path, &target)?;
        copied += 1;
    }

    Ok(copied)
}

fn replace_dir(src: &Path, dst: &Path) -> Result<()> {
    if dst.exists() {
        fs::remove_dir_all(dst).with_context(|| format!("failed to remove {dst:?}"))?;
    }
    copy_dir_all(src, dst)?;

    Ok(())
}

fn emit_clipboard_imported(app: &AppHandle) {
    if let Err(err) = app.emit(
        "clipboard://updated",
        serde_json::json!({
            "id": null,
            "deduplicated": false,
            "imported": true,
        }),
    ) {
        log::warn!("emit clipboard import update failed: {err}");
    }
}

fn emit_settings_updated(app: &AppHandle, settings: &crate::settings::Settings) {
    if let Err(err) = app.emit("settings://updated", settings) {
        log::warn!("emit settings import update failed: {err}");
    }
}

fn copy_file_into_writer<W: Write>(path: &Path, writer: &mut W) -> Result<()> {
    let file = File::open(path).with_context(|| format!("failed to open {path:?}"))?;
    let mut reader = BufReader::new(file);
    std::io::copy(&mut reader, writer).with_context(|| format!("failed to copy {path:?}"))?;

    Ok(())
}

fn dir_size(path: &Path) -> Result<u64> {
    if !path.exists() {
        return Ok(0);
    }

    let mut total = 0;
    for entry in WalkDir::new(path)
        .into_iter()
        .filter_map(std::result::Result::ok)
    {
        let path = entry.path();
        let metadata = entry
            .metadata()
            .with_context(|| format!("failed to read metadata at {path:?}"))?;

        if metadata.is_file() {
            total += metadata.len();
        }
    }

    Ok(total)
}

fn app_error<T>(message: impl Into<String>) -> Result<T> {
    Err(AppError::Other(anyhow!(message.into())))
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Cursor;
    use tempfile::tempdir;
    use zip::ZipArchive;

    #[test]
    fn backup_extension_validation_accepts_expected_suffix() {
        let path = normalize_backup_path(PathBuf::from("demo.ecopastebak")).unwrap();
        assert_eq!(path, PathBuf::from("demo.ecopastebak"));
    }

    #[test]
    fn backup_extension_validation_appends_missing_suffix() {
        let path = normalize_backup_path(PathBuf::from("demo")).unwrap();
        assert_eq!(path, PathBuf::from("demo.ecopastebak"));
    }

    #[test]
    fn backup_extension_validation_rejects_other_suffix() {
        assert!(normalize_backup_path(PathBuf::from("demo.zip")).is_err());
    }

    #[test]
    fn encrypted_payload_header_round_trips() {
        let encrypted = encrypt_payload(b"hello", "password-123").unwrap();
        let mut bytes = Vec::new();
        write_header(&mut bytes, &encrypted.header).unwrap();
        bytes.extend_from_slice(&encrypted.ciphertext);

        let mut cursor = Cursor::new(bytes);

        assert_eq!(
            inspect_backup_reader(&mut cursor).unwrap(),
            BackupContainerMode::Encrypted
        );
    }

    #[test]
    fn inspect_backup_reader_recognizes_plain_zip() {
        let mut cursor = Cursor::new(b"PK\x03\x04demo".to_vec());

        assert_eq!(
            inspect_backup_reader(&mut cursor).unwrap(),
            BackupContainerMode::Plain
        );
    }

    #[test]
    fn inspect_backup_reader_recognizes_encrypted_container() {
        let encrypted = encrypt_payload(b"hello", "password-123").unwrap();
        let mut bytes = Vec::new();
        write_header(&mut bytes, &encrypted.header).unwrap();
        bytes.extend_from_slice(&encrypted.ciphertext);
        let mut cursor = Cursor::new(bytes);

        assert_eq!(
            inspect_backup_reader(&mut cursor).unwrap(),
            BackupContainerMode::Encrypted
        );
    }

    #[test]
    fn payload_zip_contains_only_backup_whitelist() {
        let temp = tempdir().unwrap();
        let root = temp.path();
        let resources = root.join("resources");
        fs::create_dir_all(resources.join("clipboard-images/origin")).unwrap();
        fs::write(root.join("clipboard.db"), b"db").unwrap();
        fs::write(root.join("settings.json"), b"{}").unwrap();
        fs::write(root.join("window-state.json"), b"skip").unwrap();
        fs::write(root.join("settings.json.bak"), b"skip").unwrap();
        fs::write(resources.join("clipboard-images/origin/demo.png"), b"image").unwrap();

        let payload = root.join("payload.zip");
        let target = root.join("backup.ecopastebak");
        let manifest = BackupManifest {
            format_version: FORMAT_VERSION,
            app_name: "EcoPaste".to_owned(),
            app_version: "0.0.0".to_owned(),
            exported_at: Utc::now(),
            platform: "macos".to_owned(),
            encryption: ManifestEncryption::None,
            item_count: 1,
            text_count: 1,
            image_count: 0,
            files_count: 0,
            resource_bytes: 5,
        };
        let source_paths = BackupSourcePaths {
            db_path: root.join("clipboard.db"),
            resources_dir: resources,
            settings_path: root.join("settings.json"),
        };

        write_payload_zip(&source_paths, &payload, &manifest, &target).unwrap();

        let file = File::open(payload).unwrap();
        let mut archive = ZipArchive::new(file).unwrap();
        let mut names = Vec::new();
        for index in 0..archive.len() {
            let file = archive.by_index(index).unwrap();
            names.push(file.name().to_owned());
        }
        names.sort();

        assert_eq!(
            names,
            vec![
                "config/settings.json",
                "db/clipboard.db",
                "manifest.json",
                "resources/clipboard-images/origin/demo.png",
            ]
        );
    }
}

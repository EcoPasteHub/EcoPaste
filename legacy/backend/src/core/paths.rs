//! app data 目录解析的单一入口。
//!
//! 所有持久化位置都从这里取根，集中三件本来散落各模块的事：
//! - `app_local_data_dir().context(...)` 这段解析样板（原先在 settings/window/db/storage/app_store 各写一份）；
//! - `db/`、`resources/`、`config/`、`state/` 这些语义目录名；
//! - 开发 / 生产环境的数据隔离：dev 数据落 `dev/`、release 数据落 `prod/`，互不污染，
//!   便于后续导入导出 / 备份 / 迁移按环境整目录操作。
//! - 自定义数据目录：`<app_local_data>/<env>/storage.json` 始终作为启动锚点，
//!   真实数据根由该 bootstrap manifest 指向。
//!
//! 只解析路径、不建目录：创建行为是各调用方特化的（settings/window/db 建自己的目录、
//! 图片/图标写时懒建），塞进这里会改掉懒建语义。叶子文件名（`clipboard.db` /
//! `settings.json` 等）仍由各模块自己拥有——要统一的是「根在哪」。

use std::fs;
use std::path::{Path, PathBuf};

use anyhow::Context;
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use tauri::{AppHandle, Manager};

use crate::core::Result;

/// SQLite 主库与 WAL / SHM sidecar 所在目录名，挂在 [`app_data_dir`] 下。
const DB_DIR: &str = "db";
/// 资源文件（图片、应用图标）的公共父目录名，挂在 [`app_data_dir`] 下。
const RESOURCES_DIR: &str = "resources";
/// 用户配置目录名，挂在 [`app_data_dir`] 下。
const CONFIG_DIR: &str = "config";
/// 本机运行状态目录名，挂在 [`app_data_dir`] 下。
const STATE_DIR: &str = "state";
/// 固定留在 `<app_local_data>/<env>` 的 bootstrap manifest 文件名。
const STORAGE_MANIFEST_FILENAME: &str = "storage.json";
/// 写入真实数据根的 identity manifest 文件名，用于识别 EcoPaste 数据目录。
const STORAGE_IDENTITY_FILENAME: &str = ".ecopaste-storage.json";
/// 用户选择父目录后创建的数据子目录名。
const CUSTOM_DATA_DIR_NAME: &str = "EcoPasteData";
/// 存储 manifest 格式版本。
const STORAGE_MANIFEST_VERSION: u16 = 1;

/// 开发环境数据子目录名（`tauri dev`）。
const DEV_ENV_DIR: &str = "dev";
/// 生产环境数据子目录名（`tauri build`）。
const PROD_ENV_DIR: &str = "prod";

/// 当前环境的数据子目录名：dev 构建走 `dev/`，release 构建走 `prod/`。
/// `cfg!(dev)` 是 Tauri 在 `tauri dev` 时注入的 cfg，与各模块原先选文件名后缀用的判定同源。
const fn env_dir() -> &'static str {
    if cfg!(dev) {
        DEV_ENV_DIR
    } else {
        PROD_ENV_DIR
    }
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct StorageLocation {
    pub current_path: String,
    pub default_path: String,
    pub is_custom: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
struct StorageManifest {
    version: u16,
    environment: String,
    data_dir: PathBuf,
    updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
struct StorageIdentity {
    version: u16,
    environment: String,
    created_at: DateTime<Utc>,
}

/// `<app_local_data>/<env>`：固定启动锚点。自定义数据目录启用后，这里仍保留
/// `storage.json` 用于解析真实数据根。
pub fn bootstrap_dir(app: &AppHandle) -> Result<PathBuf> {
    let dir = app
        .path()
        .app_local_data_dir()
        .context("failed to resolve app local data dir")?;
    Ok(dir.join(env_dir()))
}

/// 默认数据根。未启用自定义数据目录时，真实数据仍落在 `<app_local_data>/<env>`。
pub fn default_data_dir(app: &AppHandle) -> Result<PathBuf> {
    bootstrap_dir(app)
}

/// 用户选择父目录后创建 EcoPaste 自有数据子目录。
pub fn custom_data_dir(parent: &Path) -> PathBuf {
    parent.join(CUSTOM_DATA_DIR_NAME).join(env_dir())
}

/// 返回当前真实数据根、默认数据根，以及是否处于自定义目录。
pub fn storage_location(app: &AppHandle) -> Result<StorageLocation> {
    let current = app_data_dir(app)?;
    let default = default_data_dir(app)?;
    Ok(StorageLocation {
        is_custom: current != default,
        current_path: current.to_string_lossy().into_owned(),
        default_path: default.to_string_lossy().into_owned(),
    })
}

/// `<data_root>`：当前环境所有持久化位置的根；其下按语义拆分为 db、resources、
/// config 与 state。真实根由 bootstrap manifest 决定。
pub fn app_data_dir(app: &AppHandle) -> Result<PathBuf> {
    let bootstrap = bootstrap_dir(app)?;
    let default = default_data_dir(app)?;
    let manifest_path = storage_manifest_path(&bootstrap);

    fs::create_dir_all(&bootstrap)
        .with_context(|| format!("failed to create bootstrap dir at {bootstrap:?}"))?;

    let manifest = match read_storage_manifest(&manifest_path) {
        Ok(Some(manifest)) => manifest,
        Ok(None) => {
            let manifest = storage_manifest(default.clone());
            write_storage_manifest(&manifest_path, &manifest)?;
            manifest
        }
        Err(err) => {
            log::warn!("storage manifest unreadable, using default data dir: {err}");
            let manifest = storage_manifest(default.clone());
            write_storage_manifest(&manifest_path, &manifest)?;
            manifest
        }
    };

    if manifest.environment != env_dir() || manifest.version != STORAGE_MANIFEST_VERSION {
        log::warn!("storage manifest metadata mismatch, using default data dir");
        let manifest = storage_manifest(default.clone());
        write_storage_manifest(&manifest_path, &manifest)?;
        return Ok(default);
    }

    if !manifest.data_dir.exists() {
        log::warn!(
            "storage data dir {:?} is missing, falling back to default data dir",
            manifest.data_dir
        );
        let manifest = storage_manifest(default.clone());
        write_storage_manifest(&manifest_path, &manifest)?;
        return Ok(default);
    }

    if manifest.data_dir != default {
        match has_valid_storage_identity(&manifest.data_dir) {
            Ok(true) => {}
            Ok(false) => {
                log::warn!(
                    "storage data dir {:?} identity mismatch, falling back to default data dir",
                    manifest.data_dir
                );
                let manifest = storage_manifest(default.clone());
                write_storage_manifest(&manifest_path, &manifest)?;
                return Ok(default);
            }
            Err(err) => {
                log::warn!(
                    "storage data dir {:?} identity unreadable, falling back to default data dir: {err}",
                    manifest.data_dir
                );
                let manifest = storage_manifest(default.clone());
                write_storage_manifest(&manifest_path, &manifest)?;
                return Ok(default);
            }
        }
    }

    Ok(manifest.data_dir)
}

/// 将当前真实数据根切换到指定目录。调用方负责在写入前完成数据迁移。
pub fn set_app_data_dir(app: &AppHandle, data_dir: PathBuf) -> Result<()> {
    let bootstrap = bootstrap_dir(app)?;
    fs::create_dir_all(&bootstrap)
        .with_context(|| format!("failed to create bootstrap dir at {bootstrap:?}"))?;

    write_storage_identity(&data_dir)?;
    write_storage_manifest(
        &storage_manifest_path(&bootstrap),
        &storage_manifest(data_dir),
    )
}

/// 写入真实数据根 identity manifest，供迁移目标目录校验。
pub fn write_storage_identity(data_dir: &Path) -> Result<()> {
    fs::create_dir_all(data_dir)
        .with_context(|| format!("failed to create storage dir at {data_dir:?}"))?;
    let identity = StorageIdentity {
        version: STORAGE_MANIFEST_VERSION,
        environment: env_dir().to_owned(),
        created_at: Utc::now(),
    };
    let path = data_dir.join(STORAGE_IDENTITY_FILENAME);
    let json =
        serde_json::to_string_pretty(&identity).context("failed to serialize storage identity")?;

    fs::write(&path, json).with_context(|| format!("failed to write storage identity {path:?}"))?;
    Ok(())
}

/// 校验目标目录是否可作为 EcoPaste 数据根：空目录允许使用，已有 identity 时必须匹配。
pub fn validate_storage_target(data_dir: &Path) -> Result<()> {
    let identity_path = data_dir.join(STORAGE_IDENTITY_FILENAME);
    if identity_path.exists() {
        let content = fs::read_to_string(&identity_path)
            .with_context(|| format!("failed to read storage identity {identity_path:?}"))?;
        let identity: StorageIdentity =
            serde_json::from_str(&content).context("failed to parse storage identity")?;
        if identity.version == STORAGE_MANIFEST_VERSION && identity.environment == env_dir() {
            return Ok(());
        }

        return Err(anyhow::anyhow!("目标目录不是当前环境的 EcoPaste 数据目录").into());
    }

    if data_dir.exists()
        && fs::read_dir(data_dir)
            .with_context(|| format!("failed to read storage target {data_dir:?}"))?
            .next()
            .is_some()
    {
        return Err(anyhow::anyhow!("目标 EcoPaste 数据目录已存在且不是有效数据目录").into());
    }

    Ok(())
}

/// `<app_data_dir>/db`：SQLite 主库与 sidecar 的目录。
pub fn db_dir(app: &AppHandle) -> Result<PathBuf> {
    Ok(app_data_dir(app)?.join(DB_DIR))
}

/// `<app_data_dir>/resources`：图片、应用图标等资源文件的公共父目录。
pub fn resources_dir(app: &AppHandle) -> Result<PathBuf> {
    Ok(app_data_dir(app)?.join(RESOURCES_DIR))
}

/// `<app_data_dir>/config`：用户偏好配置目录。
pub fn config_dir(app: &AppHandle) -> Result<PathBuf> {
    Ok(app_data_dir(app)?.join(CONFIG_DIR))
}

/// `<app_data_dir>/state`：窗口位置等本机运行状态目录。
pub fn state_dir(app: &AppHandle) -> Result<PathBuf> {
    Ok(app_data_dir(app)?.join(STATE_DIR))
}

fn storage_manifest_path(bootstrap: &Path) -> PathBuf {
    bootstrap.join(STORAGE_MANIFEST_FILENAME)
}

fn storage_manifest(data_dir: PathBuf) -> StorageManifest {
    StorageManifest {
        version: STORAGE_MANIFEST_VERSION,
        environment: env_dir().to_owned(),
        data_dir,
        updated_at: Utc::now(),
    }
}

fn read_storage_manifest(path: &Path) -> Result<Option<StorageManifest>> {
    if !path.exists() {
        return Ok(None);
    }

    let content = fs::read_to_string(path).with_context(|| format!("failed to read {path:?}"))?;
    Ok(Some(
        serde_json::from_str(&content).with_context(|| format!("failed to parse {path:?}"))?,
    ))
}

fn write_storage_manifest(path: &Path, manifest: &StorageManifest) -> Result<()> {
    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent)
            .with_context(|| format!("failed to create manifest dir at {parent:?}"))?;
    }

    let json =
        serde_json::to_string_pretty(manifest).context("failed to serialize storage manifest")?;
    let tmp = path.with_extension("json.tmp");
    fs::write(&tmp, json).with_context(|| format!("failed to write {tmp:?}"))?;
    fs::rename(&tmp, path).with_context(|| format!("failed to promote {tmp:?} to {path:?}"))?;
    Ok(())
}

fn has_valid_storage_identity(data_dir: &Path) -> Result<bool> {
    let identity_path = data_dir.join(STORAGE_IDENTITY_FILENAME);
    if !identity_path.exists() {
        return Ok(false);
    }

    let content = fs::read_to_string(&identity_path)
        .with_context(|| format!("failed to read storage identity {identity_path:?}"))?;
    let identity: StorageIdentity =
        serde_json::from_str(&content).context("failed to parse storage identity")?;
    Ok(identity.version == STORAGE_MANIFEST_VERSION && identity.environment == env_dir())
}

#[cfg(test)]
mod tests {
    use super::*;

    struct TempDir(PathBuf);

    impl TempDir {
        fn new() -> Self {
            let path = std::env::temp_dir()
                .join(format!("ecopaste-storage-paths-{}", uuid::Uuid::new_v4()));
            fs::create_dir_all(&path).unwrap();

            Self(path)
        }

        fn path(&self) -> &Path {
            &self.0
        }
    }

    impl Drop for TempDir {
        fn drop(&mut self) {
            fs::remove_dir_all(&self.0).ok();
        }
    }

    #[test]
    fn empty_storage_target_is_allowed() {
        let temp = TempDir::new();

        validate_storage_target(temp.path()).unwrap();
    }

    #[test]
    fn custom_data_dir_uses_named_data_container() {
        let temp = TempDir::new();

        assert_eq!(
            custom_data_dir(temp.path()),
            temp.path().join("EcoPasteData").join(env_dir())
        );
    }

    #[test]
    fn storage_target_with_matching_identity_is_allowed() {
        let temp = TempDir::new();

        write_storage_identity(temp.path()).unwrap();

        validate_storage_target(temp.path()).unwrap();
    }

    #[test]
    fn storage_target_with_mismatched_identity_is_rejected() {
        let temp = TempDir::new();
        let identity = StorageIdentity {
            version: STORAGE_MANIFEST_VERSION,
            environment: "other".to_owned(),
            created_at: Utc::now(),
        };
        fs::write(
            temp.path().join(STORAGE_IDENTITY_FILENAME),
            serde_json::to_string_pretty(&identity).unwrap(),
        )
        .unwrap();

        assert!(validate_storage_target(temp.path()).is_err());
    }

    #[test]
    fn non_empty_storage_target_without_identity_is_rejected() {
        let temp = TempDir::new();
        fs::write(temp.path().join("random.txt"), "not EcoPaste").unwrap();

        assert!(validate_storage_target(temp.path()).is_err());
    }
}

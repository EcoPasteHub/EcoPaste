//! app data 目录解析的单一入口。
//!
//! 所有持久化位置都从这里取根，集中三件本来散落各模块的事：
//! - `app_local_data_dir().context(...)` 这段解析样板（原先在 settings/window/db/storage/app_store 各写一份）；
//! - `db/`、`resources/`、`config/`、`state/` 这些语义目录名；
//! - 开发 / 生产环境的数据隔离：dev 数据落 `dev/`、release 数据落 `prod/`，互不污染，
//!   便于后续导入导出 / 备份 / 迁移按环境整目录操作。
//!
//! 只解析路径、不建目录：创建行为是各调用方特化的（settings/window/db 建自己的目录、
//! 图片/图标写时懒建），塞进这里会改掉懒建语义。叶子文件名（`clipboard.db` /
//! `settings.json` 等）仍由各模块自己拥有——要统一的是「根在哪」。

use std::path::PathBuf;

use anyhow::Context;
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

/// `<app_local_data>/<env>`：当前环境所有持久化位置的根；其下按语义拆分为 db、resources、
/// config 与 state。`<env>` 为 `dev` 或 `prod`，使两套环境的数据在文件系统上完全隔离。
pub fn app_data_dir(app: &AppHandle) -> Result<PathBuf> {
    let dir = app
        .path()
        .app_local_data_dir()
        .context("failed to resolve app local data dir")?;
    Ok(dir.join(env_dir()))
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

//! 设置持久化。
//!
//! - 落盘位置：`<app_data_dir>/settings.json`（dev/prod 由 `core::paths` 的环境子目录隔离，文件名不带后缀）。
//! - 写入流程：先把当前盘内容备份到 `settings.json.bak`，再原子写入新文件。
//!   读取时主文件解析失败回退到 `.bak`，再失败回退到 `Default`，避免一次坏盘把所有偏好弄丢。
//! - 缺字段兼容：`Settings` 各结构体都 `#[serde(default)]`，新版本新增字段不影响旧文件。

use std::fs;
use std::io::Write;
use std::path::{Path, PathBuf};
use std::sync::RwLock;

use anyhow::Context;
use tauri::AppHandle;

use crate::core::{AppError, Result};

use super::model::{Language, Settings};

const FILENAME: &str = "settings.json";
const BACKUP_SUFFIX: &str = ".bak";

pub struct SettingsStore {
    path: PathBuf,
    backup_path: PathBuf,
    current: RwLock<Settings>,
}

impl SettingsStore {
    pub fn new(app: &AppHandle) -> Result<Self> {
        let dir = crate::core::paths::app_data_dir(app)?;
        fs::create_dir_all(&dir).with_context(|| format!("failed to create dir at {dir:?}"))?;

        let path = dir.join(FILENAME);
        let backup_path = path.with_extension(format!(
            "{}{}",
            path.extension().and_then(|s| s.to_str()).unwrap_or("json"),
            BACKUP_SUFFIX
        ));

        let current = match load_from_disk(&path, &backup_path) {
            Some(settings) => settings,
            None => {
                // 真·首次启动（主文件 + 备份都不存在）：用系统 locale 推导默认语言并落盘，
                // 之后所有读取都走常规分支，避免每次启动都重算。
                let settings = default_settings_with_system_locale();
                if let Err(err) = write_atomic(&path, &backup_path, &settings) {
                    log::warn!("persist first-run settings failed: {err}");
                }
                settings
            }
        };
        log::info!("settings store ready at {path:?}");

        Ok(Self {
            path,
            backup_path,
            current: RwLock::new(current),
        })
    }

    pub fn snapshot(&self) -> Settings {
        self.current.read().expect("settings poisoned").clone()
    }

    /// 恢复默认设置并落盘，返回新的完整快照。
    pub fn reset(&self) -> Result<Settings> {
        let next = default_settings_with_system_locale();

        write_atomic(&self.path, &self.backup_path, &next)?;
        *self.current.write().expect("settings poisoned") = next.clone();
        Ok(next)
    }

    /// 用 JSON patch 深度合并到当前设置，落盘后返回新快照。
    /// patch 必须是 object；非 object 视为「整个替换」语义不友好，直接报错。
    pub fn update(&self, patch: serde_json::Value) -> Result<Settings> {
        if !patch.is_object() {
            return Err(AppError::Other(anyhow::anyhow!(
                "settings patch must be a JSON object"
            )));
        }

        let mut guard = self.current.write().expect("settings poisoned");

        let mut merged = serde_json::to_value(&*guard)
            .context("failed to serialize current settings for merge")?;
        deep_merge(&mut merged, patch);

        let next: Settings = serde_json::from_value(merged)
            .map_err(|err| AppError::Other(anyhow::anyhow!("invalid settings patch: {err}")))?;

        write_atomic(&self.path, &self.backup_path, &next)?;
        *guard = next.clone();
        Ok(next)
    }
}

/// 生成默认设置，并沿用首次启动的系统语言推导规则。
fn default_settings_with_system_locale() -> Settings {
    let mut settings = Settings::default();
    if let Some(tag) = tauri_plugin_os::locale() {
        settings.appearance.language = Language::from_system_locale(&tag);
        log::info!(
            "default settings language from locale {tag}: {:?}",
            settings.appearance.language
        );
    }
    settings
}

/// 返回 `None` 表示主文件与备份都不存在（首次启动），调用方据此走「初始化默认」分支；
/// 读取过程中遇到 IO/解析错误会逐个回退并打 warn，最终仍找不到可用文件时也返回 `Settings::default()` 包装在 `Some` 里——
/// 这条路径表示「文件存在但坏了」，不要当成首次启动覆盖系统 locale。
fn load_from_disk(path: &Path, backup: &Path) -> Option<Settings> {
    if !path.exists() && !backup.exists() {
        return None;
    }
    for candidate in [path, backup] {
        if !candidate.exists() {
            continue;
        }
        match fs::read_to_string(candidate).and_then(|content| {
            serde_json::from_str::<Settings>(&content)
                .map_err(|err| std::io::Error::new(std::io::ErrorKind::InvalidData, err))
        }) {
            Ok(settings) => return Some(settings),
            Err(err) => {
                log::warn!("settings file {candidate:?} unreadable, falling back: {err}");
            }
        }
    }
    Some(Settings::default())
}

/// 写入策略：先把现盘文件 rename 成 `.bak`（覆盖旧备份），再把新内容写到 tmp 后 rename 成主文件。
/// rename 在同一文件系统下是原子的，避免中途断电留下半截 JSON。
fn write_atomic(path: &Path, backup: &Path, settings: &Settings) -> Result<()> {
    let json = serde_json::to_string_pretty(settings).context("failed to serialize settings")?;

    if path.exists() {
        if let Err(err) = fs::rename(path, backup) {
            log::warn!("backup current settings to {backup:?} failed: {err}");
        }
    }

    let tmp = path.with_extension("json.tmp");
    {
        let mut file = fs::File::create(&tmp)
            .with_context(|| format!("failed to create tmp settings at {tmp:?}"))?;
        file.write_all(json.as_bytes())
            .with_context(|| format!("failed to write tmp settings at {tmp:?}"))?;
        file.sync_all().ok();
    }
    fs::rename(&tmp, path)
        .with_context(|| format!("failed to promote tmp settings to {path:?}"))?;
    Ok(())
}

fn deep_merge(base: &mut serde_json::Value, patch: serde_json::Value) {
    match (base, patch) {
        (serde_json::Value::Object(base_map), serde_json::Value::Object(patch_map)) => {
            for (k, v) in patch_map {
                match base_map.get_mut(&k) {
                    Some(existing) if existing.is_object() && v.is_object() => {
                        deep_merge(existing, v);
                    }
                    _ => {
                        base_map.insert(k, v);
                    }
                }
            }
        }
        (slot, patch) => {
            *slot = patch;
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn deep_merge_overrides_leaves_and_recurses_objects() {
        let mut base = serde_json::json!({
            "general": {"autoStart": false, "trayIcon": true},
            "clipboard": {"history": {"maxCount": 0}},
        });
        let patch = serde_json::json!({
            "general": {"autoStart": true},
            "clipboard": {"history": {"maxCount": 500}},
        });
        deep_merge(&mut base, patch);
        assert_eq!(
            base,
            serde_json::json!({
                "general": {"autoStart": true, "trayIcon": true},
                "clipboard": {"history": {"maxCount": 500}},
            })
        );
    }

    #[test]
    fn deep_merge_replaces_arrays_wholesale() {
        let mut base = serde_json::json!({"itemActions": ["copy", "star", "delete"]});
        let patch = serde_json::json!({"itemActions": ["copy", "pastePlain"]});
        deep_merge(&mut base, patch);
        assert_eq!(
            base,
            serde_json::json!({"itemActions": ["copy", "pastePlain"]})
        );
    }

    #[test]
    fn missing_fields_fall_back_to_defaults() {
        let partial = r#"{"general": {"autoStart": true}}"#;
        let parsed: Settings = serde_json::from_str(partial).unwrap();
        assert!(parsed.general.auto_start);
        assert!(parsed.general.tray_icon, "default kept");
        assert_eq!(parsed.shortcuts.open_clipboard, "Alt+C");
        assert_eq!(
            parsed.update.frequency,
            crate::settings::UpdateFrequency::Weekly
        );
        assert_eq!(
            parsed.clipboard.content.sort,
            crate::db::models::ClipboardItemSort::UpdatedAt
        );
        assert!(!parsed.clipboard.content.update_on_reuse);
        assert_eq!(parsed.clipboard.history.cleanup_interval_hours, 0);
    }
}

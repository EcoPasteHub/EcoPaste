use tauri::{AppHandle, Emitter, Manager};

use crate::core::Result;
use crate::settings::{Settings, SettingsStore};
use crate::{autostart, shortcut, tray, window};

/// 与前端 `src/constants/events.ts` 的 `TAURI_EVENT.SETTINGS_UPDATED` 一一对应。
const SETTINGS_UPDATED_EVENT: &str = "settings://updated";

#[tauri::command]
pub async fn get_settings(app: AppHandle) -> Result<Settings> {
    Ok(app.state::<SettingsStore>().snapshot())
}

/// 用 JSON patch 深度合并到现设置。返回合并后的完整快照。
/// 若 `shortcuts` 段被改动，会顺带重注册全局快捷键；
/// 若 `general.trayIcon` 或 `appearance.language` 被改动，重建托盘菜单/显隐。
#[tauri::command]
pub async fn update_settings(app: AppHandle, patch: serde_json::Value) -> Result<Settings> {
    let patch_obj = patch.as_object();
    let touches_shortcuts = patch_obj
        .map(|m| m.contains_key("shortcuts"))
        .unwrap_or(false);
    let touches_tray = patch_obj
        .map(|m| {
            m.get("general")
                .and_then(|v| v.as_object())
                .is_some_and(|g| g.contains_key("trayIcon"))
                || m.get("appearance")
                    .and_then(|v| v.as_object())
                    .is_some_and(|a| a.contains_key("language"))
        })
        .unwrap_or(false);

    let next = app.state::<SettingsStore>().update(patch)?;

    if touches_shortcuts {
        if let Err(err) = shortcut::apply(&app, &next.shortcuts) {
            log::warn!("re-apply shortcuts after settings update failed: {err}");
        }
    }

    if touches_tray {
        if let Err(err) = tray::apply(&app, &next) {
            log::warn!("re-apply tray after settings update failed: {err}");
        }
    }

    emit_settings_updated(&app, &next);

    Ok(next)
}

/// 恢复所有设置默认值，保留历史记录与资源文件。
#[tauri::command]
pub async fn reset_settings(app: AppHandle) -> Result<Settings> {
    let next = app.state::<SettingsStore>().reset()?;

    apply_reset_side_effects(&app, &next);
    emit_settings_updated(&app, &next);

    Ok(next)
}

/// 重置后按默认设置同步系统级副作用，失败只记日志，不回滚已落盘设置。
fn apply_reset_side_effects(app: &AppHandle, settings: &Settings) {
    if let Err(err) = autostart::set_enabled(app, settings.general.auto_start) {
        log::warn!("reset autostart failed: {err}");
    }

    if let Err(err) = shortcut::apply(app, &settings.shortcuts) {
        log::warn!("reset shortcuts failed: {err}");
    }

    if let Err(err) = tray::apply(app, settings) {
        log::warn!("reset tray failed: {err}");
    }

    if let Err(err) = window::show_taskbar_icon(app, settings.general.dock_icon) {
        log::warn!("reset taskbar icon failed: {err}");
    }
}

/// 广播最新设置快照给所有前端窗口。
fn emit_settings_updated(app: &AppHandle, settings: &Settings) {
    if let Err(err) = app.emit(SETTINGS_UPDATED_EVENT, settings) {
        log::warn!("emit settings updated event failed: {err}");
    }
}

use serde_json::json;
use tauri::{AppHandle, Manager};

use crate::admin::{self, AdminLaunchStatus};
use crate::core::Result;
use crate::settings::{Settings, SettingsStore};

#[tauri::command]
pub async fn get_run_as_admin_status(app: AppHandle) -> Result<AdminLaunchStatus> {
    let configured = app.state::<SettingsStore>().snapshot().general.run_as_admin;

    Ok(admin::status(configured))
}

#[tauri::command]
pub async fn set_run_as_admin(app: AppHandle, enabled: bool) -> Result<Settings> {
    let next = app.state::<SettingsStore>().update(json!({
        "general": {
            "runAsAdmin": enabled,
        },
    }))?;

    admin::sync_scheduled_task(enabled);
    super::settings::emit_settings_updated(&app, &next);

    Ok(next)
}

#[tauri::command]
pub async fn restart_as_admin(app: AppHandle) -> Result<()> {
    admin::launch_elevated_current_process()?;
    app.exit(0);

    Ok(())
}

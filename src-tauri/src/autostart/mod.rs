//! 自启动：直接用 `auto-launch` crate 实现，绕过 `tauri-plugin-autostart` 上游 bug
//! （tauri-apps/plugins-workspace#1922：macOS 下 `is_enabled` 误报、`enable` 路径写错）。
//!
//! 启动参数固定追加 `--auto-launch`，让前端在「静默启动」场景里识别本次启动来源。

use std::env;

use auto_launch::{AutoLaunch, AutoLaunchBuilder};
use tauri::{AppHandle, Manager};

use crate::core::{AppError, Result};

const AUTO_LAUNCH_ARG: &str = "--auto-launch";

pub struct AutostartManager {
    inner: AutoLaunch,
}

pub fn init(app: &AppHandle) -> Result<()> {
    let exe = env::current_exe()
        .map_err(|err| AppError::Other(anyhow::anyhow!("current_exe failed: {err}")))?;
    let exe_path = exe.to_string_lossy().to_string();

    let app_name = app.package_info().name.clone();

    let inner = AutoLaunchBuilder::new()
        .set_app_name(&app_name)
        .set_app_path(&exe_path)
        .set_args(&[AUTO_LAUNCH_ARG])
        .build()
        .map_err(|err| AppError::Other(anyhow::anyhow!("build AutoLaunch failed: {err}")))?;

    app.manage(AutostartManager { inner });
    Ok(())
}

pub fn is_enabled(app: &AppHandle) -> Result<bool> {
    let manager = app.state::<AutostartManager>();
    manager
        .inner
        .is_enabled()
        .map_err(|err| AppError::Other(anyhow::anyhow!("is_enabled failed: {err}")))
}

pub fn set_enabled(app: &AppHandle, enabled: bool) -> Result<()> {
    let manager = app.state::<AutostartManager>();
    if enabled {
        manager
            .inner
            .enable()
            .map_err(|err| AppError::Other(anyhow::anyhow!("enable autostart failed: {err}")))
    } else {
        manager
            .inner
            .disable()
            .map_err(|err| AppError::Other(anyhow::anyhow!("disable autostart failed: {err}")))
    }
}

pub fn launched_via_autostart() -> bool {
    env::args().any(|arg| arg == AUTO_LAUNCH_ARG)
}

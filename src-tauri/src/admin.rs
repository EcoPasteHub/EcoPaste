//! Windows administrator launch support.
//!
//! The persistent setting records intent. The current process token remains the
//! source of truth for whether the app is actually elevated.

#[cfg(target_os = "windows")]
use std::path::{Path, PathBuf};

#[cfg(target_os = "windows")]
use anyhow::Context;
#[cfg(target_os = "windows")]
use serde::Deserialize;

#[cfg(target_os = "windows")]
use crate::core::windows_args;
use crate::core::{AppError, Result};

#[cfg(target_os = "windows")]
const ADMIN_RESTARTED_ARG: &str = "--ecopaste-admin-restarted";
#[cfg(target_os = "windows")]
const TASK_NAME: &str = "EcoPasteAdmin";
#[cfg(target_os = "windows")]
const CREATE_NO_WINDOW: u32 = 0x08000000;
#[cfg(target_os = "windows")]
const SETTINGS_FILENAME: &str = "settings.json";
#[cfg(target_os = "windows")]
const STORAGE_MANIFEST_FILENAME: &str = "storage.json";
#[cfg(target_os = "windows")]
const DEV_ENV_DIR: &str = "dev";
#[cfg(target_os = "windows")]
const PROD_ENV_DIR: &str = "prod";

#[derive(Debug, Clone, serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub struct AdminLaunchStatus {
    pub configured: bool,
    pub running_as_admin: bool,
    pub task_ready: bool,
}

#[cfg(target_os = "windows")]
#[derive(Debug, Default, Deserialize)]
#[serde(default, rename_all = "camelCase")]
struct EarlySettings {
    general: EarlyGeneral,
}

#[cfg(target_os = "windows")]
#[derive(Debug, Default, Deserialize)]
#[serde(default, rename_all = "camelCase")]
struct EarlyGeneral {
    auto_start: bool,
    run_as_admin: bool,
}

#[cfg(target_os = "windows")]
#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
struct StorageManifest {
    data_dir: PathBuf,
    environment: String,
    version: u16,
}

pub fn status(configured: bool) -> AdminLaunchStatus {
    AdminLaunchStatus {
        configured,
        running_as_admin: is_running_as_admin(),
        task_ready: is_scheduled_task_ready(),
    }
}

#[cfg(target_os = "windows")]
pub fn is_running_as_admin() -> bool {
    use windows::Win32::Foundation::{CloseHandle, HANDLE};
    use windows::Win32::Security::{
        GetTokenInformation, TokenElevation, TOKEN_ELEVATION, TOKEN_QUERY,
    };
    use windows::Win32::System::Threading::{GetCurrentProcess, OpenProcessToken};

    unsafe {
        let mut token_handle = HANDLE::default();
        if OpenProcessToken(GetCurrentProcess(), TOKEN_QUERY, &mut token_handle).is_err() {
            return false;
        }

        let mut elevation = TOKEN_ELEVATION::default();
        let mut return_length = 0_u32;
        let result = GetTokenInformation(
            token_handle,
            TokenElevation,
            Some(&mut elevation as *mut _ as *mut _),
            std::mem::size_of::<TOKEN_ELEVATION>() as u32,
            &mut return_length,
        );

        let _ = CloseHandle(token_handle);

        result.is_ok() && elevation.TokenIsElevated != 0
    }
}

#[cfg(not(target_os = "windows"))]
pub fn is_running_as_admin() -> bool {
    false
}

pub fn is_scheduled_task_ready() -> bool {
    #[cfg(target_os = "windows")]
    {
        is_scheduled_task_exists() && is_scheduled_task_path_valid()
    }

    #[cfg(not(target_os = "windows"))]
    {
        false
    }
}

/// Keep the highest-privilege logon task aligned with both persisted intents.
/// The task is used as the Windows autostart entry only when administrator mode
/// and autostart are both enabled.
pub fn sync_scheduled_task(configured: bool, auto_start: bool) -> bool {
    #[cfg(target_os = "windows")]
    {
        if configured && auto_start {
            if !is_running_as_admin() {
                return is_scheduled_task_ready();
            }
            if is_scheduled_task_ready() {
                return true;
            }

            if let Err(err) = create_scheduled_task() {
                log::warn!("sync admin scheduled task failed: {err}");
                return false;
            }
            return is_scheduled_task_ready();
        }

        if is_scheduled_task_exists() {
            if let Err(err) = delete_scheduled_task() {
                log::warn!("delete admin scheduled task failed: {err}");
            }
        }
        false
    }

    #[cfg(not(target_os = "windows"))]
    {
        let _ = configured;
        let _ = auto_start;
        false
    }
}

pub fn launch_elevated_current_process() -> Result<()> {
    #[cfg(target_os = "windows")]
    {
        if try_launch_elevated_current_process() {
            return Ok(());
        }

        Err(AppError::Other(anyhow::anyhow!(
            "administrator permission request was cancelled or failed"
        )))
    }

    #[cfg(not(target_os = "windows"))]
    {
        Err(AppError::Other(anyhow::anyhow!(
            "administrator launch is only available on Windows"
        )))
    }
}

pub fn handle_startup_auto_elevation() {
    #[cfg(target_os = "windows")]
    {
        if cfg!(debug_assertions) {
            return;
        }

        let Ok(early) = early_admin_settings() else {
            return;
        };
        if !early.general.run_as_admin {
            return;
        }

        if is_running_as_admin() {
            if early.general.auto_start {
                sync_scheduled_task(true, true);
            }
            return;
        }

        if has_admin_restart_marker() {
            return;
        }

        if try_launch_elevated_current_process() {
            std::process::exit(0);
        }
    }
}

#[cfg(target_os = "windows")]
fn is_scheduled_task_exists() -> bool {
    use std::os::windows::process::CommandExt;
    use std::process::Command;

    let output = Command::new("schtasks")
        .args(["/Query", "/TN", TASK_NAME])
        .creation_flags(CREATE_NO_WINDOW)
        .output();

    matches!(output, Ok(output) if output.status.success())
}

#[cfg(target_os = "windows")]
fn is_scheduled_task_path_valid() -> bool {
    use std::os::windows::process::CommandExt;
    use std::process::Command;

    let current_exe = match std::env::current_exe() {
        Ok(path) => path.to_string_lossy().to_lowercase(),
        Err(_) => return false,
    };
    let output = Command::new("schtasks")
        .args(["/Query", "/TN", TASK_NAME, "/XML", "ONE"])
        .creation_flags(CREATE_NO_WINDOW)
        .output();

    let Ok(output) = output else {
        return false;
    };
    if !output.status.success() {
        return false;
    }

    scheduled_task_xml_is_valid(&decode_command_output(&output.stdout), &current_exe)
}

#[cfg(target_os = "windows")]
fn scheduled_task_xml_is_valid(task_xml: &str, current_exe: &str) -> bool {
    let task_xml = task_xml.to_lowercase();

    task_xml.contains(&current_exe.to_lowercase()) && task_xml.contains("<logontrigger")
}

#[cfg(target_os = "windows")]
fn create_scheduled_task() -> Result<()> {
    use std::os::windows::process::CommandExt;
    use std::process::Command;

    let exe = std::env::current_exe().context("failed to resolve current executable")?;
    let action = scheduled_task_action(&exe);

    let _ = Command::new("schtasks")
        .args(["/Delete", "/TN", TASK_NAME, "/F"])
        .creation_flags(CREATE_NO_WINDOW)
        .output();

    let output = Command::new("schtasks")
        .args([
            "/Create", "/TN", TASK_NAME, "/TR", &action, "/SC", "ONLOGON", "/RL", "HIGHEST", "/F",
        ])
        .creation_flags(CREATE_NO_WINDOW)
        .output()
        .context("failed to create administrator launch task")?;

    if output.status.success() {
        return Ok(());
    }

    let stderr = decode_command_output(&output.stderr);
    Err(AppError::Other(anyhow::anyhow!(
        "failed to create administrator launch task: {stderr}"
    )))
}

#[cfg(target_os = "windows")]
fn delete_scheduled_task() -> Result<()> {
    use std::os::windows::process::CommandExt;
    use std::process::Command;

    let output = Command::new("schtasks")
        .args(["/Delete", "/TN", TASK_NAME, "/F"])
        .creation_flags(CREATE_NO_WINDOW)
        .output()
        .context("failed to delete administrator launch task")?;

    if output.status.success() {
        return Ok(());
    }

    let stderr = decode_command_output(&output.stderr);
    Err(AppError::Other(anyhow::anyhow!(
        "failed to delete administrator launch task: {stderr}"
    )))
}

#[cfg(target_os = "windows")]
fn decode_command_output(bytes: &[u8]) -> String {
    let utf16 = bytes.len() >= 2
        && bytes.len().is_multiple_of(2)
        && (bytes.starts_with(&[0xff, 0xfe])
            || bytes
                .chunks_exact(2)
                .take(64)
                .filter(|chunk| chunk[1] == 0)
                .count()
                > 16);

    if !utf16 {
        return String::from_utf8_lossy(bytes).into_owned();
    }

    let bytes = bytes.strip_prefix(&[0xff, 0xfe]).unwrap_or(bytes);
    let units = bytes
        .chunks_exact(2)
        .map(|chunk| u16::from_le_bytes([chunk[0], chunk[1]]))
        .collect::<Vec<_>>();

    String::from_utf16_lossy(&units)
}

#[cfg(target_os = "windows")]
fn run_via_scheduled_task() -> bool {
    use std::os::windows::process::CommandExt;
    use std::process::Command;

    let output = Command::new("schtasks")
        .args(["/Run", "/TN", TASK_NAME])
        .creation_flags(CREATE_NO_WINDOW)
        .output();

    matches!(output, Ok(output) if output.status.success())
}

#[cfg(target_os = "windows")]
fn try_launch_elevated_current_process() -> bool {
    if can_use_scheduled_task_for_current_args()
        && is_scheduled_task_exists()
        && is_scheduled_task_path_valid()
        && run_via_scheduled_task()
    {
        return true;
    }

    try_launch_with_uac()
}

#[cfg(target_os = "windows")]
fn try_launch_with_uac() -> bool {
    use std::os::windows::ffi::OsStrExt;
    use windows::core::PCWSTR;
    use windows::Win32::UI::Shell::ShellExecuteW;
    use windows::Win32::UI::WindowsAndMessaging::SW_SHOWNORMAL;

    let Ok(exe) = std::env::current_exe() else {
        return false;
    };
    let params = restart_args()
        .into_iter()
        .map(|arg| windows_args::quote_arg(&arg))
        .collect::<Vec<_>>()
        .join(" ");

    let operation = wide_null("runas");
    let file: Vec<u16> = exe
        .as_os_str()
        .encode_wide()
        .chain(std::iter::once(0))
        .collect();
    let params = wide_null(&params);

    unsafe {
        let result = ShellExecuteW(
            None,
            PCWSTR(operation.as_ptr()),
            PCWSTR(file.as_ptr()),
            PCWSTR(params.as_ptr()),
            PCWSTR(std::ptr::null()),
            SW_SHOWNORMAL,
        );

        result.0 as usize > 32
    }
}

#[cfg(target_os = "windows")]
fn early_admin_settings() -> Result<EarlySettings> {
    let Some(base) = std::env::var_os("LOCALAPPDATA") else {
        return Ok(EarlySettings::default());
    };

    let bootstrap = PathBuf::from(base)
        .join("com.ayangweb.eco-paste")
        .join(env_dir());
    let data_dir = early_data_dir(&bootstrap)?;
    let settings_path = data_dir.join("config").join(SETTINGS_FILENAME);
    if !settings_path.exists() {
        return Ok(EarlySettings::default());
    }

    let content = std::fs::read_to_string(&settings_path)
        .with_context(|| format!("failed to read early settings at {settings_path:?}"))?;
    let settings: EarlySettings =
        serde_json::from_str(&content).context("failed to parse early settings")?;

    Ok(settings)
}

#[cfg(target_os = "windows")]
fn early_data_dir(bootstrap: &Path) -> Result<PathBuf> {
    let default = bootstrap.to_path_buf();
    let manifest_path = bootstrap.join(STORAGE_MANIFEST_FILENAME);
    if !manifest_path.exists() {
        return Ok(default);
    }

    let content = std::fs::read_to_string(&manifest_path)
        .with_context(|| format!("failed to read storage manifest at {manifest_path:?}"))?;
    let manifest: StorageManifest =
        serde_json::from_str(&content).context("failed to parse storage manifest")?;
    if manifest.version != 1 || manifest.environment != env_dir() || !manifest.data_dir.exists() {
        return Ok(default);
    }

    Ok(manifest.data_dir)
}

#[cfg(target_os = "windows")]
fn scheduled_task_action(exe: &Path) -> String {
    format!(
        "{} {}",
        windows_args::quote_arg(exe.to_string_lossy()),
        windows_args::quote_arg(ADMIN_RESTARTED_ARG)
    )
}

#[cfg(target_os = "windows")]
fn can_use_scheduled_task_for_current_args() -> bool {
    std::env::args()
        .skip(1)
        .all(|arg| arg == ADMIN_RESTARTED_ARG)
}

#[cfg(target_os = "windows")]
fn has_admin_restart_marker() -> bool {
    std::env::args().any(|arg| arg == ADMIN_RESTARTED_ARG)
}

#[cfg(target_os = "windows")]
fn restart_args() -> Vec<String> {
    let mut args = std::env::args()
        .skip(1)
        .filter(|arg| arg != ADMIN_RESTARTED_ARG)
        .collect::<Vec<_>>();
    args.push(ADMIN_RESTARTED_ARG.to_owned());
    args
}

#[cfg(target_os = "windows")]
fn env_dir() -> &'static str {
    if cfg!(dev) {
        DEV_ENV_DIR
    } else {
        PROD_ENV_DIR
    }
}

#[cfg(target_os = "windows")]
fn wide_null(value: &str) -> Vec<u16> {
    use std::os::windows::ffi::OsStrExt;

    std::ffi::OsStr::new(value)
        .encode_wide()
        .chain(std::iter::once(0))
        .collect()
}

#[cfg(all(test, target_os = "windows"))]
mod tests {
    use super::{decode_command_output, scheduled_task_xml_is_valid};

    #[test]
    fn admin_task_requires_logon_trigger_and_current_executable() {
        let exe = r"c:\program files\ecopaste\ecopaste.exe";
        let valid = format!(
            "<Task><Triggers><LogonTrigger/></Triggers><Actions><Exec><Command>{exe}</Command></Exec></Actions></Task>"
        );
        let one_time = format!(
            "<Task><Triggers><TimeTrigger/></Triggers><Actions><Exec><Command>{exe}</Command></Exec></Actions></Task>"
        );

        assert!(scheduled_task_xml_is_valid(&valid, exe));
        assert!(!scheduled_task_xml_is_valid(&one_time, exe));
        assert!(!scheduled_task_xml_is_valid(
            &valid,
            r"c:\other\ecopaste.exe"
        ));
    }

    #[test]
    fn decodes_utf16_schtasks_output() {
        let text = "<Task><LogonTrigger/></Task>";
        let mut bytes = vec![0xff, 0xfe];
        for unit in text.encode_utf16() {
            bytes.extend_from_slice(&unit.to_le_bytes());
        }

        assert_eq!(decode_command_output(&bytes), text);
    }
}

use auto_launch::{AutoLaunch, AutoLaunchBuilder, WindowsEnableMode};

use crate::core::{AppError, Result};

use super::AUTO_LAUNCH_ARG;

const WINDOWS_RUN_KEY: &str = r"SOFTWARE\Microsoft\Windows\CurrentVersion\Run";
const WINDOWS_STARTUP_APPROVED_RUN_KEY: &str =
    r"SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run";
const WIN32_ERROR_FILE_NOT_FOUND: u32 = 2;
const WIN32_ERROR_ACCESS_DENIED: u32 = 5;

pub(super) struct PlatformAutostart {
    inner: AutoLaunch,
}

impl PlatformAutostart {
    pub(super) fn new(app_name: &str, exe_path: &str) -> Result<Self> {
        let mut builder = AutoLaunchBuilder::new();
        builder
            .set_app_name(app_name)
            .set_app_path(exe_path)
            .set_args(&[AUTO_LAUNCH_ARG])
            .set_windows_enable_mode(WindowsEnableMode::CurrentUser);

        let inner = builder.build().map_err(|err| {
            log::error!("autostart init: build AutoLaunch failed: {err}");
            AppError::Other(anyhow::anyhow!("{err}"))
        })?;

        Ok(Self { inner })
    }

    pub(super) fn is_enabled(&self) -> Result<bool> {
        self.inner.is_enabled().map_err(|err| {
            log::error!("autostart is_enabled failed: {err}");
            AppError::Other(anyhow::anyhow!("{err}"))
        })
    }

    pub(super) fn set_enabled(&self, enabled: bool) -> Result<()> {
        if enabled {
            if keep_existing_system_run_entry_if_needed(self.inner.get_app_name())? {
                return Ok(());
            }

            return self.inner.enable().map_err(|err| {
                log::error!("autostart enable failed: {err}");
                AppError::Other(anyhow::anyhow!("{err}"))
            });
        }

        cleanup_windows_run_entries(self.inner.get_app_name())
    }
}

fn keep_existing_system_run_entry_if_needed(app_name: &str) -> Result<bool> {
    if !system_run_entry_exists(app_name)? {
        return Ok(false);
    }

    match cleanup_system_run_entry(app_name) {
        Ok(()) => Ok(false),
        Err(err) if registry_error_is_access_denied(&err) => {
            cleanup_current_user_run_entry(app_name)?;
            log::warn!(
                "autostart: HKLM Run entry exists but cannot be removed; keeping it as the only startup entry"
            );
            Ok(true)
        }
        Err(err) => Err(registry_app_error("remove system autostart entry", err)),
    }
}

fn system_run_entry_exists(app_name: &str) -> Result<bool> {
    use windows_registry::LOCAL_MACHINE;

    match LOCAL_MACHINE
        .open(WINDOWS_RUN_KEY)
        .and_then(|key| key.get_string(app_name))
    {
        Ok(_) => Ok(true),
        Err(err) if registry_error_is_file_not_found(&err) => Ok(false),
        Err(err) => Err(registry_app_error("read system autostart entry", err)),
    }
}

fn cleanup_system_run_entry(app_name: &str) -> std::result::Result<(), windows_result::Error> {
    use windows_registry::LOCAL_MACHINE;

    cleanup_run_entry(LOCAL_MACHINE, app_name)?;
    cleanup_startup_approved_entry(LOCAL_MACHINE, app_name);
    Ok(())
}

fn cleanup_current_user_run_entry(app_name: &str) -> Result<()> {
    use windows_registry::CURRENT_USER;

    cleanup_run_entry(CURRENT_USER, app_name)
        .map_err(|err| registry_app_error("remove current-user autostart entry", err))?;
    cleanup_startup_approved_entry(CURRENT_USER, app_name);
    Ok(())
}

fn cleanup_windows_run_entries(app_name: &str) -> Result<()> {
    cleanup_current_user_run_entry(app_name)?;
    cleanup_system_run_entry(app_name)
        .map_err(|err| registry_app_error("remove system autostart entry", err))
}

fn cleanup_run_entry(
    root_key: &windows_registry::Key,
    app_name: &str,
) -> std::result::Result<(), windows_result::Error> {
    match root_key
        .options()
        .write()
        .open(WINDOWS_RUN_KEY)
        .and_then(|key| key.remove_value(app_name))
    {
        Ok(()) => Ok(()),
        Err(err) if registry_error_is_file_not_found(&err) => Ok(()),
        Err(err) => Err(err),
    }
}

fn cleanup_startup_approved_entry(root_key: &windows_registry::Key, app_name: &str) {
    match root_key
        .options()
        .write()
        .open(WINDOWS_STARTUP_APPROVED_RUN_KEY)
        .and_then(|key| key.remove_value(app_name))
    {
        Ok(()) => {}
        Err(err) if registry_error_is_file_not_found(&err) => {}
        Err(err) => {
            log::debug!("autostart: cleanup StartupApproved value failed: {err}");
        }
    }
}

fn registry_error_is_access_denied(err: &windows_result::Error) -> bool {
    err.code() == windows_result::HRESULT::from_win32(WIN32_ERROR_ACCESS_DENIED)
}

fn registry_error_is_file_not_found(err: &windows_result::Error) -> bool {
    err.code() == windows_result::HRESULT::from_win32(WIN32_ERROR_FILE_NOT_FOUND)
}

fn registry_app_error(action: &str, err: windows_result::Error) -> AppError {
    log::error!("autostart: {action} failed: {err}");
    AppError::Other(anyhow::anyhow!("{err}"))
}

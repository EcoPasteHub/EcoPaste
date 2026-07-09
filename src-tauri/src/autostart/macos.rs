use auto_launch::{AutoLaunch, AutoLaunchBuilder};

use crate::core::{AppError, Result};

use super::AUTO_LAUNCH_ARG;

pub(super) struct PlatformAutostart {
    inner: AutoLaunch,
}

impl PlatformAutostart {
    pub(super) fn new(app_name: &str, exe_path: &str) -> Result<Self> {
        let inner = AutoLaunchBuilder::new()
            .set_app_name(app_name)
            .set_app_path(exe_path)
            .set_args(&[AUTO_LAUNCH_ARG])
            .build()
            .map_err(|err| {
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
            return self.inner.enable().map_err(|err| {
                log::error!("autostart enable failed: {err}");
                AppError::Other(anyhow::anyhow!("{err}"))
            });
        }

        self.inner.disable().map_err(|err| {
            log::error!("autostart disable failed: {err}");
            AppError::Other(anyhow::anyhow!("{err}"))
        })
    }
}

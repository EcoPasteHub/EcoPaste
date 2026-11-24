//! 云同步命令：探测证书指纹、登录 / 登出、状态、拉取近期历史。

use tauri::{AppHandle, Manager};

use crate::cloud::{CloudError, CloudState, CloudStatus, ProbeResult};
use crate::core::{AppError, Result};
use crate::i18n::{self, commands::Key};

fn state(app: &AppHandle) -> CloudState {
    app.state::<CloudState>().inner().clone()
}

/// 把云同步错误映射为用户可读文案；未知错误保留原始描述。
fn cloud_error(app: &AppHandle, err: anyhow::Error) -> AppError {
    let key = match err.downcast_ref::<CloudError>() {
        Some(CloudError::Unreachable) => Some(Key::CloudUnreachable),
        Some(CloudError::PinMismatch) => Some(Key::CloudPinMismatch),
        Some(CloudError::BadCredentials) => Some(Key::CloudBadCredentials),
        Some(CloudError::Locked) => Some(Key::CloudLocked),
        Some(CloudError::NotSignedIn) => Some(Key::CloudNotSignedIn),
        Some(CloudError::BadAddress) => Some(Key::CloudBadAddress),
        None => None,
    };
    match key {
        Some(key) => AppError::Other(anyhow::anyhow!(i18n::commands::label(
            i18n::current_language(app),
            key
        ))),
        None => AppError::Other(err),
    }
}

#[tauri::command]
pub async fn cloud_get_status(app: AppHandle) -> Result<CloudStatus> {
    Ok(state(&app).status())
}

/// 连接服务器并返回其证书指纹，供用户与管理程序里显示的指纹核对。
#[tauri::command]
pub async fn cloud_probe(app: AppHandle, server_url: String) -> Result<ProbeResult> {
    state(&app)
        .probe(&server_url)
        .await
        .map_err(|err| cloud_error(&app, err))
}

#[tauri::command]
pub async fn cloud_login(
    app: AppHandle,
    server_url: String,
    fingerprint: String,
    username: String,
    password: String,
) -> Result<CloudStatus> {
    state(&app)
        .login(&server_url, &fingerprint, &username, &password)
        .await
        .map_err(|err| cloud_error(&app, err))
}

#[tauri::command]
pub async fn cloud_logout(app: AppHandle) -> Result<()> {
    state(&app).logout().await;
    Ok(())
}

/// 拉取云端近期文本 / 图片到本机历史，返回条数。
#[tauri::command]
pub async fn cloud_pull_recent(app: AppHandle) -> Result<usize> {
    state(&app)
        .pull_recent()
        .await
        .map_err(|err| cloud_error(&app, err))
}

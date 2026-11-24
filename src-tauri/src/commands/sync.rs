//! 局域网同步命令：状态、配对、设备管理。

use tauri::{AppHandle, Manager};

use crate::core::{AppError, Result};
use crate::i18n::{self, commands::Key};
use crate::sync::{PairError, PairingCode, PeerStatus, SyncState, SyncStatus};

fn state(app: &AppHandle) -> SyncState {
    app.state::<SyncState>().inner().clone()
}

fn pair_error(app: &AppHandle, err: PairError) -> AppError {
    let key = match err {
        PairError::NotRunning => Key::SyncNotRunning,
        PairError::NoDevice => Key::SyncNoDevice,
        PairError::CodeInvalid => Key::SyncCodeInvalid,
        PairError::AddressInvalid => Key::SyncAddressInvalid,
        PairError::Unreachable => Key::SyncUnreachable,
    };
    let message = i18n::commands::label(i18n::current_language(app), key);
    AppError::Other(anyhow::anyhow!(message))
}

#[tauri::command]
pub async fn sync_get_status(app: AppHandle) -> Result<SyncStatus> {
    Ok(state(&app).status())
}

/// 生成 6 位配对码并开始等待另一台设备输入。
#[tauri::command]
pub async fn sync_start_pairing(app: AppHandle) -> Result<PairingCode> {
    state(&app)
        .start_pairing()
        .map_err(|err| pair_error(&app, err))
}

#[tauri::command]
pub async fn sync_cancel_pairing(app: AppHandle) -> Result<()> {
    state(&app).cancel_pairing();
    Ok(())
}

/// 输入对端展示的 6 位配对码完成关联；`host` 可选，填写时直接连接该 IP（mDNS 不可用的兜底）。
#[tauri::command]
pub async fn sync_pair_with_code(
    app: AppHandle,
    code: String,
    host: Option<String>,
) -> Result<PeerStatus> {
    let code = code.trim();
    if code.len() != 6 || !code.chars().all(|c| c.is_ascii_digit()) {
        return Err(pair_error(&app, PairError::CodeInvalid));
    }
    state(&app)
        .pair_with_code(code, host.as_deref())
        .await
        .map_err(|err| pair_error(&app, err))
}

#[tauri::command]
pub async fn sync_rename_peer(app: AppHandle, id: String, name: String) -> Result<()> {
    if name.trim().is_empty() {
        return Ok(());
    }
    state(&app).rename_peer(&id, &name)?;
    Ok(())
}

#[tauri::command]
pub async fn sync_remove_peer(app: AppHandle, id: String) -> Result<()> {
    state(&app).remove_peer(&id).await?;
    Ok(())
}

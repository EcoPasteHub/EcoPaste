//! WebDAV 数据模型与状态契约。

use serde::{Deserialize, Serialize};

/// WebDAV 同步执行结果。
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct WebDavSyncResult {
    pub success: bool,
    pub uploaded_items: usize,
    pub uploaded_images: usize,
    pub backed_up_settings: bool,
    pub error: Option<String>,
    pub synced_at: String,
}

/// WebDAV 从远端恢复数据的结果。
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct WebDavRestoreResult {
    pub success: bool,
    pub restored_items: usize,
    pub restored_images: usize,
    pub restored_settings: bool,
    pub error: Option<String>,
}

/// WebDAV 连接测试结果。
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct WebDavTestResult {
    pub success: bool,
    pub message: String,
}

/// 远端同步清单文件结构。
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct WebDavManifest {
    pub version: u32,
    pub app_name: String,
    pub app_version: String,
    pub last_synced_at: String,
    pub total_items: usize,
    pub batches: Vec<String>,
}

/// 本地保存的同步状态，用于支持增量同步。
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct WebDavSyncState {
    pub last_synced_at: Option<String>,
    pub synced_image_hashes: Vec<String>,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_webdav_manifest_serialization() {
        let manifest = WebDavManifest {
            version: 1,
            app_name: "EcoPaste".to_owned(),
            app_version: "1.1.0".to_owned(),
            last_synced_at: "2026-11-03T18:00:00Z".to_owned(),
            total_items: 42,
            batches: vec!["batch_1.json".to_owned()],
        };

        let json = serde_json::to_string(&manifest).expect("serialize manifest");
        let decoded: WebDavManifest = serde_json::from_str(&json).expect("deserialize manifest");
        assert_eq!(decoded.version, 1);
        assert_eq!(decoded.total_items, 42);
        assert_eq!(decoded.batches.len(), 1);
    }

    #[test]
    fn test_webdav_sync_state_default() {
        let state = WebDavSyncState::default();
        assert!(state.last_synced_at.is_none());
        assert!(state.synced_image_hashes.is_empty());
    }
}

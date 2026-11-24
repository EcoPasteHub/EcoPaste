//! 本机设备身份与已配对设备表。
//!
//! 落盘位置：`<app_data_dir>/state/sync-devices.json`，原子替换写入。
//! 配对密钥目前明文存盘（与设置同目录同权限域）；后续可迁移到系统钥匙串。

use std::fs;
use std::path::{Path, PathBuf};
use std::sync::Mutex;

use anyhow::{Context, Result};
use chrono::Utc;
use serde::{Deserialize, Serialize};
use tauri::AppHandle;

const FILENAME: &str = "sync-devices.json";

/// 当前平台标识，随配对信息展示给对端。
pub fn current_platform() -> &'static str {
    if cfg!(target_os = "macos") {
        "macos"
    } else {
        "windows"
    }
}

/// 系统主机名，取不到时回落到应用名。
pub fn system_hostname() -> String {
    std::env::var("COMPUTERNAME")
        .or_else(|_| std::env::var("HOSTNAME"))
        .ok()
        .map(|name| name.trim().to_owned())
        .filter(|name| !name.is_empty())
        .unwrap_or_else(|| "EcoPaste".to_owned())
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "camelCase")]
pub struct PairedDevice {
    pub id: String,
    pub name: String,
    pub platform: String,
    /// 配对时协商出的 32 字节长期密钥（hex）。不得序列化给前端，见 [`PairedDevice::public`]。
    pub key: String,
    pub paired_at: String,
    /// 最近一次成功连接的对端地址（`ip:port`）；mDNS 不可用时据此回连。
    #[serde(default)]
    pub addr: Option<String>,
}

impl PairedDevice {
    pub fn key_bytes(&self) -> Option<[u8; 32]> {
        let bytes = hex_decode(&self.key)?;
        bytes.try_into().ok()
    }
}

#[derive(Debug, Default, Serialize, Deserialize)]
#[serde(default, rename_all = "camelCase")]
struct StoreData {
    device_id: String,
    devices: Vec<PairedDevice>,
}

pub struct DeviceStore {
    path: PathBuf,
    data: Mutex<StoreData>,
}

impl DeviceStore {
    pub fn load(app: &AppHandle) -> Result<Self> {
        let dir = crate::core::paths::state_dir(app)?;
        Self::load_from(dir.join(FILENAME))
    }

    pub fn load_from(path: PathBuf) -> Result<Self> {
        let mut data: StoreData = fs::read_to_string(&path)
            .ok()
            .and_then(|text| serde_json::from_str(&text).ok())
            .unwrap_or_default();

        let fresh = data.device_id.is_empty();
        if fresh {
            data.device_id = uuid::Uuid::new_v4().to_string();
        }

        let store = Self {
            path,
            data: Mutex::new(data),
        };
        if fresh {
            store.persist()?;
        }
        Ok(store)
    }

    pub fn device_id(&self) -> String {
        self.data
            .lock()
            .expect("device store poisoned")
            .device_id
            .clone()
    }

    pub fn list(&self) -> Vec<PairedDevice> {
        self.data
            .lock()
            .expect("device store poisoned")
            .devices
            .clone()
    }

    pub fn get(&self, id: &str) -> Option<PairedDevice> {
        self.list().into_iter().find(|d| d.id == id)
    }

    /// 新增或覆盖（重新配对）一台设备。
    pub fn upsert(
        &self,
        id: &str,
        name: &str,
        platform: &str,
        key: &[u8; 32],
        addr: Option<String>,
    ) -> Result<PairedDevice> {
        let device = PairedDevice {
            id: id.to_owned(),
            name: name.to_owned(),
            platform: platform.to_owned(),
            key: hex_encode(key),
            paired_at: Utc::now().to_rfc3339(),
            addr,
        };
        {
            let mut data = self.data.lock().expect("device store poisoned");
            data.devices.retain(|d| d.id != id);
            data.devices.push(device.clone());
        }
        self.persist()?;
        Ok(device)
    }

    /// 记录对端最新地址；未变化时不落盘。
    pub fn set_addr(&self, id: &str, addr: &str) -> Result<()> {
        let changed = {
            let mut data = self.data.lock().expect("device store poisoned");
            match data.devices.iter_mut().find(|d| d.id == id) {
                Some(device) if device.addr.as_deref() != Some(addr) => {
                    device.addr = Some(addr.to_owned());
                    true
                }
                _ => false,
            }
        };
        if changed {
            self.persist()?;
        }
        Ok(())
    }

    pub fn rename(&self, id: &str, name: &str) -> Result<bool> {
        let found = {
            let mut data = self.data.lock().expect("device store poisoned");
            match data.devices.iter_mut().find(|d| d.id == id) {
                Some(device) => {
                    device.name = name.to_owned();
                    true
                }
                None => false,
            }
        };
        if found {
            self.persist()?;
        }
        Ok(found)
    }

    pub fn remove(&self, id: &str) -> Result<bool> {
        let removed = {
            let mut data = self.data.lock().expect("device store poisoned");
            let before = data.devices.len();
            data.devices.retain(|d| d.id != id);
            data.devices.len() != before
        };
        if removed {
            self.persist()?;
        }
        Ok(removed)
    }

    fn persist(&self) -> Result<()> {
        let json = {
            let data = self.data.lock().expect("device store poisoned");
            serde_json::to_vec_pretty(&*data)?
        };
        write_atomic(&self.path, &json)
    }
}

fn write_atomic(path: &Path, bytes: &[u8]) -> Result<()> {
    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent).with_context(|| format!("create dir {parent:?}"))?;
    }
    let tmp = path.with_extension("json.tmp");
    fs::write(&tmp, bytes).with_context(|| format!("write {tmp:?}"))?;
    fs::rename(&tmp, path).with_context(|| format!("replace {path:?}"))?;
    Ok(())
}

pub fn hex_encode(bytes: &[u8]) -> String {
    bytes.iter().map(|b| format!("{b:02x}")).collect()
}

pub fn hex_decode(text: &str) -> Option<Vec<u8>> {
    if !text.len().is_multiple_of(2) {
        return None;
    }
    (0..text.len())
        .step_by(2)
        .map(|i| u8::from_str_radix(text.get(i..i + 2)?, 16).ok())
        .collect()
}

#[cfg(test)]
mod tests {
    use super::*;

    fn temp_path() -> PathBuf {
        std::env::temp_dir()
            .join(format!("ecopaste-sync-{}", uuid::Uuid::new_v4()))
            .join(FILENAME)
    }

    #[test]
    fn device_id_is_stable_across_reloads() {
        let path = temp_path();
        let first = DeviceStore::load_from(path.clone()).unwrap().device_id();
        let second = DeviceStore::load_from(path.clone()).unwrap().device_id();
        assert_eq!(first, second);
        fs::remove_dir_all(path.parent().unwrap()).ok();
    }

    #[test]
    fn paired_devices_persist_and_remove() {
        let path = temp_path();
        let store = DeviceStore::load_from(path.clone()).unwrap();
        store
            .upsert("peer", "Laptop", "windows", &[9u8; 32], None)
            .unwrap();
        store.set_addr("peer", "10.0.0.2:52526").unwrap();

        let reloaded = DeviceStore::load_from(path.clone()).unwrap();
        let device = reloaded.get("peer").unwrap();
        assert_eq!(device.key_bytes(), Some([9u8; 32]));
        assert_eq!(device.addr.as_deref(), Some("10.0.0.2:52526"));

        assert!(reloaded.rename("peer", "Desk").unwrap());
        assert_eq!(reloaded.get("peer").unwrap().name, "Desk");
        assert!(reloaded.remove("peer").unwrap());
        assert!(reloaded.get("peer").is_none());
        fs::remove_dir_all(path.parent().unwrap()).ok();
    }

    #[test]
    fn hex_round_trips() {
        let bytes = [0u8, 1, 254, 255];
        assert_eq!(hex_decode(&hex_encode(&bytes)).unwrap(), bytes);
        assert!(hex_decode("abc").is_none());
    }
}

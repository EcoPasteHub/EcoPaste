//! mDNS 设备发现：注册本机服务、浏览同网段其它 EcoPaste。

use std::net::IpAddr;

use anyhow::{Context, Result};
use mdns_sd::{Receiver, ServiceDaemon, ServiceEvent, ServiceInfo};

use super::protocol::PROTOCOL_VERSION;

pub const SERVICE_TYPE: &str = "_ecopaste-sync._tcp.local.";

/// 浏览到的对端服务。
#[derive(Debug, Clone)]
pub struct Discovered {
    pub fullname: String,
    pub device_id: String,
    pub addrs: Vec<IpAddr>,
    pub port: u16,
    /// 对端正在展示配对码，接受配对连接。
    pub pairing: bool,
}

impl Discovered {
    /// 从 mDNS 解析结果提取对端信息；缺少 `id` 或协议版本不符时返回 `None`。
    pub fn from_service(info: &ServiceInfo) -> Option<Self> {
        let device_id = info.get_property_val_str("id")?.to_owned();
        let version: u8 = info.get_property_val_str("v")?.parse().ok()?;
        if version != PROTOCOL_VERSION {
            return None;
        }

        let mut addrs: Vec<IpAddr> = info
            .get_addresses()
            .iter()
            .filter(|ip| ip.is_ipv4())
            .copied()
            .collect();
        addrs.sort();

        Some(Self {
            fullname: info.get_fullname().to_owned(),
            device_id,
            addrs,
            port: info.get_port(),
            pairing: info.get_property_val_str("pair") == Some("1"),
        })
    }
}

pub struct Discovery {
    daemon: ServiceDaemon,
    device_id: String,
    name: String,
    port: u16,
}

impl Discovery {
    /// 注册本机服务并开始浏览。
    pub fn start(device_id: &str, name: &str, port: u16) -> Result<(Self, Receiver<ServiceEvent>)> {
        let daemon = ServiceDaemon::new().context("start mdns daemon")?;
        let this = Self {
            daemon,
            device_id: device_id.to_owned(),
            name: name.to_owned(),
            port,
        };
        this.announce(false)?;
        let events = this.daemon.browse(SERVICE_TYPE).context("browse mdns")?;
        Ok((this, events))
    }

    /// 更新 `pair` 标记并重新通告。
    pub fn set_pairing(&self, pairing: bool) -> Result<()> {
        self.announce(pairing)
    }

    fn announce(&self, pairing: bool) -> Result<()> {
        let short: String = self
            .device_id
            .chars()
            .filter(|c| *c != '-')
            .take(12)
            .collect();
        let host = format!("ecopaste-{short}.local.");
        let props = [
            ("id", self.device_id.as_str()),
            ("name", self.name.as_str()),
            ("pair", if pairing { "1" } else { "0" }),
            ("v", &PROTOCOL_VERSION.to_string()),
        ];
        let info = ServiceInfo::new(SERVICE_TYPE, &short, &host, "", self.port, &props[..])
            .context("build mdns service")?
            .enable_addr_auto();
        self.daemon
            .register(info)
            .context("register mdns service")?;
        Ok(())
    }

    pub fn shutdown(&self) {
        if let Err(err) = self.daemon.shutdown() {
            log::debug!("mdns shutdown: {err}");
        }
    }
}

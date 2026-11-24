//! 同步运行时：监听 / 发现 / 连接管理 / 广播 / 配对入口。

use std::collections::HashMap;
use std::net::{IpAddr, SocketAddr};
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::{Arc, Mutex};
use std::time::{Duration, Instant};

use anyhow::{anyhow, bail, Result};
use mdns_sd::ServiceEvent;
use serde::Serialize;
use serde_json::json;
use tauri::{AppHandle, Emitter, Manager};
use tokio::io::{AsyncRead, AsyncWrite};
use tokio::net::{TcpListener, TcpStream};
use tokio::sync::{mpsc, oneshot, watch};
use tokio::time::timeout;

use super::apply;
use super::discovery::{Discovered, Discovery};
use super::files::{self, LocalOffer, Offers, OFFER_TTL};
use super::identity::{current_platform, system_hostname, DeviceStore};
use super::pairing::{PairOutcome, Pairing, PairingCode, CODE_TTL};
use super::protocol::{
    read_plain, write_plain, DeviceMeta, FileEntry, Hello, Intro, Message, SecureReader,
    SecureWriter, CHUNK_SIZE, DEFAULT_PORT, PROTOCOL_VERSION,
};
use super::{pairing, remote_files, session, DEVICES_EVENT, PAIRING_EVENT};
use crate::db::models::{ClipboardItem, ClipboardKind, ClipboardSubKind};
use crate::settings::SettingsStore;

const HANDSHAKE_TIMEOUT: Duration = Duration::from_secs(15);
const CONNECT_TIMEOUT: Duration = Duration::from_secs(3);
const PING_INTERVAL: Duration = Duration::from_secs(15);
const IDLE_TIMEOUT: Duration = Duration::from_secs(45);
const SUPERVISE_INTERVAL: Duration = Duration::from_secs(5);
const FETCH_TIMEOUT: Duration = Duration::from_secs(60);
const OUTBOX_DEPTH: usize = 64;

struct Running {
    discovery: Discovery,
    stop: watch::Sender<bool>,
    name: String,
    port: u16,
}

struct PeerConn {
    conn_id: u64,
    tx: mpsc::Sender<Message>,
}

struct Pending {
    peer_id: String,
    tx: oneshot::Sender<std::result::Result<Vec<u8>, String>>,
}

/// 已为某条内容登记的文件 offer：`(offer id, 清单, 登记时间)`。
type PreparedOffer = (String, Vec<FileEntry>, Instant);

struct Inner {
    app: AppHandle,
    store: DeviceStore,
    pairing: Pairing,
    running: Mutex<Option<Running>>,
    peers: Mutex<HashMap<String, PeerConn>>,
    discovered: Mutex<HashMap<String, Discovered>>,
    offers: Mutex<Offers>,
    /// 内容哈希 -> 已登记的文件 offer，局域网与云端广播共用同一份清单。
    prepared: Mutex<HashMap<String, PreparedOffer>>,
    pending: Mutex<HashMap<u64, Pending>>,
    seq: AtomicU64,
}

#[derive(Clone)]
pub struct SyncState {
    inner: Arc<Inner>,
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct PeerStatus {
    pub id: String,
    pub name: String,
    pub platform: String,
    pub online: bool,
    pub paired_at: String,
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct SyncStatus {
    pub enabled: bool,
    pub running: bool,
    pub device_id: String,
    pub device_name: String,
    pub platform: String,
    pub peers: Vec<PeerStatus>,
    pub pairing: Option<PairingCode>,
    /// 本机可供手动连接的地址（`ip`，端口非默认时为 `ip:port`）。
    pub addresses: Vec<String>,
}

/// 配对失败的原因，对应前端展示的文案。
#[derive(Debug, thiserror::Error)]
pub enum PairError {
    #[error("sync is not running")]
    NotRunning,
    #[error("no device is showing a pairing code")]
    NoDevice,
    #[error("pairing code is wrong or expired")]
    CodeInvalid,
    #[error("address is invalid")]
    AddressInvalid,
    #[error("device is unreachable")]
    Unreachable,
}

/// 解析手动输入的地址：`ip` 或 `ip:port`，省略端口时使用 [`DEFAULT_PORT`]。
pub fn parse_manual_address(input: &str) -> Option<SocketAddr> {
    let input = input.trim();
    if let Ok(addr) = input.parse::<SocketAddr>() {
        return Some(addr);
    }
    input
        .parse::<IpAddr>()
        .ok()
        .map(|ip| SocketAddr::new(ip, DEFAULT_PORT))
}

/// 本机 IPv4 地址：排除回环、链路本地与代理 TUN 常用的 198.18.0.0/15，私网地址优先。
fn local_ipv4_addresses() -> Vec<IpAddr> {
    let mut addrs: Vec<IpAddr> = if_addrs::get_if_addrs()
        .unwrap_or_default()
        .into_iter()
        .filter(|iface| !iface.is_loopback() && !iface.is_link_local())
        .map(|iface| iface.ip())
        .filter(|ip| match ip {
            IpAddr::V4(v4) => !(v4.octets()[0] == 198 && (v4.octets()[1] & 0xFE) == 18),
            IpAddr::V6(_) => false,
        })
        .collect();
    addrs.sort_by_key(|ip| match ip {
        IpAddr::V4(v4) if v4.is_private() => 0,
        _ => 1,
    });
    addrs.dedup();
    addrs
}

impl SyncState {
    pub fn new(app: AppHandle, store: DeviceStore) -> Self {
        Self {
            inner: Arc::new(Inner {
                app,
                store,
                pairing: Pairing::default(),
                running: Mutex::new(None),
                peers: Mutex::new(HashMap::new()),
                discovered: Mutex::new(HashMap::new()),
                offers: Mutex::new(Offers::default()),
                prepared: Mutex::new(HashMap::new()),
                pending: Mutex::new(HashMap::new()),
                seq: AtomicU64::new(1),
            }),
        }
    }

    /// 按当前设置启停同步；设备名变化时重启以刷新通告。
    pub async fn apply_settings(&self) {
        let settings = self.inner.app.state::<SettingsStore>().snapshot();
        let name = self.inner.device_name();
        let (running, current_name) = {
            let guard = self.inner.running.lock().expect("sync poisoned");
            (guard.is_some(), guard.as_ref().map(|r| r.name.clone()))
        };

        if settings.sync.enabled {
            if running && current_name.as_deref() != Some(name.as_str()) {
                self.stop();
            }
            if !self.is_running() {
                if let Err(err) = self.start().await {
                    log::error!("sync start failed: {err:#}");
                }
            }
        } else if running {
            self.stop();
        }
    }

    /// 测试用：当前监听端口。
    #[cfg(test)]
    pub fn listen_port(&self) -> Option<u16> {
        self.inner.listen_port()
    }

    pub fn is_running(&self) -> bool {
        self.inner.running.lock().expect("sync poisoned").is_some()
    }

    async fn start(&self) -> Result<()> {
        let inner = self.inner.clone();
        let listener = match TcpListener::bind(("0.0.0.0", DEFAULT_PORT)).await {
            Ok(listener) => listener,
            Err(err) => {
                log::warn!("sync: port {DEFAULT_PORT} unavailable ({err}), using a random port");
                TcpListener::bind(("0.0.0.0", 0)).await?
            }
        };
        let port = listener.local_addr()?.port();
        let name = inner.device_name();

        let (discovery, events) = Discovery::start(&inner.store.device_id(), &name, port)?;
        let (stop_tx, stop_rx) = watch::channel(false);

        *inner.running.lock().expect("sync poisoned") = Some(Running {
            discovery,
            stop: stop_tx,
            name,
            port,
        });

        tauri::async_runtime::spawn(accept_loop(inner.clone(), listener, stop_rx.clone()));
        tauri::async_runtime::spawn(browse_loop(inner.clone(), events, stop_rx.clone()));
        tauri::async_runtime::spawn(supervise_loop(inner.clone(), stop_rx));

        log::info!("sync started on port {port}");
        inner.emit_devices();
        Ok(())
    }

    fn stop(&self) {
        let inner = &self.inner;
        if let Some(running) = inner.running.lock().expect("sync poisoned").take() {
            let _ = running.stop.send(true);
            running.discovery.shutdown();
        }
        inner.pairing.cancel();
        inner.peers.lock().expect("sync poisoned").clear();
        inner.discovered.lock().expect("sync poisoned").clear();
        inner.fail_all_pending(None);
        log::info!("sync stopped");
        inner.emit_devices();
    }

    pub fn status(&self) -> SyncStatus {
        let inner = &self.inner;
        let settings = inner.app.state::<SettingsStore>().snapshot();
        let online: Vec<String> = inner
            .peers
            .lock()
            .expect("sync poisoned")
            .keys()
            .cloned()
            .collect();

        SyncStatus {
            enabled: settings.sync.enabled,
            running: self.is_running(),
            device_id: inner.store.device_id(),
            device_name: inner.device_name(),
            platform: current_platform().to_owned(),
            peers: inner
                .store
                .list()
                .into_iter()
                .map(|d| PeerStatus {
                    online: online.contains(&d.id),
                    id: d.id,
                    name: d.name,
                    platform: d.platform,
                    paired_at: d.paired_at,
                })
                .collect(),
            pairing: inner.pairing.snapshot(),
            addresses: match inner.listen_port() {
                Some(port) => local_ipv4_addresses()
                    .into_iter()
                    .map(|ip| {
                        if port == DEFAULT_PORT {
                            ip.to_string()
                        } else {
                            SocketAddr::new(ip, port).to_string()
                        }
                    })
                    .collect(),
                None => Vec::new(),
            },
        }
    }

    /// 生成并展示配对码，同时在 mDNS 上标记本机接受配对。
    pub fn start_pairing(&self) -> Result<PairingCode, PairError> {
        if !self.is_running() {
            return Err(PairError::NotRunning);
        }
        let code = self.inner.pairing.start();
        self.inner.refresh_pair_flag();

        let inner = self.inner.clone();
        tauri::async_runtime::spawn(async move {
            tokio::time::sleep(CODE_TTL + Duration::from_secs(1)).await;
            if !inner.pairing.is_active() {
                inner.refresh_pair_flag();
                inner.emit_pairing("expired", None);
            }
        });
        Ok(code)
    }

    pub fn cancel_pairing(&self) {
        self.inner.pairing.cancel();
        self.inner.refresh_pair_flag();
    }

    /// 输入对端展示的 6 位码配对。
    ///
    /// `host` 非空时直接连接该地址（mDNS 被网络拦截时的兜底）；
    /// 否则尝试所有经 mDNS 发现、正在展示配对码的设备。
    pub async fn pair_with_code(
        &self,
        code: &str,
        host: Option<&str>,
    ) -> Result<PeerStatus, PairError> {
        let inner = self.inner.clone();
        if !self.is_running() {
            return Err(PairError::NotRunning);
        }

        if let Some(host) = host.map(str::trim).filter(|h| !h.is_empty()) {
            let addr = parse_manual_address(host).ok_or(PairError::AddressInvalid)?;
            let target = Discovered {
                fullname: String::new(),
                device_id: String::new(),
                addrs: vec![addr.ip()],
                port: addr.port(),
                pairing: true,
            };
            return match connect_pair(&inner, &target, code).await {
                Ok((outcome, dialed)) => {
                    let device = inner.finish_pairing(outcome, Some(dialed.ip()))?;
                    Ok(device_status(device))
                }
                Err(ConnectError::Unreachable) => Err(PairError::Unreachable),
                Err(ConnectError::Other(err)) => {
                    log::info!("sync: pairing with {addr} failed: {err:#}");
                    Err(PairError::CodeInvalid)
                }
            };
        }

        let my_id = inner.store.device_id();
        let candidates: Vec<Discovered> = inner
            .discovered
            .lock()
            .expect("sync poisoned")
            .values()
            .filter(|d| d.pairing && d.device_id != my_id)
            .cloned()
            .collect();
        if candidates.is_empty() {
            return Err(PairError::NoDevice);
        }

        for candidate in candidates {
            match connect_pair(&inner, &candidate, code).await {
                Ok((outcome, dialed)) => {
                    let device = inner.finish_pairing(outcome, Some(dialed.ip()))?;
                    return Ok(device_status(device));
                }
                Err(err) => {
                    log::info!("sync: pairing with {} failed: {err}", candidate.device_id)
                }
            }
        }
        Err(PairError::CodeInvalid)
    }

    pub async fn remove_peer(&self, id: &str) -> Result<()> {
        let tx = self
            .inner
            .peers
            .lock()
            .expect("sync poisoned")
            .get(id)
            .map(|p| p.tx.clone());
        if let Some(tx) = tx {
            // 尽力通知对端；对端离线时对方下次连接会因密钥不存在被拒绝。
            let _ = timeout(Duration::from_secs(1), tx.send(Message::Unpair)).await;
        }
        self.inner.store.remove(id)?;
        self.inner.peers.lock().expect("sync poisoned").remove(id);
        self.inner.emit_devices();
        Ok(())
    }

    pub fn rename_peer(&self, id: &str, name: &str) -> Result<()> {
        self.inner.store.rename(id, name.trim())?;
        self.inner.emit_devices();
        Ok(())
    }

    /// 为复制的文件生成清单并登记 offer（同一条内容复用，供局域网与云端共用）。
    pub async fn prepare_files(&self, item: &ClipboardItem) -> Result<(String, Vec<FileEntry>)> {
        let inner = &self.inner;
        {
            let mut prepared = inner.prepared.lock().expect("sync poisoned");
            prepared.retain(|_, (_, _, at)| at.elapsed() < OFFER_TTL);
            if let Some((offer, entries, _)) = prepared.get(&item.content_hash) {
                if inner.offers.lock().expect("sync poisoned").contains(offer) {
                    return Ok((offer.clone(), entries.clone()));
                }
            }
        }

        let roots: Vec<String> = item
            .content
            .split('\n')
            .filter(|s| !s.is_empty())
            .map(str::to_owned)
            .collect();
        let (entries, paths) =
            tauri::async_runtime::spawn_blocking(move || files::build_offer(&roots)).await??;

        let offer = uuid::Uuid::new_v4().to_string();
        inner
            .offers
            .lock()
            .expect("sync poisoned")
            .register(offer.clone(), LocalOffer { paths });
        inner.prepared.lock().expect("sync poisoned").insert(
            item.content_hash.clone(),
            (offer.clone(), entries.clone(), Instant::now()),
        );
        Ok((offer, entries))
    }

    /// 读取本机已登记 offer 中某文件的一段（云端中转与局域网共用）。
    pub async fn read_offer_range(
        &self,
        offer: &str,
        index: u32,
        offset: u64,
        len: u32,
    ) -> Result<Vec<u8>> {
        self.inner.read_offer_range(offer, index, offset, len).await
    }

    /// 本机新复制了一条内容：按设置推送给所有在线的已配对设备。
    pub fn broadcast(&self, item: ClipboardItem, ts: u64) {
        let inner = self.inner.clone();
        tauri::async_runtime::spawn(async move {
            if let Err(err) = broadcast_item(&inner, item, ts).await {
                log::warn!("sync: broadcast failed: {err:#}");
            }
        });
    }
}

impl Inner {
    fn device_name(&self) -> String {
        let configured = self
            .app
            .state::<SettingsStore>()
            .snapshot()
            .sync
            .device_name;
        let configured = configured.trim();
        if configured.is_empty() {
            system_hostname()
        } else {
            configured.to_owned()
        }
    }

    fn emit_devices(&self) {
        if let Err(err) = self.app.emit(DEVICES_EVENT, ()) {
            log::warn!("emit {DEVICES_EVENT} failed: {err}");
        }
    }

    fn emit_pairing(&self, state: &str, peer: Option<&str>) {
        if let Err(err) = self
            .app
            .emit(PAIRING_EVENT, json!({ "state": state, "peer": peer }))
        {
            log::warn!("emit {PAIRING_EVENT} failed: {err}");
        }
    }

    fn refresh_pair_flag(&self) {
        let active = self.pairing.is_active();
        if let Some(running) = self.running.lock().expect("sync poisoned").as_ref() {
            if let Err(err) = running.discovery.set_pairing(active) {
                log::warn!("sync: update pairing flag failed: {err:#}");
            }
        }
    }

    fn listen_port(&self) -> Option<u16> {
        self.running
            .lock()
            .expect("sync poisoned")
            .as_ref()
            .map(|r| r.port)
    }

    fn my_meta(&self) -> DeviceMeta {
        DeviceMeta {
            name: self.device_name(),
            platform: current_platform().to_owned(),
            port: self.listen_port().unwrap_or(DEFAULT_PORT),
        }
    }

    /// 记录对端地址；失败只记日志，不影响连接。
    fn remember_addr(&self, peer_id: &str, addr: SocketAddr) {
        if let Err(err) = self.store.set_addr(peer_id, &addr.to_string()) {
            log::warn!("sync: persist address of {peer_id} failed: {err:#}");
        }
    }

    fn finish_pairing(
        &self,
        outcome: PairOutcome,
        peer_ip: Option<IpAddr>,
    ) -> Result<super::identity::PairedDevice, PairError> {
        let addr = peer_ip.map(|ip| SocketAddr::new(ip, outcome.peer_port).to_string());
        let device = self
            .store
            .upsert(
                &outcome.peer_id,
                &outcome.peer_name,
                &outcome.peer_platform,
                &outcome.long_key,
                addr,
            )
            .map_err(|err| {
                log::error!("sync: persist paired device failed: {err:#}");
                PairError::CodeInvalid
            })?;
        self.emit_devices();
        Ok(device)
    }

    fn is_stopped(stop: &watch::Receiver<bool>) -> bool {
        *stop.borrow()
    }

    fn fail_all_pending(&self, peer_id: Option<&str>) {
        let mut pending = self.pending.lock().expect("sync poisoned");
        let ids: Vec<u64> = pending
            .iter()
            .filter(|(_, p)| peer_id.is_none_or(|id| p.peer_id == id))
            .map(|(req, _)| *req)
            .collect();
        for req in ids {
            if let Some(p) = pending.remove(&req) {
                let _ = p.tx.send(Err("device disconnected".to_owned()));
            }
        }
    }

    /// 收到分块请求时立即续期文件清单，再把磁盘读取交给异步任务。
    fn read_offer_range(
        &self,
        offer: &str,
        index: u32,
        offset: u64,
        len: u32,
    ) -> impl std::future::Future<Output = Result<Vec<u8>>> + Send + 'static {
        let local = self
            .offers
            .lock()
            .expect("sync poisoned")
            .get_for_read(offer);

        async move {
            let local = local.ok_or_else(|| anyhow!("file is no longer available"))?;
            files::read_range(&local, index, offset, len.min(CHUNK_SIZE as u32)).await
        }
    }
}

/// 向对端拉取文件的一段数据（虚拟文件流使用）。
pub async fn fetch_file_range(
    state: &SyncState,
    peer_id: &str,
    offer: &str,
    index: u32,
    offset: u64,
    len: u32,
) -> Result<Vec<u8>> {
    let inner = &state.inner;
    if let Some(origin) = peer_id.strip_prefix(crate::cloud::PEER_PREFIX) {
        return crate::cloud::fetch_file_range(&inner.app, origin, offer, index, offset, len).await;
    }
    let tx = inner
        .peers
        .lock()
        .expect("sync poisoned")
        .get(peer_id)
        .map(|p| p.tx.clone())
        .ok_or_else(|| anyhow!("device is offline"))?;

    let req = inner.seq.fetch_add(1, Ordering::Relaxed);
    let (reply_tx, reply_rx) = oneshot::channel();
    inner.pending.lock().expect("sync poisoned").insert(
        req,
        Pending {
            peer_id: peer_id.to_owned(),
            tx: reply_tx,
        },
    );

    let sent = tx
        .send(Message::FetchFile {
            req,
            offer: offer.to_owned(),
            index,
            offset,
            len,
        })
        .await;
    if sent.is_err() {
        inner.pending.lock().expect("sync poisoned").remove(&req);
        bail!("device is offline");
    }

    match timeout(FETCH_TIMEOUT, reply_rx).await {
        Ok(Ok(Ok(data))) => Ok(data),
        Ok(Ok(Err(message))) => bail!("{message}"),
        Ok(Err(_)) => bail!("device disconnected"),
        Err(_) => {
            inner.pending.lock().expect("sync poisoned").remove(&req);
            bail!("transfer timed out")
        }
    }
}

async fn accept_loop(inner: Arc<Inner>, listener: TcpListener, mut stop: watch::Receiver<bool>) {
    loop {
        tokio::select! {
            _ = stop.changed() => break,
            accepted = listener.accept() => {
                let Ok((stream, addr)) = accepted else { continue };
                let inner = inner.clone();
                let stop = stop.clone();
                tauri::async_runtime::spawn(async move {
                    if let Err(err) = handle_incoming(inner, stream, addr, stop).await {
                        log::debug!("sync: incoming connection from {addr} ended: {err:#}");
                    }
                });
            }
        }
    }
}

async fn handle_incoming(
    inner: Arc<Inner>,
    stream: TcpStream,
    addr: SocketAddr,
    stop: watch::Receiver<bool>,
) -> Result<()> {
    stream.set_nodelay(true).ok();
    let (mut reader, mut writer) = tokio::io::split(stream);
    let my_id = inner.store.device_id();

    let intro: Intro = timeout(HANDSHAKE_TIMEOUT, read_plain(&mut reader)).await??;
    match intro {
        Intro::Pair { version, device_id } => {
            if version != PROTOCOL_VERSION {
                bail!("protocol version mismatch");
            }
            let Some(code) = inner.pairing.begin_attempt() else {
                bail!("no pairing code is active");
            };
            write_plain(
                &mut writer,
                &Hello {
                    device_id: my_id.clone(),
                },
            )
            .await?;

            let result = timeout(
                HANDSHAKE_TIMEOUT,
                pairing::run_handshake(
                    reader,
                    writer,
                    &code,
                    &my_id,
                    &device_id,
                    inner.my_meta(),
                    false,
                ),
            )
            .await;

            match result {
                Ok(Ok(outcome)) => {
                    inner.pairing.cancel();
                    inner.refresh_pair_flag();
                    let name = outcome.peer_name.clone();
                    inner.finish_pairing(outcome, Some(addr.ip()))?;
                    inner.emit_pairing("success", Some(&name));
                    Ok(())
                }
                other => {
                    inner.pairing.finish_failed();
                    inner.refresh_pair_flag();
                    inner.emit_pairing("failed", None);
                    match other {
                        Ok(Err(err)) => Err(err),
                        _ => bail!("pairing timed out"),
                    }
                }
            }
        }
        Intro::Session {
            version,
            device_id,
            port,
        } => {
            if version != PROTOCOL_VERSION {
                bail!("protocol version mismatch");
            }
            let key = inner
                .store
                .get(&device_id)
                .and_then(|d| d.key_bytes())
                .ok_or_else(|| anyhow!("unknown device"))?;
            write_plain(
                &mut writer,
                &Hello {
                    device_id: my_id.clone(),
                },
            )
            .await?;

            let (secure_r, secure_w) = timeout(
                HANDSHAKE_TIMEOUT,
                session::server_handshake(reader, writer, &key, &my_id, &device_id),
            )
            .await??;
            inner.remember_addr(&device_id, SocketAddr::new(addr.ip(), port));
            run_connection(inner, device_id, secure_r, secure_w, stop).await;
            Ok(())
        }
    }
}

/// 依次尝试对端的各个地址，返回连上的流与实际地址。
async fn dial(discovered: &Discovered) -> Option<(TcpStream, SocketAddr)> {
    for ip in &discovered.addrs {
        let addr = SocketAddr::new(*ip, discovered.port);
        if let Ok(Ok(stream)) = timeout(CONNECT_TIMEOUT, TcpStream::connect(addr)).await {
            stream.set_nodelay(true).ok();
            return Some((stream, addr));
        }
    }
    None
}

/// 配对连接失败的分类：连不上（网络问题）与其它（码错误、协议不符等）。
#[derive(Debug)]
enum ConnectError {
    Unreachable,
    Other(anyhow::Error),
}

impl std::fmt::Display for ConnectError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Unreachable => write!(f, "device is unreachable"),
            Self::Other(err) => write!(f, "{err:#}"),
        }
    }
}

impl From<anyhow::Error> for ConnectError {
    fn from(err: anyhow::Error) -> Self {
        Self::Other(err)
    }
}

/// 发起配对。`peer.device_id` 为空表示手动地址，接受对端自报的身份。
async fn connect_pair(
    inner: &Arc<Inner>,
    peer: &Discovered,
    code: &str,
) -> std::result::Result<(PairOutcome, SocketAddr), ConnectError> {
    let (stream, dialed) = dial(peer).await.ok_or(ConnectError::Unreachable)?;
    let (mut reader, mut writer) = tokio::io::split(stream);
    let my_id = inner.store.device_id();

    write_plain(
        &mut writer,
        &Intro::Pair {
            version: PROTOCOL_VERSION,
            device_id: my_id.clone(),
        },
    )
    .await?;
    let hello: Hello = timeout(HANDSHAKE_TIMEOUT, read_plain(&mut reader))
        .await
        .map_err(|_| anyhow!("handshake timed out"))??;
    if !peer.device_id.is_empty() && hello.device_id != peer.device_id {
        return Err(anyhow!("device identity mismatch").into());
    }
    if hello.device_id == my_id {
        return Err(anyhow!("cannot pair with this device itself").into());
    }

    let outcome = timeout(
        HANDSHAKE_TIMEOUT,
        pairing::run_handshake(
            reader,
            writer,
            code,
            &my_id,
            &hello.device_id,
            inner.my_meta(),
            true,
        ),
    )
    .await
    .map_err(|_| anyhow!("handshake timed out"))??;
    Ok((outcome, dialed))
}

fn device_status(device: super::identity::PairedDevice) -> PeerStatus {
    PeerStatus {
        online: false,
        id: device.id,
        name: device.name,
        platform: device.platform,
        paired_at: device.paired_at,
    }
}

async fn connect_session(
    inner: Arc<Inner>,
    peer: Discovered,
    stop: watch::Receiver<bool>,
) -> Result<()> {
    let key = inner
        .store
        .get(&peer.device_id)
        .and_then(|d| d.key_bytes())
        .ok_or_else(|| anyhow!("device is not paired"))?;
    let (stream, dialed) = dial(&peer)
        .await
        .ok_or_else(|| anyhow!("device is unreachable"))?;
    let (mut reader, mut writer) = tokio::io::split(stream);
    let my_id = inner.store.device_id();

    write_plain(
        &mut writer,
        &Intro::Session {
            version: PROTOCOL_VERSION,
            device_id: my_id.clone(),
            port: inner.listen_port().unwrap_or(DEFAULT_PORT),
        },
    )
    .await?;
    let hello: Hello = timeout(HANDSHAKE_TIMEOUT, read_plain(&mut reader)).await??;
    if hello.device_id != peer.device_id {
        bail!("device identity mismatch");
    }

    let (secure_r, secure_w) = timeout(
        HANDSHAKE_TIMEOUT,
        session::client_handshake(reader, writer, &key, &my_id, &peer.device_id),
    )
    .await??;
    inner.remember_addr(&peer.device_id, dialed);
    run_connection(inner, peer.device_id, secure_r, secure_w, stop).await;
    Ok(())
}

async fn browse_loop(
    inner: Arc<Inner>,
    events: mdns_sd::Receiver<ServiceEvent>,
    mut stop: watch::Receiver<bool>,
) {
    let my_id = inner.store.device_id();
    loop {
        tokio::select! {
            _ = stop.changed() => break,
            event = events.recv_async() => {
                let Ok(event) = event else { break };
                match event {
                    ServiceEvent::ServiceResolved(info) => {
                        let Some(found) = Discovered::from_service(&info) else { continue };
                        if found.device_id == my_id {
                            continue;
                        }
                        inner
                            .discovered
                            .lock()
                            .expect("sync poisoned")
                            .insert(found.device_id.clone(), found);
                    }
                    ServiceEvent::ServiceRemoved(_, fullname) => {
                        inner
                            .discovered
                            .lock()
                            .expect("sync poisoned")
                            .retain(|_, d| d.fullname != fullname);
                    }
                    _ => {}
                }
            }
        }
    }
}

/// 周期检查：对已配对、尚未连接的设备发起会话。
/// 地址优先用 mDNS 发现结果，发现不到时用上次记录的地址（mDNS 被网络拦截的兜底）。
/// 只由 `device_id` 字典序较小的一方拨号，避免双向同时连接。
async fn supervise_loop(inner: Arc<Inner>, mut stop: watch::Receiver<bool>) {
    let my_id = inner.store.device_id();
    let mut interval = tokio::time::interval(SUPERVISE_INTERVAL);
    loop {
        tokio::select! {
            _ = stop.changed() => break,
            _ = interval.tick() => {}
        }

        let targets: Vec<Discovered> = {
            let discovered = inner.discovered.lock().expect("sync poisoned");
            let peers = inner.peers.lock().expect("sync poisoned");
            inner
                .store
                .list()
                .into_iter()
                .filter(|device| my_id < device.id && !peers.contains_key(&device.id))
                .filter_map(|device| {
                    if let Some(found) = discovered.get(&device.id) {
                        return Some(found.clone());
                    }
                    let addr: SocketAddr = device.addr.as_deref()?.parse().ok()?;
                    Some(Discovered {
                        fullname: String::new(),
                        device_id: device.id,
                        addrs: vec![addr.ip()],
                        port: addr.port(),
                        pairing: false,
                    })
                })
                .collect()
        };

        for target in targets {
            let inner = inner.clone();
            let stop = stop.clone();
            tauri::async_runtime::spawn(async move {
                if let Err(err) = connect_session(inner, target.clone(), stop).await {
                    log::debug!("sync: connect to {} failed: {err:#}", target.device_id);
                }
            });
        }
    }
}

/// 已认证连接的主循环：注册到连接表，收发消息直到断开。
async fn run_connection<R, W>(
    inner: Arc<Inner>,
    peer_id: String,
    mut reader: SecureReader<R>,
    mut writer: SecureWriter<W>,
    mut stop: watch::Receiver<bool>,
) where
    R: AsyncRead + Unpin + Send + 'static,
    W: AsyncWrite + Unpin + Send + 'static,
{
    if Inner::is_stopped(&stop) {
        return;
    }

    let conn_id = inner.seq.fetch_add(1, Ordering::Relaxed);
    let (tx, mut rx) = mpsc::channel::<Message>(OUTBOX_DEPTH);
    inner.peers.lock().expect("sync poisoned").insert(
        peer_id.clone(),
        PeerConn {
            conn_id,
            tx: tx.clone(),
        },
    );
    inner.emit_devices();
    log::info!("sync: connected to {peer_id}");

    let writer_task = tauri::async_runtime::spawn(async move {
        while let Some(message) = rx.recv().await {
            if writer.send(&message).await.is_err() {
                break;
            }
        }
    });

    let ping_tx = tx.clone();
    let ping_task = tauri::async_runtime::spawn(async move {
        let mut interval = tokio::time::interval(PING_INTERVAL);
        loop {
            interval.tick().await;
            if ping_tx.send(Message::Ping).await.is_err() {
                break;
            }
        }
    });

    let mut images: HashMap<u64, (Vec<u8>, u64, u64)> = HashMap::new();
    loop {
        let message = tokio::select! {
            _ = stop.changed() => break,
            received = timeout(IDLE_TIMEOUT, reader.recv::<Message>()) => match received {
                Ok(Ok(message)) => message,
                Ok(Err(err)) => {
                    log::debug!("sync: connection to {peer_id} closed: {err:#}");
                    break;
                }
                Err(_) => {
                    log::info!("sync: connection to {peer_id} timed out");
                    break;
                }
            },
        };

        if !handle_message(&inner, &peer_id, &tx, message, &mut images).await {
            break;
        }
    }

    ping_task.abort();
    writer_task.abort();
    {
        let mut peers = inner.peers.lock().expect("sync poisoned");
        if peers.get(&peer_id).is_some_and(|p| p.conn_id == conn_id) {
            peers.remove(&peer_id);
        }
    }
    inner.fail_all_pending(Some(&peer_id));
    inner.emit_devices();
    log::info!("sync: disconnected from {peer_id}");
}

/// 处理一条消息；返回 `false` 表示应断开连接。
async fn handle_message(
    inner: &Arc<Inner>,
    peer_id: &str,
    tx: &mpsc::Sender<Message>,
    message: Message,
    images: &mut HashMap<u64, (Vec<u8>, u64, u64)>,
) -> bool {
    match message {
        Message::Ping => {
            let _ = tx.send(Message::Pong).await;
        }
        Message::Pong => {}
        Message::Unpair => {
            if let Err(err) = inner.store.remove(peer_id) {
                log::warn!("sync: remove unpaired device failed: {err:#}");
            }
            return false;
        }
        Message::ClipText {
            ts,
            plain,
            html,
            rtf,
        } => {
            let app = inner.app.clone();
            let ticket = apply::accept_remote(ts);
            log::info!("sync: received text from {peer_id} (ts {ts}, accepted {ticket:?})");
            tauri::async_runtime::spawn(async move {
                let result = match ticket {
                    Some(ticket) => apply::apply_text(&app, ticket, plain, html, rtf).await,
                    None => apply::apply_text_with(&app, plain, html, rtf, false).await,
                };
                if let Err(err) = result {
                    log::warn!("sync: apply text failed: {err:#}");
                }
            });
        }
        Message::ImageStart { ts, id, total } => {
            if total > apply::image_limit(&inner.app) {
                log::info!("sync: image of {total} bytes exceeds the local limit");
                return true;
            }
            images.insert(id, (Vec::with_capacity(total as usize), total, ts));
        }
        Message::ImageChunk { id, data } => {
            if let Some((buf, total, _)) = images.get_mut(&id) {
                if (buf.len() + data.len()) as u64 > *total {
                    images.remove(&id);
                } else {
                    buf.extend_from_slice(&data);
                }
            }
        }
        Message::ImageEnd { id } => {
            if let Some((png, total, ts)) = images.remove(&id) {
                if png.len() as u64 == total {
                    let app = inner.app.clone();
                    let ticket = apply::accept_remote(ts);
                    log::info!(
                        "sync: received image from {peer_id} (ts {ts}, accepted {ticket:?})"
                    );
                    tauri::async_runtime::spawn(async move {
                        let result = match ticket {
                            Some(ticket) => apply::apply_image(&app, ticket, png).await,
                            None => apply::apply_image_with(&app, png, false).await,
                        };
                        if let Err(err) = result {
                            log::warn!("sync: apply image failed: {err:#}");
                        }
                    });
                }
            }
        }
        Message::ClipFiles { ts, offer, entries } => {
            if !inner
                .app
                .state::<SettingsStore>()
                .snapshot()
                .sync
                .sync_files
            {
                return true;
            }
            let state = SyncState {
                inner: inner.clone(),
            };
            let Some(ticket) = apply::accept_remote(ts) else {
                log::info!(
                    "sync: ignored files from {peer_id} older than the local clipboard (ts {ts})"
                );
                return true;
            };
            remote_files::present(&inner.app, state, peer_id, &offer, entries, ticket);
        }
        Message::FetchFile {
            req,
            offer,
            index,
            offset,
            len,
        } => {
            let read = inner.read_offer_range(&offer, index, offset, len);
            let tx = tx.clone();
            tauri::async_runtime::spawn(async move {
                let reply = match read.await {
                    Ok(data) => Message::FileData { req, data },
                    Err(err) => Message::FileError {
                        req,
                        message: err.to_string(),
                    },
                };
                let _ = tx.send(reply).await;
            });
        }
        Message::FileData { req, data } => {
            if let Some(p) = inner.pending.lock().expect("sync poisoned").remove(&req) {
                let _ = p.tx.send(Ok(data));
            }
        }
        Message::FileError { req, message } => {
            if let Some(p) = inner.pending.lock().expect("sync poisoned").remove(&req) {
                let _ = p.tx.send(Err(message));
            }
        }
    }
    true
}

/// 把本机新复制的条目转换为消息并推送给所有在线设备。
async fn broadcast_item(inner: &Arc<Inner>, item: ClipboardItem, ts: u64) -> Result<()> {
    let settings = inner.app.state::<SettingsStore>().snapshot();
    if !settings.sync.enabled || item.is_sensitive {
        return Ok(());
    }

    let senders: Vec<mpsc::Sender<Message>> = inner
        .peers
        .lock()
        .expect("sync poisoned")
        .values()
        .map(|p| p.tx.clone())
        .collect();
    if senders.is_empty() {
        return Ok(());
    }

    match item.kind {
        ClipboardKind::Text if settings.sync.sync_text => {
            let message = text_message(&item, ts);
            for tx in &senders {
                let _ = tx.send(message.clone()).await;
            }
        }
        ClipboardKind::Image if settings.sync.sync_image => {
            let store = inner
                .app
                .state::<crate::clipboard::ImageStore>()
                .inner()
                .clone();
            let png = Arc::new(tokio::fs::read(store.origin_path(&item.content)).await?);
            let id = inner.seq.fetch_add(1, Ordering::Relaxed);
            for tx in senders {
                let png = png.clone();
                tauri::async_runtime::spawn(async move {
                    if tx
                        .send(Message::ImageStart {
                            ts,
                            id,
                            total: png.len() as u64,
                        })
                        .await
                        .is_err()
                    {
                        return;
                    }
                    for chunk in png.chunks(CHUNK_SIZE) {
                        if tx
                            .send(Message::ImageChunk {
                                id,
                                data: chunk.to_vec(),
                            })
                            .await
                            .is_err()
                        {
                            return;
                        }
                    }
                    let _ = tx.send(Message::ImageEnd { id }).await;
                });
            }
        }
        ClipboardKind::Files if settings.sync.sync_files => {
            let (offer, entries) = SyncState {
                inner: inner.clone(),
            }
            .prepare_files(&item)
            .await?;
            let message = Message::ClipFiles { ts, offer, entries };
            for tx in &senders {
                let _ = tx.send(message.clone()).await;
            }
        }
        _ => {}
    }
    Ok(())
}

/// 文本条目的（纯文本、HTML、RTF）三个表示。
pub fn text_parts(item: &ClipboardItem) -> (String, Option<String>, Option<String>) {
    let plain = || {
        item.search_text
            .clone()
            .unwrap_or_else(|| item.content.clone())
    };
    match item.sub_kind {
        Some(ClipboardSubKind::Html) => (plain(), Some(item.content.clone()), None),
        Some(ClipboardSubKind::Rtf) => (plain(), None, Some(item.content.clone())),
        _ => (item.content.clone(), None, None),
    }
}

fn text_message(item: &ClipboardItem, ts: u64) -> Message {
    match item.sub_kind {
        Some(ClipboardSubKind::Html) => Message::ClipText {
            ts,
            plain: item
                .search_text
                .clone()
                .unwrap_or_else(|| item.content.clone()),
            html: Some(item.content.clone()),
            rtf: None,
        },
        Some(ClipboardSubKind::Rtf) => Message::ClipText {
            ts,
            plain: item
                .search_text
                .clone()
                .unwrap_or_else(|| item.content.clone()),
            html: None,
            rtf: Some(item.content.clone()),
        },
        _ => Message::ClipText {
            ts,
            plain: item.content.clone(),
            html: None,
            rtf: None,
        },
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn manual_address_accepts_ip_with_or_without_port() {
        assert_eq!(
            parse_manual_address(" 192.168.1.20 "),
            Some("192.168.1.20:52526".parse().unwrap())
        );
        assert_eq!(
            parse_manual_address("10.0.0.5:6000"),
            Some("10.0.0.5:6000".parse().unwrap())
        );
        assert_eq!(parse_manual_address("not-an-ip"), None);
        assert_eq!(parse_manual_address("300.1.1.1"), None);
    }

    #[test]
    fn local_addresses_skip_loopback_and_tun_ranges() {
        for ip in local_ipv4_addresses() {
            let IpAddr::V4(v4) = ip else {
                panic!("only ipv4 expected");
            };
            assert!(!v4.is_loopback());
            assert!(!v4.is_link_local());
            assert!(!(v4.octets()[0] == 198 && (v4.octets()[1] & 0xFE) == 18));
        }
    }
}

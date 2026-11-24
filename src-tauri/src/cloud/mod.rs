//! 云同步：连接自托管的 EcoPasteServer（HTTPS + WebSocket，证书指纹固定），
//! 把本机新复制推送到云端、把其它设备的新内容实时应用到本机，并可拉取云端近期历史。
//!
//! 与局域网同步并存：写入剪贴板前经 `WritebackGuard` 抑制回环，同一内容经两条通道到达时由
//! `sync::apply` 里的短 TTL 哈希去重。文件不预传，只上传清单；粘贴时经服务器 WebSocket 向源设备按需拉取。

mod client;
#[cfg(test)]
mod e2e_tests;
mod pin;

use std::collections::HashMap;
use std::path::PathBuf;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::{Arc, Mutex};
use std::time::Duration;

use anyhow::{anyhow, Result};
use base64::Engine;
use futures_util::{SinkExt, StreamExt};
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use tauri::{AppHandle, Emitter, Manager};
use tokio::sync::{mpsc, oneshot, watch};
use tokio_tungstenite::tungstenite::client::IntoClientRequest;
use tokio_tungstenite::tungstenite::http::HeaderValue;
use tokio_tungstenite::tungstenite::{Error as WsError, Message as WsMessage};
use tokio_tungstenite::{connect_async_tls_with_config, Connector};

pub use client::{normalize_server_url, CloudError};
use client::{ws_url, Api, NewItem, ServerItem};

use crate::db::models::{ClipboardItem, ClipboardKind};
use crate::settings::SettingsStore;
use crate::sync::{self, SyncState};

/// 云端设备在 `remote_files` 中的伪对端前缀：`cloud:<origin device id>`。
pub const PEER_PREFIX: &str = "cloud:";
/// 云同步状态变化事件；前端收到后重新拉取状态。与前端 `src/constants/events.ts` 同步。
pub const STATUS_EVENT: &str = "cloud://status";

const FETCH_TIMEOUT: Duration = Duration::from_secs(60);
const MAX_BACKOFF: Duration = Duration::from_secs(30);
const STATE_FILE: &str = "cloud.json";

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(default, rename_all = "camelCase")]
struct Creds {
    server_url: String,
    fingerprint: String,
    username: String,
    token: String,
    device_id: i64,
    last_seq: i64,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub enum LinkState {
    Disconnected,
    Connecting,
    Connected,
    Error,
}

struct Inner {
    app: AppHandle,
    path: PathBuf,
    creds: Mutex<Creds>,
    link: Mutex<(LinkState, Option<String>)>,
    out: Mutex<Option<mpsc::Sender<String>>>,
    pending: Mutex<HashMap<u64, FetchReply>>,
    seq: AtomicU64,
    epoch: watch::Sender<u64>,
}

type FetchReply = oneshot::Sender<std::result::Result<Vec<u8>, String>>;

#[derive(Clone)]
pub struct CloudState(Arc<Inner>);

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct CloudStatus {
    pub enabled: bool,
    pub logged_in: bool,
    pub server_url: String,
    pub fingerprint: String,
    pub username: String,
    pub state: LinkState,
    pub error: Option<String>,
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ProbeResult {
    pub server_url: String,
    pub fingerprint: String,
}

/// 注册 [`CloudState`] 并启动后台连接循环。
pub fn init(app: &AppHandle) -> crate::core::Result<()> {
    let path = crate::core::paths::state_dir(app)?.join(STATE_FILE);
    let creds: Creds = std::fs::read_to_string(&path)
        .ok()
        .and_then(|text| serde_json::from_str(&text).ok())
        .unwrap_or_default();

    let state = CloudState::build(app, path, creds);
    app.manage(state.clone());

    tauri::async_runtime::spawn(run(state.0.clone()));
    Ok(())
}

/// 设置变化后调用（开关云同步）。
pub fn apply_settings(app: &AppHandle) {
    if let Some(state) = app.try_state::<CloudState>() {
        state.0.epoch.send_modify(|n| *n += 1);
    }
}

/// 本机新复制了一条内容：按设置上传到云端。
pub fn broadcast_item(app: &AppHandle, item: &ClipboardItem) {
    let Some(state) = app.try_state::<CloudState>() else {
        return;
    };
    let inner = state.0.clone();
    let item = item.clone();
    tauri::async_runtime::spawn(async move {
        if let Err(err) = upload_item(&inner, &item).await {
            log::warn!("cloud: upload failed: {err:#}");
        }
    });
}

impl Inner {
    fn emit_status(&self) {
        if let Err(err) = self.app.emit(STATUS_EVENT, ()) {
            log::warn!("emit {STATUS_EVENT} failed: {err}");
        }
    }

    fn set_link(&self, state: LinkState, error: Option<String>) {
        *self.link.lock().expect("cloud poisoned") = (state, error);
        self.emit_status();
    }

    fn creds(&self) -> Creds {
        self.creds.lock().expect("cloud poisoned").clone()
    }

    fn save_creds(&self, creds: Creds) {
        match serde_json::to_vec_pretty(&creds) {
            Ok(json) => {
                if let Some(dir) = self.path.parent() {
                    let _ = std::fs::create_dir_all(dir);
                }
                let tmp = self.path.with_extension("json.tmp");
                if let Err(err) =
                    std::fs::write(&tmp, json).and_then(|()| std::fs::rename(&tmp, &self.path))
                {
                    log::warn!("cloud: persist state failed: {err}");
                }
            }
            Err(err) => log::warn!("cloud: serialize state failed: {err}"),
        }
        *self.creds.lock().expect("cloud poisoned") = creds;
    }

    fn enabled(&self) -> bool {
        self.app.state::<SettingsStore>().snapshot().cloud.enabled
    }

    fn api(&self) -> Result<Api> {
        let creds = self.creds();
        if creds.token.is_empty() {
            return Err(CloudError::NotSignedIn.into());
        }
        Ok(Api::new(
            &creds.server_url,
            Some(creds.fingerprint),
            Some(creds.token),
        )?
        .0)
    }

    fn fail_pending(&self, reason: &str) {
        for (_, tx) in self.pending.lock().expect("cloud poisoned").drain() {
            let _ = tx.send(Err(reason.to_owned()));
        }
    }

    fn sign_out_local(&self) {
        let mut creds = self.creds();
        creds.token.clear();
        creds.device_id = 0;
        creds.last_seq = 0;
        self.save_creds(creds);
    }
}

impl CloudState {
    fn build(app: &AppHandle, path: PathBuf, creds: Creds) -> Self {
        let (epoch, _) = watch::channel(0);
        Self(Arc::new(Inner {
            app: app.clone(),
            path,
            creds: Mutex::new(creds),
            link: Mutex::new((LinkState::Disconnected, None)),
            out: Mutex::new(None),
            pending: Mutex::new(HashMap::new()),
            seq: AtomicU64::new(1),
            epoch,
        }))
    }

    pub fn status(&self) -> CloudStatus {
        let inner = &self.0;
        let creds = inner.creds();
        let (state, error) = inner.link.lock().expect("cloud poisoned").clone();
        CloudStatus {
            enabled: inner.enabled(),
            logged_in: !creds.token.is_empty(),
            server_url: creds.server_url,
            fingerprint: creds.fingerprint,
            username: creds.username,
            state,
            error,
        }
    }

    /// 读取服务器证书指纹（不信任），供用户核对后再登录。
    pub async fn probe(&self, server_url: &str) -> Result<ProbeResult> {
        let base = normalize_server_url(server_url)?;
        let fingerprint = Api::probe(&base).await?;
        Ok(ProbeResult {
            server_url: base,
            fingerprint,
        })
    }

    pub async fn login(
        &self,
        server_url: &str,
        fingerprint: &str,
        username: &str,
        password: &str,
    ) -> Result<CloudStatus> {
        let inner = &self.0;
        let base = normalize_server_url(server_url)?;
        let (api, _) = Api::new(&base, Some(fingerprint.to_owned()), None)?;

        let settings = inner.app.state::<SettingsStore>().snapshot();
        let device_name = {
            let name = settings.sync.device_name.trim().to_owned();
            if name.is_empty() {
                sync::identity::system_hostname()
            } else {
                name
            }
        };
        let login = api
            .login(
                username.trim(),
                password,
                &device_name,
                sync::identity::current_platform(),
            )
            .await?;

        let authed = Api::new(
            &base,
            Some(fingerprint.to_owned()),
            Some(login.token.clone()),
        )?
        .0;
        let latest = authed.me().await.map(|me| me.latest_seq).unwrap_or(0);

        inner.save_creds(Creds {
            server_url: base,
            fingerprint: fingerprint.to_owned(),
            username: username.trim().to_owned(),
            token: login.token,
            device_id: login.device_id,
            last_seq: latest,
        });
        inner.epoch.send_modify(|n| *n += 1);
        inner.emit_status();
        Ok(self.status())
    }

    pub async fn logout(&self) {
        let inner = &self.0;
        if let Ok(api) = inner.api() {
            let _ = api.logout().await;
        }
        inner.sign_out_local();
        inner.epoch.send_modify(|n| *n += 1);
        inner.emit_status();
    }

    /// 拉取云端最近的文本 / 图片到本机历史（不改动系统剪贴板）。返回处理的条数。
    pub async fn pull_recent(&self) -> Result<usize> {
        let inner = &self.0;
        let api = inner.api()?;
        let limit = inner
            .app
            .state::<SettingsStore>()
            .snapshot()
            .cloud
            .pull_recent
            .clamp(1, 200);
        let mut items = api.history(limit).await?;
        items.reverse();

        let mut applied = 0;
        for item in items {
            if apply_item(inner, &api, &item, false).await.is_ok() {
                applied += 1;
            }
        }
        Ok(applied)
    }
}

/// 后台主循环：按设置与登录状态维持 WebSocket 连接，断线退避重连。
async fn run(inner: Arc<Inner>) {
    let mut epoch = inner.epoch.subscribe();
    let mut backoff = Duration::from_secs(1);

    loop {
        let creds = inner.creds();
        if !inner.enabled() || creds.token.is_empty() {
            inner.set_link(LinkState::Disconnected, None);
            if epoch.changed().await.is_err() {
                return;
            }
            backoff = Duration::from_secs(1);
            continue;
        }

        inner.set_link(LinkState::Connecting, None);
        match session(&inner, &creds, &mut epoch).await {
            Ok(()) => backoff = Duration::from_secs(1),
            Err(err) => {
                if matches!(
                    err.downcast_ref::<CloudError>(),
                    Some(CloudError::NotSignedIn)
                ) {
                    log::warn!("cloud: token rejected, signing out");
                    inner.sign_out_local();
                    inner.set_link(LinkState::Error, Some(CloudError::NotSignedIn.to_string()));
                    continue;
                }
                log::info!("cloud: connection ended: {err:#}");
                inner.set_link(LinkState::Error, Some(err.to_string()));
            }
        }

        tokio::select! {
            () = tokio::time::sleep(backoff) => {}
            changed = epoch.changed() => {
                if changed.is_err() { return; }
                backoff = Duration::from_secs(1);
                continue;
            }
        }
        backoff = (backoff * 2).min(MAX_BACKOFF);
    }
}

#[derive(Debug, Deserialize, Default)]
#[serde(default)]
struct WsMsg {
    t: String,
    item: Value,
    req: u64,
    from: i64,
    to: i64,
    offer: String,
    index: u32,
    offset: u64,
    len: u32,
    data: String,
    err: String,
}

async fn session(
    inner: &Arc<Inner>,
    creds: &Creds,
    epoch: &mut watch::Receiver<u64>,
) -> Result<()> {
    let (config, _) = pin::client_config(Some(creds.fingerprint.clone()));
    let url = ws_url(
        &creds.server_url,
        &format!("/api/ws?since={}", creds.last_seq),
    );
    let mut request = url.into_client_request()?;
    request.headers_mut().insert(
        "Authorization",
        HeaderValue::from_str(&format!("Bearer {}", creds.token))?,
    );

    let (ws, _) =
        connect_async_tls_with_config(request, None, true, Some(Connector::Rustls(config)))
            .await
            .map_err(|err| match err {
                WsError::Http(resp) if resp.status().as_u16() == 401 => {
                    anyhow::Error::new(CloudError::NotSignedIn)
                }
                WsError::Http(resp) => anyhow!("server answered {}", resp.status()),
                other if format!("{other:?}").contains("fingerprint does not match") => {
                    anyhow::Error::new(CloudError::PinMismatch)
                }
                _ => anyhow::Error::new(CloudError::Unreachable),
            })?;

    log::info!("cloud: connected to {}", creds.server_url);
    inner.set_link(LinkState::Connected, None);

    let (mut sink, mut stream) = ws.split();
    let (tx, mut rx) = mpsc::channel::<String>(64);
    *inner.out.lock().expect("cloud poisoned") = Some(tx);

    let result = loop {
        tokio::select! {
            changed = epoch.changed() => {
                let _ = changed;
                break Ok(());
            }
            outgoing = rx.recv() => {
                let Some(text) = outgoing else { break Ok(()) };
                if let Err(err) = sink.send(WsMessage::text(text)).await {
                    break Err(anyhow!(err));
                }
            }
            frame = stream.next() => match frame {
                Some(Ok(WsMessage::Text(text))) => handle_text(inner, text.as_str()),
                Some(Ok(WsMessage::Close(_))) | None => break Err(anyhow!("server closed the connection")),
                Some(Ok(_)) => {}
                Some(Err(err)) => break Err(anyhow!(err)),
            },
        }
    };

    *inner.out.lock().expect("cloud poisoned") = None;
    inner.fail_pending("cloud connection closed");
    let _ = sink.close().await;
    result
}

fn handle_text(inner: &Arc<Inner>, text: &str) {
    let Ok(msg) = serde_json::from_str::<WsMsg>(text) else {
        return;
    };

    match msg.t.as_str() {
        "item" => {
            let Ok(item) = serde_json::from_value::<ServerItem>(msg.item) else {
                return;
            };
            let inner = inner.clone();
            tauri::async_runtime::spawn(async move {
                let mut creds = inner.creds();
                if item.seq > creds.last_seq {
                    creds.last_seq = item.seq;
                    inner.save_creds(creds);
                }
                let Ok(api) = inner.api() else { return };
                if let Err(err) = apply_item(&inner, &api, &item, true).await {
                    log::warn!("cloud: apply item {} failed: {err:#}", item.seq);
                }
            });
        }
        "fetch" => {
            let inner = inner.clone();
            tauri::async_runtime::spawn(async move {
                let state = inner.app.state::<SyncState>().inner().clone();
                let reply = match state
                    .read_offer_range(&msg.offer, msg.index, msg.offset, msg.len)
                    .await
                {
                    Ok(data) => json!({
                        "t": "fdata", "req": msg.req, "to": msg.from,
                        "data": base64::engine::general_purpose::STANDARD.encode(data),
                    }),
                    Err(err) => json!({
                        "t": "fdata", "req": msg.req, "to": msg.from, "err": err.to_string(),
                    }),
                };
                send_raw(&inner, reply.to_string()).await;
            });
        }
        "fdata" => {
            if let Some(tx) = inner
                .pending
                .lock()
                .expect("cloud poisoned")
                .remove(&msg.req)
            {
                let result = if msg.err.is_empty() {
                    base64::engine::general_purpose::STANDARD
                        .decode(msg.data)
                        .map_err(|err| err.to_string())
                } else {
                    Err(msg.err)
                };
                let _ = tx.send(result);
            }
            let _ = msg.to;
        }
        _ => {}
    }
}

async fn send_raw(inner: &Inner, text: String) {
    let tx = inner.out.lock().expect("cloud poisoned").clone();
    if let Some(tx) = tx {
        let _ = tx.send(text).await;
    }
}

/// 向源设备（经服务器中转）拉取文件的一段数据。
pub async fn fetch_file_range(
    app: &AppHandle,
    origin: &str,
    offer: &str,
    index: u32,
    offset: u64,
    len: u32,
) -> Result<Vec<u8>> {
    let state = app
        .try_state::<CloudState>()
        .ok_or_else(|| anyhow!("cloud sync is not available"))?;
    let inner = &state.0;
    let origin: i64 = origin
        .parse()
        .map_err(|_| anyhow!("invalid origin device"))?;

    let req = inner.seq.fetch_add(1, Ordering::Relaxed);
    let (tx, rx) = oneshot::channel();
    inner
        .pending
        .lock()
        .expect("cloud poisoned")
        .insert(req, tx);

    let sent = inner.out.lock().expect("cloud poisoned").clone();
    let Some(sent) = sent else {
        inner.pending.lock().expect("cloud poisoned").remove(&req);
        return Err(CloudError::Unreachable.into());
    };
    let message = json!({
        "t": "fetch", "req": req, "to": origin, "offer": offer,
        "index": index, "offset": offset, "len": len,
    });
    if sent.send(message.to_string()).await.is_err() {
        inner.pending.lock().expect("cloud poisoned").remove(&req);
        return Err(CloudError::Unreachable.into());
    }

    match tokio::time::timeout(FETCH_TIMEOUT, rx).await {
        Ok(Ok(Ok(data))) => Ok(data),
        Ok(Ok(Err(message))) => Err(anyhow!(message)),
        Ok(Err(_)) => Err(anyhow!("cloud connection closed")),
        Err(_) => {
            inner.pending.lock().expect("cloud poisoned").remove(&req);
            Err(anyhow!("transfer timed out"))
        }
    }
}

#[derive(Deserialize, Default)]
#[serde(default)]
struct TextPayload {
    plain: String,
    html: Option<String>,
    rtf: Option<String>,
}

#[derive(Deserialize, Default)]
#[serde(default)]
struct ImagePayload {
    blob: String,
}

#[derive(Serialize, Deserialize, Default)]
#[serde(default, rename_all = "camelCase")]
struct FileEntryPayload {
    rel_path: String,
    is_dir: bool,
    size: u64,
}

#[derive(Deserialize, Default)]
#[serde(default)]
struct FilesPayload {
    offer: String,
    entries: Vec<FileEntryPayload>,
}

/// 应用一条云端条目。`live` 为 true 时写系统剪贴板（实时），否则仅入历史（拉取历史）。
async fn apply_item(inner: &Arc<Inner>, api: &Api, item: &ServerItem, live: bool) -> Result<()> {
    if item.origin == inner.creds().device_id {
        return Ok(());
    }

    match item.kind.as_str() {
        "text" => {
            let p: TextPayload = serde_json::from_value(item.payload.clone())?;
            sync::apply_text_with(&inner.app, p.plain, p.html, p.rtf, live).await
        }
        "image" => {
            let p: ImagePayload = serde_json::from_value(item.payload.clone())?;
            if !crate::cloud::client::is_hash(&p.blob) {
                return Err(anyhow!("image item without a blob"));
            }
            let png = api.get_blob(&p.blob).await?;
            if png.len() as u64 > sync::image_limit(&inner.app) {
                return Err(anyhow!("image exceeds the local size limit"));
            }
            sync::apply_image_with(&inner.app, png, live).await
        }
        "files" if live => {
            if !inner
                .app
                .state::<SettingsStore>()
                .snapshot()
                .sync
                .sync_files
            {
                return Ok(());
            }
            let p: FilesPayload = serde_json::from_value(item.payload.clone())?;
            let entries = p
                .entries
                .into_iter()
                .map(|e| sync::FileEntry {
                    rel_path: e.rel_path,
                    is_dir: e.is_dir,
                    size: e.size,
                })
                .collect();
            let state = inner.app.state::<SyncState>().inner().clone();
            sync::present_remote_files(
                &inner.app,
                state,
                &format!("{PEER_PREFIX}{}", item.origin),
                &p.offer,
                entries,
            );
            Ok(())
        }
        _ => Ok(()),
    }
}

async fn upload_item(inner: &Arc<Inner>, item: &ClipboardItem) -> Result<()> {
    let settings = inner.app.state::<SettingsStore>().snapshot();
    if !settings.cloud.enabled || item.is_sensitive || inner.creds().token.is_empty() {
        return Ok(());
    }
    let api = inner.api()?;

    match item.kind {
        ClipboardKind::Text if settings.sync.sync_text => {
            let (plain, html, rtf) = sync::text_parts(item);
            api.post_item(&NewItem {
                hash: &item.content_hash,
                kind: "text",
                size: item.content.len() as i64,
                blob: None,
                payload: json!({ "plain": plain, "html": html, "rtf": rtf }),
            })
            .await?;
        }
        ClipboardKind::Image if settings.sync.sync_image => {
            let store = inner
                .app
                .state::<crate::clipboard::ImageStore>()
                .inner()
                .clone();
            let png = tokio::fs::read(store.origin_path(&item.content)).await?;
            let hash = api.put_blob(png).await?;
            api.post_item(&NewItem {
                hash: &item.content_hash,
                kind: "image",
                size: item.size.unwrap_or(0),
                blob: Some(&hash),
                payload: json!({ "width": item.width, "height": item.height, "blob": hash }),
            })
            .await?;
        }
        ClipboardKind::Files if settings.sync.sync_files => {
            let state = inner.app.state::<SyncState>().inner().clone();
            let (offer, entries) = state.prepare_files(item).await?;
            let entries: Vec<FileEntryPayload> = entries
                .into_iter()
                .map(|e| FileEntryPayload {
                    rel_path: e.rel_path,
                    is_dir: e.is_dir,
                    size: e.size,
                })
                .collect();
            api.post_item(&NewItem {
                hash: &item.content_hash,
                kind: "files",
                size: 0,
                blob: None,
                payload: json!({ "offer": offer, "entries": entries }),
            })
            .await?;
        }
        _ => {}
    }
    Ok(())
}

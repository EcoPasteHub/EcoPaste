//! 接收侧：把对端发来的内容写入本机剪贴板并入库。
//!
//! 写剪贴板前先向 [`WritebackGuard`] 登记哈希，监听线程读到同内容时会跳过，
//! 因此收到的内容不会被再次入库，也不会再广播出去（星型同步天然无回环）。

use std::io::Cursor;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Arc;

use anyhow::{anyhow, Result};
use tauri::{AppHandle, Manager};

use crate::clipboard::{
    build_item_with_settings, persist_and_notify, write_to_clipboard_once, ClipboardPayload,
    ImagePayload, ImageStore, TextPayload, WritebackGuard,
};
use crate::db::models::ClipboardItem;
use crate::db::DatabaseState;
use crate::settings::SettingsStore;

/// 接收图片的绝对上限（`capture.max_image_mb = 0` 不限制时兜底）。
pub const HARD_IMAGE_LIMIT: u64 = 512 * 1024 * 1024;

/// 测试用：`Some` 时把「写剪贴板」改为记录条目，让无剪贴板权限的环境也能验证同步链路。
#[cfg(test)]
pub static TEST_SINK: std::sync::Mutex<Option<Vec<ClipboardItem>>> = std::sync::Mutex::new(None);

const WRITE_RETRY_ATTEMPTS: u32 = 40;
const WRITE_RETRY_MAX_DELAY: std::time::Duration = std::time::Duration::from_millis(500);
const WRITE_RETRY_BASE: std::time::Duration = std::time::Duration::from_millis(30);

/// 最新一次剪贴板写入意图的序号。远端内容到达时领取新序号，本机发生新复制时也会推进；
/// 序号已被推进的写入视为过期并放弃，防止剪贴板忙时退避重试的旧内容晚于新内容落地、覆盖新内容。
static LATEST_WRITE: AtomicU64 = AtomicU64::new(0);

/// 为一次远端写入领取序号。
pub(crate) fn claim_write() -> u64 {
    LATEST_WRITE.fetch_add(1, Ordering::SeqCst) + 1
}

/// 领取 `ticket` 之后是否已有更新的远端内容或本机复制。
pub(crate) fn is_superseded(ticket: u64) -> bool {
    LATEST_WRITE.load(Ordering::SeqCst) != ticket
}

/// 本机发生了新的复制：作废尚未完成的远端写入。
pub fn supersede_remote_writes() {
    LATEST_WRITE.fetch_add(1, Ordering::SeqCst);
}

/// 本机允许接收的图片字节上限。
pub fn image_limit(app: &AppHandle) -> u64 {
    app.state::<SettingsStore>()
        .snapshot()
        .clipboard
        .capture
        .max_image_bytes()
        .map_or(HARD_IMAGE_LIMIT, |limit| limit.min(HARD_IMAGE_LIMIT))
}

/// 近期已应用的远端内容哈希：局域网与云端同时到达同一条内容时只处理一次。
static RECENT: std::sync::Mutex<Vec<(String, std::time::Instant)>> =
    std::sync::Mutex::new(Vec::new());
const RECENT_TTL: std::time::Duration = std::time::Duration::from_secs(20);

/// 记录哈希并返回是否在 TTL 内已见过。
fn seen_recently(hash: &str) -> bool {
    let mut recent = RECENT.lock().expect("recent poisoned");
    recent.retain(|(_, at)| at.elapsed() < RECENT_TTL);
    if recent.iter().any(|(h, _)| h == hash) {
        return true;
    }
    recent.push((hash.to_owned(), std::time::Instant::now()));
    false
}

/// 局域网收到的文本：`ticket` 须在消息到达时（按到达顺序）用 [`claim_write`] 领取，
/// 这样并发处理的多条消息仍以到达顺序决定谁是「最新」。
pub async fn apply_text(
    app: &AppHandle,
    ticket: u64,
    plain: String,
    html: Option<String>,
    rtf: Option<String>,
) -> Result<()> {
    apply_text_ticketed(app, ticket, plain, html, rtf, true).await
}

/// `to_clipboard = false` 时只入历史、不写系统剪贴板（拉取云端历史用）。
pub async fn apply_text_with(
    app: &AppHandle,
    plain: String,
    html: Option<String>,
    rtf: Option<String>,
    to_clipboard: bool,
) -> Result<()> {
    let ticket = if to_clipboard { claim_write() } else { 0 };
    apply_text_ticketed(app, ticket, plain, html, rtf, to_clipboard).await
}

async fn apply_text_ticketed(
    app: &AppHandle,
    ticket: u64,
    plain: String,
    html: Option<String>,
    rtf: Option<String>,
    to_clipboard: bool,
) -> Result<()> {
    if !app.state::<SettingsStore>().snapshot().sync.sync_text {
        return Ok(());
    }

    apply_payload(
        app,
        ClipboardPayload::Text(TextPayload {
            text: plain,
            html,
            rtf,
        }),
        to_clipboard,
        ticket,
    )
    .await
}

/// 局域网收到的图片：`ticket` 的领取时机同 [`apply_text`]。
pub async fn apply_image(app: &AppHandle, ticket: u64, png: Vec<u8>) -> Result<()> {
    apply_image_ticketed(app, ticket, png, true).await
}

pub async fn apply_image_with(app: &AppHandle, png: Vec<u8>, to_clipboard: bool) -> Result<()> {
    let ticket = if to_clipboard { claim_write() } else { 0 };
    apply_image_ticketed(app, ticket, png, to_clipboard).await
}

async fn apply_image_ticketed(
    app: &AppHandle,
    ticket: u64,
    png: Vec<u8>,
    to_clipboard: bool,
) -> Result<()> {
    if !app.state::<SettingsStore>().snapshot().sync.sync_image {
        return Ok(());
    }

    let (width, height) = image::ImageReader::new(Cursor::new(&png))
        .with_guessed_format()?
        .into_dimensions()
        .map_err(|err| anyhow!("invalid image: {err}"))?;

    apply_payload(
        app,
        ClipboardPayload::Image(ImagePayload {
            bytes: png,
            width,
            height,
        }),
        to_clipboard,
        ticket,
    )
    .await
}

async fn apply_payload(
    app: &AppHandle,
    payload: ClipboardPayload,
    to_clipboard: bool,
    ticket: u64,
) -> Result<()> {
    let settings = app.state::<SettingsStore>().snapshot();
    let store = app.state::<ImageStore>().inner().clone();

    let Some(item) = build_item_with_settings(
        &store,
        &payload,
        &settings.clipboard.capture,
        &settings.clipboard.sensitive,
        settings.clipboard.content.copy_plain,
    )?
    else {
        return Ok(());
    };

    if to_clipboard && seen_recently(&item.content_hash) {
        log::info!("sync: ignored a repeated received item (ticket {ticket})");
        return Ok(());
    }

    if to_clipboard {
        // 写入失败（如剪贴板被长期占用）时仍入历史，用户可从历史面板粘贴。
        if let Err(err) = write_item_with_retry(app, &store, &item, ticket).await {
            log::warn!("sync: writing received item to the clipboard failed: {err:#}");
        }
    }

    let pool = app.state::<DatabaseState>().pool().await;
    persist_and_notify(app, &pool, &item, None).await?;
    Ok(())
}

/// 其它程序（剪贴板监听器、历史面板等）可能短暂占用剪贴板导致打开被拒绝，退避重试。
/// 重试期间若 `ticket` 已过期（来了更新的内容），放弃写入，避免旧内容覆盖新内容。
async fn write_item_with_retry(
    app: &AppHandle,
    store: &ImageStore,
    item: &ClipboardItem,
    ticket: u64,
) -> Result<()> {
    let mut delay = WRITE_RETRY_BASE;
    let mut attempt = 1;

    loop {
        if is_superseded(ticket) {
            log::info!("sync: dropped a received item superseded by newer clipboard content (ticket {ticket})");
            return Ok(());
        }

        match write_item(app, store, item) {
            Ok(()) => return Ok(()),
            Err(err) if attempt >= WRITE_RETRY_ATTEMPTS => return Err(err),
            Err(err) => {
                log::debug!("sync: clipboard busy (attempt {attempt}): {err}");
                tokio::time::sleep(delay).await;
                delay = delay.saturating_mul(2).min(WRITE_RETRY_MAX_DELAY);
                attempt += 1;
            }
        }
    }
}

/// 同步段内写系统剪贴板（平台剪贴板句柄不可跨 await 持有）。
fn write_item(app: &AppHandle, store: &ImageStore, item: &ClipboardItem) -> Result<()> {
    #[cfg(test)]
    if let Some(sink) = TEST_SINK.lock().expect("test sink poisoned").as_mut() {
        sink.push(item.clone());
        return Ok(());
    }

    let guard = app.state::<Arc<WritebackGuard>>();
    write_to_clipboard_once(store, guard.inner().as_ref(), item, false)?;
    Ok(())
}

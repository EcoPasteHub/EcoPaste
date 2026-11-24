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

/// 本机已知最新一条内容（本机复制或已采用的远端内容）的逻辑时间戳，单位毫秒。
///
/// 混合逻辑时钟：本机复制取 `max(墙钟, 已知最新 + 1)`，因此看过对端内容之后的复制一定更新；
/// 对端内容只有时间戳更大才会被采用，较旧的到达不会覆盖较新的剪贴板。
static LATEST_STAMP: AtomicU64 = AtomicU64::new(0);

/// 对端时间戳允许超前本机墙钟的上限，防止时钟错乱的设备把「最新」推到遥远的未来。
const MAX_FUTURE_SKEW_MS: u64 = 10 * 60 * 1000;

fn wall_clock_ms() -> u64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map_or(0, |d| d.as_millis() as u64)
}

/// 取下一个本机时间戳并把它登记为最新。
pub(crate) fn claim_write() -> u64 {
    let now = wall_clock_ms();
    let mut current = LATEST_STAMP.load(Ordering::SeqCst);

    loop {
        let next = now.max(current + 1);
        match LATEST_STAMP.compare_exchange(current, next, Ordering::SeqCst, Ordering::SeqCst) {
            Ok(_) => return next,
            Err(actual) => current = actual,
        }
    }
}

fn clamp_future(ts: u64, now: u64) -> u64 {
    ts.min(now + MAX_FUTURE_SKEW_MS)
}

/// 对端内容带着时间戳 `ts` 到达：比本机已知最新更新则登记并返回采用的时间戳，否则返回 `None`。
pub(crate) fn accept_remote(ts: u64) -> Option<u64> {
    let ts = clamp_future(ts, wall_clock_ms());
    let mut current = LATEST_STAMP.load(Ordering::SeqCst);

    loop {
        if ts <= current {
            return None;
        }
        match LATEST_STAMP.compare_exchange(current, ts, Ordering::SeqCst, Ordering::SeqCst) {
            Ok(_) => return Some(ts),
            Err(actual) => current = actual,
        }
    }
}

/// 登记之后是否已有更新的内容（本机新复制或更晚的远端内容）。
pub(crate) fn is_superseded(stamp: u64) -> bool {
    LATEST_STAMP.load(Ordering::SeqCst) != stamp
}

/// 本机发生了新的复制：登记并返回它的时间戳（随广播发出），同时作废尚未完成的远端写入。
pub fn stamp_local_copy() -> u64 {
    claim_write()
}

/// 近期出现在本机剪贴板上的内容哈希（远端写入的与本机复制的）。
///
/// 微信输入法的跨设备剪贴板同步会在对端写入后，把这些内容按序重放回本机剪贴板；
/// 若把重放当作新复制再广播，内容会在两台设备间来回弹跳（A、B 交替互相覆盖）。
static RECENT_CONTENT: std::sync::Mutex<Vec<(String, std::time::Instant)>> =
    std::sync::Mutex::new(Vec::new());

/// 重放判定窗口：覆盖输入法经云端往返所需的时间。
const REPLAY_WINDOW: std::time::Duration = std::time::Duration::from_secs(15);

/// 登记一条刚出现在本机剪贴板上的内容。
pub fn remember_content(content_hash: &str) {
    let mut recent = RECENT_CONTENT.lock().expect("recent content poisoned");
    recent.retain(|(_, at)| at.elapsed() < REPLAY_WINDOW);
    recent.push((content_hash.to_owned(), std::time::Instant::now()));
}

/// 该内容近期是否已出现在本机剪贴板上。
pub fn is_recent_content(content_hash: &str) -> bool {
    let mut recent = RECENT_CONTENT.lock().expect("recent content poisoned");
    recent.retain(|(_, at)| at.elapsed() < REPLAY_WINDOW);
    recent.iter().any(|(hash, _)| hash == content_hash)
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

/// 局域网收到的文本：`ticket` 是 [`accept_remote`] 在消息到达时登记的时间戳。
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

/// 局域网收到的图片：`ticket` 同 [`apply_text`]。
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
        if !is_superseded(ticket) {
            remember_content(&item.content_hash);
        }

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

#[cfg(test)]
mod clock_tests {
    use super::*;

    #[test]
    fn newest_stamp_wins_and_older_arrivals_are_rejected() {
        let local = stamp_local_copy();

        assert_eq!(accept_remote(local), None);
        assert_eq!(accept_remote(local.saturating_sub(5_000)), None);

        let newer = accept_remote(wall_clock_ms() + 60_000).expect("newer content is accepted");
        assert!(is_superseded(local));
        assert!(!is_superseded(newer));
        assert_eq!(accept_remote(newer), None);

        // 看过对端内容之后的本机复制必然更新。
        assert!(stamp_local_copy() > newer);
    }

    #[test]
    fn far_future_stamps_are_clamped() {
        assert_eq!(clamp_future(u64::MAX, 1_000), 1_000 + MAX_FUTURE_SKEW_MS);
        assert_eq!(clamp_future(500, 1_000), 500);
    }
}

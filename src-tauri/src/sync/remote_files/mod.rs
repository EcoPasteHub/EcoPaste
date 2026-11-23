//! 接收侧「虚拟文件」剪贴板：对端复制的文件只放清单，用户在任意应用粘贴时才按需拉取内容。
//!
//! - Windows：自实现 `IDataObject`（`FileGroupDescriptorW` + 按需 `IStream` 的 `FileContents`）；
//! - macOS：TODO，需要 `NSFilePromiseProvider`（当前环境无法验证），暂时仅记录日志。

#[cfg(target_os = "windows")]
mod windows;

use std::sync::Mutex;
use std::time::{Duration, Instant};

use tauri::AppHandle;

use super::manager::SyncState;
use super::protocol::FileEntry;

/// 测试用：被捕获的虚拟文件 offer，替代真实剪贴板放置。
#[cfg(test)]
#[allow(dead_code)]
pub struct Captured {
    pub state: SyncState,
    pub peer_id: String,
    pub offer: String,
    pub entries: Vec<FileEntry>,
}

/// 测试用：`Some` 时 [`present`] 只记录 offer，不触碰系统剪贴板。
#[cfg(test)]
pub static CAPTURED: std::sync::Mutex<Option<Vec<Captured>>> = std::sync::Mutex::new(None);

#[cfg(all(test, target_os = "windows"))]
pub use windows::create_data_object;

/// 近期收到的远端文件清单（各文件字节数）在多久内用于识别回声。
const ECHO_TTL: Duration = Duration::from_secs(120);
const MAX_RECENT_OFFERS: usize = 8;

/// 近期收到的远端文件清单中各文件的字节数。
static RECENT_OFFERS: Mutex<Vec<(Instant, Vec<u64>)>> = Mutex::new(Vec::new());

fn remember_offer(entries: &[FileEntry]) {
    let sizes: Vec<u64> = entries
        .iter()
        .filter(|entry| !entry.is_dir)
        .map(|entry| entry.size)
        .collect();
    if sizes.is_empty() {
        return;
    }

    let mut recent = RECENT_OFFERS.lock().expect("recent offers poisoned");
    recent.retain(|(at, _)| at.elapsed() < ECHO_TTL);
    if recent.len() >= MAX_RECENT_OFFERS {
        recent.remove(0);
    }
    recent.push((Instant::now(), sizes));
}

/// 本机捕获到的文件是否只是近期收到的远端虚拟文件被其它程序落盘后重新放回剪贴板。
///
/// 输入法剪贴板（如微信输入法）、远程桌面工具会读取虚拟文件内容、另存为临时文件再放回剪贴板，
/// 文件名与路径都变了，只有字节数不变。若把它当本机新复制再广播，两端会互相把文件推回去，
/// 反复覆盖用户剪贴板。这里按「所有文件的字节数都能在同一份近期清单里对上」判定为回声。
pub fn is_echo(paths: &[&str]) -> bool {
    let mut sizes = Vec::with_capacity(paths.len());
    for path in paths {
        match std::fs::metadata(path) {
            Ok(meta) if meta.is_file() => sizes.push(meta.len()),
            _ => return false,
        }
    }

    sizes_match_recent_offer(&sizes)
}

fn sizes_match_recent_offer(sizes: &[u64]) -> bool {
    if sizes.is_empty() {
        return false;
    }

    let mut recent = RECENT_OFFERS.lock().expect("recent offers poisoned");
    recent.retain(|(at, _)| at.elapsed() < ECHO_TTL);
    recent.iter().any(|(_, offered)| {
        let mut remaining = offered.clone();
        sizes.iter().all(|size| {
            let Some(index) = remaining.iter().position(|offered| offered == size) else {
                return false;
            };
            remaining.swap_remove(index);
            true
        })
    })
}

/// 把对端文件清单呈现为本机剪贴板上的虚拟文件。
pub fn present(
    app: &AppHandle,
    state: SyncState,
    peer_id: &str,
    offer: &str,
    entries: Vec<FileEntry>,
) {
    remember_offer(&entries);

    #[cfg(test)]
    if let Some(captured) = CAPTURED.lock().expect("capture poisoned").as_mut() {
        captured.push(Captured {
            state,
            peer_id: peer_id.to_owned(),
            offer: offer.to_owned(),
            entries,
        });
        return;
    }

    #[cfg(target_os = "windows")]
    windows::present(
        app,
        state,
        peer_id,
        offer,
        entries,
        super::apply::claim_write(),
    );

    #[cfg(not(target_os = "windows"))]
    {
        let _ = (app, state, offer);
        log::info!(
            "sync: received {} file entries from {peer_id}; virtual file clipboard is not implemented on this platform yet",
            entries.len()
        );
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn entry(size: u64) -> FileEntry {
        FileEntry {
            rel_path: format!("f{size}"),
            is_dir: false,
            size,
        }
    }

    #[test]
    fn echo_requires_every_size_from_one_recent_offer() {
        remember_offer(&[entry(6_247_001), entry(18_002)]);

        assert!(sizes_match_recent_offer(&[6_247_001]));
        assert!(sizes_match_recent_offer(&[18_002, 6_247_001]));
        assert!(!sizes_match_recent_offer(&[6_247_001, 6_247_001]));
        assert!(!sizes_match_recent_offer(&[6_247_001, 999_003]));
        assert!(!sizes_match_recent_offer(&[]));
    }
}

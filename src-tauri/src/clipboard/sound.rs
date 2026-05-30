//! 复制成功提示音。音频字节直接 `include_bytes!` 打入二进制——
//! 24KB 的 mp3 比额外维护 resource 路径 + 运行时文件 IO 简单。
//!
//! rodio 的 `OutputStream` 是 `!Send` 且必须存活到播放结束，所以每次播放都
//! 新开一条短命线程：建流 → 解码 → 播放 → sink 空了即释放。剪贴板事件频率
//! 远低于音频开销（建流 ~ms 级），不必维护常驻 worker。

use std::io::Cursor;

use rodio::{Decoder, OutputStream, Sink};
use tauri::{AppHandle, Manager};

use crate::settings::SettingsStore;

const COPY_SOUND_BYTES: &[u8] = include_bytes!("../../assets/sounds/copy.mp3");

/// 若设置启用了 `feedback.copy_sound`，异步播放一次提示音。
/// 失败仅 warn——提示音不应阻断剪贴板入库主流程。
pub fn maybe_play_copy(app: &AppHandle) {
    let enabled = app
        .try_state::<SettingsStore>()
        .map(|s| s.snapshot().clipboard.feedback.copy_sound)
        .unwrap_or(false);
    if !enabled {
        return;
    }
    spawn_play();
}

fn spawn_play() {
    std::thread::Builder::new()
        .name("copy-sound".into())
        .spawn(|| {
            if let Err(err) = play_blocking() {
                log::warn!("play copy sound failed: {err}");
            }
        })
        .ok();
}

fn play_blocking() -> Result<(), String> {
    let (_stream, handle) = OutputStream::try_default().map_err(|e| e.to_string())?;
    let sink = Sink::try_new(&handle).map_err(|e| e.to_string())?;
    let source = Decoder::new(Cursor::new(COPY_SOUND_BYTES)).map_err(|e| e.to_string())?;
    sink.append(source);
    sink.sleep_until_end();
    Ok(())
}

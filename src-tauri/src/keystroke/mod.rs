//! 模拟系统级粘贴。
//!
//! 写回剪贴板由 `clipboard::write` 负责；本模块只负责「按键模拟」这一步——
//! 配合 watcher 的 `WritebackGuard` 抑制自身写回带来的回环（见 4.1）。
//!
//! - macOS：⌘V（CGEvent）
//! - Windows：Shift+Insert（SendInput）。比 Ctrl+V 兼容性更好，传统 Win32
//!   控件、终端、部分 Electron 应用都接收。

#[cfg(target_os = "macos")]
mod macos;
#[cfg(target_os = "windows")]
mod windows;

// 由 `commands::paste` 调用——4.2 第三项已接入；第四项（窗口/焦点时序）仍在路上。
#[cfg(target_os = "macos")]
pub use macos::simulate_paste;
#[cfg(target_os = "windows")]
pub use windows::simulate_paste;

#[cfg(not(any(target_os = "macos", target_os = "windows")))]
pub fn simulate_paste() -> crate::core::error::Result<()> {
    Err(anyhow::anyhow!("simulate_paste not implemented on this platform").into())
}

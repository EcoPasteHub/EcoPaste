//! 模拟系统级粘贴。
//!
//! 写回剪贴板由 `clipboard::write` 负责；本模块只负责「按键模拟」这一步——
//! 配合 watcher 的 `WritebackGuard` 抑制自身写回带来的回环。
//!
//! - macOS：⌘V（CGEvent），依赖「辅助功能」授权，调用前用 `is_paste_permitted` 判断
//! - Windows：Shift+Insert（SendInput）。比 Ctrl+V 兼容性更好，传统 Win32
//!   控件、终端、部分 Electron 应用都接收。

#[cfg(target_os = "macos")]
mod macos;
#[cfg(target_os = "windows")]
mod windows;

#[cfg(target_os = "macos")]
pub use macos::{is_paste_permitted, prompt_paste_permission, simulate_paste};
#[cfg(target_os = "windows")]
pub use windows::simulate_paste;

#[cfg(not(any(target_os = "macos", target_os = "windows")))]
pub fn simulate_paste() -> crate::core::error::Result<()> {
    Err(anyhow::anyhow!("simulate_paste not implemented on this platform").into())
}

/// 非 macOS 平台的按键模拟不依赖系统授权，始终允许。
#[cfg(not(target_os = "macos"))]
pub fn is_paste_permitted() -> bool {
    true
}

#[cfg(not(target_os = "macos"))]
pub fn prompt_paste_permission() {}

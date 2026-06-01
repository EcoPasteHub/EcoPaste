//! 主窗口 `focusable=false` 让 Tauri 在 Windows 上收不到 `tauri://blur`，
//! 无法用焦点事件实现「失焦自动隐藏」。这里装一颗 `WH_MOUSE_LL` 低级钩子，
//! 监听全局鼠标按下：命中主窗口外的位置就让主窗口隐藏。
//! macOS 走 NSPanel 的 `window_did_resign_key`，无需本模块——故仅 windows target 启用。

#[cfg(target_os = "windows")]
mod windows;

#[cfg(target_os = "windows")]
pub use windows::{disable_outside_click_hide, enable_outside_click_hide};

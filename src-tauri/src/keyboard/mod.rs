//! 主窗口 focusable=false 后，Windows 上 WebView 收不到键盘事件，
//! 需要装 OS 级低级钩子捕获导航键再 emit 给前端；macOS NSPanel 不受影响。
//! 与 `keystroke/`（向外注入按键模拟粘贴）方向相反：本模块是向内捕获。

#[cfg_attr(not(target_os = "windows"), allow(dead_code))]
pub const NAV_EVENT: &str = "keyboard://nav";

#[cfg(target_os = "windows")]
mod windows;
#[cfg(target_os = "windows")]
pub use windows::{disable_navigation_keys, enable_navigation_keys};

#[cfg(not(target_os = "windows"))]
mod noop;
#[cfg(not(target_os = "windows"))]
pub use noop::{disable_navigation_keys, enable_navigation_keys};

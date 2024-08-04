#[cfg(target_os = "linux")]
mod linux;
#[cfg(target_os = "macos")]
mod mac;
#[cfg(target_os = "windows")]
pub mod win;

#[cfg(target_os = "linux")]
pub use linux::observe_app;
#[cfg(target_os = "linux")]
pub use linux::get_foreground_apps;

#[cfg(target_os = "macos")]
pub use mac::observe_app;

#[cfg(target_os = "macos")]
pub use mac::get_foreground_apps;

#[cfg(target_os = "windows")]
pub use win::observe_app;

#[cfg(target_os = "windows")]
pub use win::get_foreground_apps;

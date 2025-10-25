#[cfg(target_os = "macos")]
mod macos;

#[cfg(target_os = "windows")]
mod windows;

#[cfg(target_os = "linux")]
mod linux;

#[cfg(target_os = "macos")]
pub use macos::*;

#[cfg(target_os = "windows")]
pub use windows::*;

#[cfg(target_os = "linux")]
pub use linux::*;

#[cfg(not(target_os = "macos"))]
pub fn wait(millis: u64) {
    use std::{thread, time};

    thread::sleep(time::Duration::from_millis(millis));
}

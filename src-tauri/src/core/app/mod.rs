#[cfg(target_os = "macos")]
mod macos;

#[derive(Debug, serde::Serialize, Clone)]
pub struct App {
    pub name: String,
    pub process_id: i32,
}

#[cfg(target_os = "macos")]
pub use macos::observe_app;

#[cfg(target_os = "macos")]
pub use macos::get_frontmost_apps;

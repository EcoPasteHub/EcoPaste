use fs_extra::dir::get_size;
use std::{path::PathBuf, process::Command};
use tauri::{
    command, generate_handler,
    plugin::{Builder, TauriPlugin},
    Result, Wry,
};

#[derive(Debug, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct Metadata {
    size: u64,
    is_dir: bool,
    is_file: bool,
    is_exist: bool,
    file_name: String,
}

#[command]
async fn metadata(path: PathBuf) -> Result<Metadata> {
    let size = get_size(&path).unwrap_or(0);
    let is_dir = path.is_dir();
    let is_file = path.is_file();
    let is_exist = path.exists();
    let file_name = path
        .file_name()
        .map(|name| name.to_string_lossy().to_string())
        .unwrap_or_default();

    Ok(Metadata {
        size,
        is_dir,
        is_file,
        is_exist,
        file_name,
    })
}

#[cfg(target_os = "macos")]
#[command]
pub async fn preview_path(path: &str, finder: bool) -> Result<()> {
    let args = if finder { vec!["-R", path] } else { vec![path] };

    Command::new("open").args(args).spawn()?;

    Ok(())
}

#[cfg(target_os = "windows")]
#[command]
pub async fn preview_path(path: &str, finder: bool) -> Result<()> {
    let args = if finder {
        vec!["/select,", path]
    } else {
        vec![path]
    };

    Command::new("explorer").args(args).spawn()?;

    Ok(())
}

#[cfg(target_os = "linux")]
#[command]
pub async fn preview_path(path: &str, finder: bool) -> Result<()> {
    if finder {
        showfile::show_path_in_file_manager(path);
    } else {
        Command::new("xdg-open").arg(path).spawn()?;
    }

    Ok(())
}

pub fn init() -> TauriPlugin<Wry> {
    Builder::new("fs-extra")
        .invoke_handler(generate_handler![metadata, preview_path])
        .build()
}

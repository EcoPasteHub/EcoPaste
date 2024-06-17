use image_base64::to_base64;
use std::{fs, path::PathBuf};
use tauri::{
    command, generate_handler,
    plugin::{Builder, TauriPlugin},
    Result, Wry,
};

#[derive(Debug, serde::Serialize)]
struct Metadata {
    size: u64,
    is_dir: bool,
    is_file: bool,
    is_exist: bool,
}

fn get_dir_size(path: PathBuf) -> Result<u64> {
    let mut size = 0;

    for entry in fs::read_dir(path)? {
        let entry = entry?;
        let metadata = entry.metadata()?;

        if metadata.is_file() {
            size += metadata.len();
        } else if metadata.is_dir() {
            size += get_dir_size(entry.path())?;
        }
    }

    Ok(size)
}

#[command]
async fn metadata(path: PathBuf) -> Result<Metadata> {
    let mut size = 0;
    let mut is_dir = false;
    let mut is_file = false;
    let is_exist = path.exists();

    if is_exist {
        let metadata = fs::metadata(&path)?;

        is_dir = metadata.is_dir();
        is_file = metadata.is_file();

        size = if is_file {
            metadata.len()
        } else {
            get_dir_size(path)?
        };
    }

    Ok(Metadata {
        size,
        is_dir,
        is_file,
        is_exist,
    })
}

#[command]
async fn exists(path: PathBuf) -> bool {
    path.exists()
}

#[command]
pub async fn get_image_base64(path: &str) -> Result<String> {
    Ok(to_base64(path))
}

pub fn init() -> TauriPlugin<Wry> {
    Builder::new("fs-extra")
        .invoke_handler(generate_handler![metadata, exists, get_image_base64])
        .build()
}

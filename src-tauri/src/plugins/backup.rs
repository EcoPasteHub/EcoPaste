use super::fs_extra::preview_path;
use flate2::{read::GzDecoder, write::GzEncoder, Compression};
use std::fs::{read_dir, File};
use tar::Archive;
use tauri::{
    api::path::{app_data_dir, download_dir},
    command, generate_handler,
    plugin::{Builder, TauriPlugin},
    AppHandle, Result, Wry,
};

#[command]
async fn export_data(app_handle: AppHandle, file_name: String) -> Result<()> {
    let src_dir = app_data_dir(&app_handle.config()).unwrap();
    let dst_path = download_dir().unwrap().join(file_name.clone());
    let dst_file = File::create(dst_path.clone())?;

    let enc = GzEncoder::new(dst_file, Compression::default());
    let mut tar = tar::Builder::new(enc);

    for entry in read_dir(&src_dir)? {
        let entry = entry?;
        let path = entry.path();
        let name = path.strip_prefix(&src_dir).unwrap();
        if path.is_dir() {
            tar.append_dir_all(name, path.clone())?;
        } else {
            tar.append_file(name, &mut File::open(path.clone())?)?;
        }
    }

    tar.finish()?;

    preview_path(dst_path.to_str().unwrap(), true)
        .await
        .unwrap();

    Ok(())
}

#[command]
async fn import_data(app_handle: AppHandle, path: String) -> Result<bool> {
    let dst_dir = app_data_dir(&app_handle.config()).unwrap();

    let file = File::open(path)?;
    let decoder = GzDecoder::new(file);
    let mut archive = Archive::new(decoder);

    for entry in archive.entries()? {
        let mut entry = entry?;
        let path = entry.path()?;

        #[cfg(target_os = "windows")]
        let path = std::path::Path::new(&path.to_string_lossy().replace("\\", "/")).to_path_buf();

        entry.unpack(dst_dir.join(path))?;
    }

    Ok(true)
}

pub fn init() -> TauriPlugin<Wry> {
    Builder::new("backup")
        .invoke_handler(generate_handler![export_data, import_data])
        .build()
}

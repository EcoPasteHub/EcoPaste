use super::fs_extra::{get_file_name, preview_path};
use flate2::{read::GzDecoder, write::GzEncoder, Compression};
use fs_extra::{
    dir::{ls, CopyOptions, DirEntryAttr, DirEntryValue},
    move_items,
};
use std::{
    collections::HashSet,
    fs::{create_dir_all, read_dir, remove_dir, File},
    path::PathBuf,
};
use tar::Archive;
use tauri::{
    api::path::download_dir,
    command, generate_handler,
    plugin::{Builder, TauriPlugin},
    Wry,
};

#[command]
async fn export_data(src_dir: PathBuf, file_name: String) -> tauri::Result<()> {
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
async fn import_data(dst_dir: PathBuf, path: String) -> tauri::Result<bool> {
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

#[command]
async fn move_data(from: PathBuf, to: PathBuf) -> Result<PathBuf, String> {
    create_dir_all(to.clone()).map_err(|err| err.to_string())?;

    let mut config = HashSet::new();
    config.insert(DirEntryAttr::Path);

    let ls_result = ls(&from, &config).unwrap();

    let mut from_items = Vec::new();

    for item in ls_result.items {
        if let Some(path) = item.get(&DirEntryAttr::Path) {
            if let &DirEntryValue::String(ref path) = path {
                let path = PathBuf::from(path);
                let is_dir = path.is_dir();
                let is_file = path.is_file();
                let file_name = get_file_name(path.clone());

                // 忽略主题插件和窗口状态插件生成的的文件，无法修改存储路径
                let skip_files = ["tauri-plugin-theme", ".window-state"];
                if is_file && skip_files.contains(&file_name.as_str()) {
                    continue;
                }

                // 忽略日志插件生成的目录，无法修改存储路径
                let skip_dirs = ["logs"];
                if is_dir && skip_dirs.contains(&file_name.as_str()) {
                    continue;
                }

                from_items.push(path);
            }
        }
    }

    let options = CopyOptions {
        overwrite: true,
        skip_exist: false,
        buffer_size: 64000,
        copy_inside: false,
        content_only: false,
        depth: 0,
    };

    move_items(&from_items, &to, &options).map_err(|err| err.to_string())?;

    let _ = remove_dir(from);

    Ok(to)
}

pub fn init() -> TauriPlugin<Wry> {
    Builder::new("backup")
        .invoke_handler(generate_handler![export_data, import_data, move_data])
        .build()
}

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
use tauri::{command, AppHandle, Runtime};
use tauri_plugin_fs_pro::{metadata, open, OpenOptions};

// 导出数据
#[command]
pub async fn export_data<R: Runtime>(
    app_handle: AppHandle<R>,
    src_dir: String,
    dst_path: PathBuf,
) -> Result<(), String> {
    let dst_file = File::create(dst_path.clone()).map_err(|err| err.to_string())?;
    let enc = GzEncoder::new(dst_file, Compression::default());
    let mut tar = tar::Builder::new(enc);

    for entry in read_dir(&src_dir).map_err(|err| err.to_string())? {
        let path = entry.map_err(|err| err.to_string())?.path();
        let metadata = metadata(path.clone()).await?;

        if metadata.is_file {
            if metadata.extname == "db" || metadata.name.starts_with(".store-backup.") {
                let file = &mut File::open(path.clone()).map_err(|err| err.to_string())?;

                tar.append_file(metadata.name.clone(), file)
                    .map_err(|err| err.to_string())?;
            }
        }

        if metadata.is_dir && metadata.name == "images" {
            tar.append_dir_all(metadata.name, path.clone())
                .map_err(|err| err.to_string())?;
        }
    }

    tar.finish().map_err(|err| err.to_string())?;

    let open_options = OpenOptions {
        explorer: Some(true),
        enter_dir: Some(false),
    };

    let _ = open(app_handle, dst_path, Some(open_options)).await;

    Ok(())
}

// 导入数据
#[command]
pub async fn import_data(src_path: String, dst_dir: PathBuf) -> Result<bool, String> {
    let src_file = File::open(src_path).map_err(|err| err.to_string())?;
    let decoder = GzDecoder::new(src_file);
    let mut archive = Archive::new(decoder);

    for entry in archive.entries().map_err(|err| err.to_string())? {
        let mut entry = entry.map_err(|err| err.to_string())?;
        let path = entry.path().map_err(|err| err.to_string())?;

        #[cfg(target_os = "windows")]
        let path = std::path::Path::new(&path.to_string_lossy().replace("\\", "/")).to_path_buf();

        entry
            .unpack(dst_dir.join(path))
            .map_err(|err| err.to_string())?;
    }

    Ok(true)
}

// 移动数据
#[command]
pub async fn move_data(from: PathBuf, to: PathBuf) -> Result<PathBuf, String> {
    create_dir_all(to.clone()).map_err(|err| err.to_string())?;

    let mut config = HashSet::new();
    config.insert(DirEntryAttr::Path);

    let ls_result = ls(&from, &config).map_err(|err| err.to_string())?;

    let mut from_items = Vec::new();

    for item in ls_result.items {
        if let Some(path) = item.get(&DirEntryAttr::Path) {
            if let &DirEntryValue::String(ref path) = path {
                let path = PathBuf::from(path);
                let metadata = metadata(path.clone()).await?;

                if metadata.is_dir && metadata.name != "images" {
                    continue;
                }

                if metadata.is_file && metadata.extname == "json" {
                    continue;
                }

                from_items.push(path.clone());
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

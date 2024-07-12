use super::fs_extra::preview_path;
use std::{
    fs::File,
    io::{self, BufReader},
};
use tauri::{
    api::path::{app_data_dir, download_dir},
    command, generate_handler,
    plugin::{Builder, TauriPlugin},
    AppHandle, Result, Wry,
};
use walkdir::WalkDir;
use zip::{read::ZipArchive, write::SimpleFileOptions, CompressionMethod, ZipWriter};

#[command]
async fn export_data(app_handle: AppHandle, file_name: String) -> Result<()> {
    let entry_dir = app_data_dir(&app_handle.config()).unwrap();
    let output_path = download_dir().unwrap().join(file_name);

    let file = File::create(output_path.clone()).unwrap();
    let mut zip = ZipWriter::new(file);
    let walkdir = WalkDir::new(entry_dir.clone());
    let it = walkdir.into_iter();

    let options = SimpleFileOptions::default()
        .compression_method(CompressionMethod::Deflated)
        .unix_permissions(0o755);

    for entry in it {
        let entry = entry.unwrap();
        let path = entry.path();
        let name = path.strip_prefix(entry_dir.clone()).unwrap();

        if path.is_file() {
            zip.start_file(name.to_string_lossy(), options).unwrap();
            let mut f = File::open(path)?;
            std::io::copy(&mut f, &mut zip)?;
        } else if !name.as_os_str().is_empty() {
            zip.add_directory(name.to_string_lossy(), options).unwrap();
        }
    }

    zip.finish().unwrap();

    preview_path(output_path.to_str().unwrap(), true)
        .await
        .unwrap();

    Ok(())
}

#[command]
async fn import_data(app_handle: AppHandle, path: String) -> Result<bool> {
    let out_dir = app_data_dir(&app_handle.config()).unwrap();

    let zip_file = File::open(path).unwrap();
    let mut archive = ZipArchive::new(BufReader::new(zip_file)).unwrap();

    for i in 0..archive.len() {
        let mut file = archive.by_index(i).unwrap();
        let outpath = match file.enclosed_name() {
            Some(path) => out_dir.join(path),
            None => continue,
        };

        if file.is_dir() {
            std::fs::create_dir_all(&outpath).unwrap();
        } else {
            if let Some(p) = outpath.parent() {
                if !p.exists() {
                    std::fs::create_dir_all(&p).unwrap();
                }
            }
            let mut outfile = File::create(&outpath).unwrap();
            io::copy(&mut file, &mut outfile)?;
        }

        // 设置解压后的文件的权限
        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;

            if let Some(mode) = file.unix_mode() {
                std::fs::set_permissions(&outpath, std::fs::Permissions::from_mode(mode))?;
            }
        }
    }

    Ok(true)
}

pub fn init() -> TauriPlugin<Wry> {
    Builder::new("backup")
        .invoke_handler(generate_handler![export_data, import_data])
        .build()
}

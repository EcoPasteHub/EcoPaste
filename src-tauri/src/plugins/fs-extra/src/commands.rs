use fs_extra::dir::get_size;
use showfile::show_path_in_file_manager;
use std::path::PathBuf;
use tauri::{command, AppHandle, Runtime};
use tauri_plugin_shell::ShellExt;

#[derive(Debug, serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Metadata {
    pub size: u64,
    pub is_dir: bool,
    pub is_file: bool,
    pub is_exist: bool,
    pub name: String,
    pub extname: String,
}

// 获取文件（夹）名
fn get_name(path: PathBuf) -> String {
    path.file_name()
        .map(|name| name.to_string_lossy().to_string())
        .unwrap_or_default()
}

// 获取文件后缀名
fn get_extname(path: PathBuf) -> String {
    path.extension()
        .map(|ext| ext.to_string_lossy().to_string())
        .unwrap_or_default()
}

// 获取路径的元信息
#[command]
pub async fn metadata(path: PathBuf) -> Result<Metadata, String> {
    let size = get_size(&path).unwrap_or(0);
    let is_dir = path.is_dir();
    let is_file = path.is_file();
    let is_exist = path.exists();
    let name = get_name(path.clone());
    let extname = get_extname(path);

    Ok(Metadata {
        size,
        is_dir,
        is_file,
        is_exist,
        name,
        extname,
    })
}

// 在默认程序或者文件资源管理器打开指定路径
#[command]
pub async fn open_path<R: Runtime>(
    app_handle: AppHandle<R>,
    path: String,
    finder: bool,
) -> Result<(), String> {
    if finder {
        show_path_in_file_manager(path);
    } else {
        let shell = app_handle.shell();

        shell.open(path, None).map_err(|err| err.to_string())?;
    }

    Ok(())
}

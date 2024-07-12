use std::{fs, path::PathBuf, process::Command};
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
    let mut file_name = String::new();
    let is_exist = path.exists();

    if is_exist {
        let metadata = fs::metadata(&path)?;

        is_dir = metadata.is_dir();
        is_file = metadata.is_file();

        if let Some(name) = path.file_name() {
            file_name = name.to_string_lossy().to_string();
        }

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
        file_name,
    })
}

#[command]
pub async fn preview_path(path: &str, finder: bool) -> Result<()> {
    let mut program = "open";
    let mut args = vec![path];

    if cfg!(target_os = "windows") {
        program = "explorer"
    }

    if finder {
        if cfg!(target_os = "windows") {
            args.insert(0, "/select,")
        } else {
            args.insert(0, "-R")
        }
    }

    Command::new(program).args(args).spawn().unwrap();

    Ok(())
}

pub fn init() -> TauriPlugin<Wry> {
    Builder::new("fs-extra")
        .invoke_handler(generate_handler![metadata, preview_path])
        .build()
}

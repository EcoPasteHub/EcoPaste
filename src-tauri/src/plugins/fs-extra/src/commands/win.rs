use std::process::Command;
use tauri::command;

// 预览路径文件（夹）
#[command]
pub async fn preview_path(path: &str, finder: bool) -> Result<(), String> {
    let args = if finder {
        vec!["/select,", path]
    } else {
        vec![path]
    };

    Command::new("explorer")
        .args(args)
        .spawn()
        .map_err(|err| err.to_string())?;

    Ok(())
}

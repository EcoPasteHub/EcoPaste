use std::process::Command;
use tauri::command;

// 预览路径文件（夹）
#[command]
pub async fn preview_path(path: &str, finder: bool) -> Result<(), String> {
    if finder {
        showfile::show_path_in_file_manager(path);
    } else {
        Command::new("xdg-open")
            .arg(path)
            .spawn()
            .map_err(|err| err.to_string())?;
    }

    Ok(())
}

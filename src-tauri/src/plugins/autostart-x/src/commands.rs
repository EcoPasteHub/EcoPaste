use std::env::args;
use tauri::command;

// 是否为自动启动
#[command]
pub async fn is_autostart() -> bool {
    let args: Vec<String> = args().collect();

    return args.contains(&"--auto-launch".to_string());
}

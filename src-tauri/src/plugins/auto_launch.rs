use std::env;
use tauri::{
    command, generate_handler,
    plugin::{Builder, TauriPlugin},
    Result, Wry,
};

#[command]
async fn is_auto_launch() -> Result<bool> {
    let args: Vec<String> = env::args().collect();

    println!("Args: {:?}", args);

    Ok(args.contains(&"--auto-launch".to_string()))
}

pub fn init() -> TauriPlugin<Wry> {
    Builder::new("auto_launch")
        .invoke_handler(generate_handler![is_auto_launch])
        .build()
}

use crate::AUTO_LAUNCH_ARG;
use std::env;
use tauri::{
    command, generate_handler,
    plugin::{Builder, TauriPlugin},
    Result, Wry,
};

#[command]
async fn is_auto_launch() -> Result<bool> {
    let args: Vec<String> = env::args().collect();

    Ok(args.contains(&AUTO_LAUNCH_ARG.to_string()))
}

pub fn init() -> TauriPlugin<Wry> {
    Builder::new("auto_launch")
        .invoke_handler(generate_handler![is_auto_launch])
        .build()
}

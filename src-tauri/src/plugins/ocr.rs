use tauri::{
    command, generate_handler,
    plugin::{Builder, TauriPlugin},
    Result, Wry,
};

#[command]
async fn system_ocr(path: &str) -> Result<String> {
    use tauri::api::process::{Command, CommandEvent};

    let (mut rx, _child) = Command::new_sidecar("ocr")
        .expect("Failed to find sidecar")
        .args(&[path])
        .spawn()
        .expect("Failed to spawn sidecar");

    loop {
        let event = rx.recv().await;
        match event {
            Some(CommandEvent::Stdout(line)) => {
                return Ok(line.to_string());
            }
            _ => return Ok("".to_string()),
        }
    }
}

pub fn init() -> TauriPlugin<Wry> {
    Builder::new("ocr")
        .invoke_handler(generate_handler![system_ocr])
        .build()
}

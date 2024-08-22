use tauri::{
    command, generate_handler,
    plugin::{Builder, TauriPlugin},
    Wry,
};

#[command]
#[cfg(not(target_os = "linux"))]
async fn system_ocr(path: &str) -> Result<String, String> {
    use tauri::api::process::{Command, CommandEvent};

    let (mut rx, _child) = Command::new_sidecar("ocr")
        .map_err(|err| err.to_string())?
        .args(&[path])
        .spawn()
        .map_err(|err| err.to_string())?;

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

#[command]
#[cfg(target_os = "linux")]
async fn system_ocr(path: &str) -> tauri::Result<String> {
    let output = match std::process::Command::new("tesseract")
        .arg(path)
        .arg("stdout")
        .args(["-l", "eng+chi_sim+jpn"])
        .output()
    {
        Ok(output) => output,
        Err(e) => {
            log::error!("Failed to exec tesseract: {:?}", e);
            return Ok(String::default());
        }
    };

    let text = if output.status.success() {
        String::from_utf8(output.stdout)
            .unwrap_or_default()
            .split_whitespace()
            .collect()
    } else {
        let content = String::from_utf8(output.stderr).unwrap_or_default();
        log::error!("Error while executing tesseract: {:?}", content);
        String::default()
    };

    Ok(text)
}

pub fn init() -> TauriPlugin<Wry> {
    Builder::new("ocr")
        .invoke_handler(generate_handler![system_ocr])
        .build()
}

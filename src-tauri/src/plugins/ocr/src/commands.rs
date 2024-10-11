use tauri::{command, AppHandle, Runtime};

#[command]
#[cfg(not(target_os = "linux"))]
pub async fn system_ocr<R: Runtime>(
    app_handle: AppHandle<R>,
    path: &str,
) -> Result<String, String> {
    use tauri_plugin_shell::{process::CommandEvent, ShellExt};

    // https://tauri.app/develop/sidecar/#running-it-from-rust
    let (mut rx, _child) = app_handle
        .shell()
        .sidecar("ocr")
        .map_err(|err| err.to_string())?
        .args(&[path])
        .spawn()
        .map_err(|err| err.to_string())?;

    loop {
        let event = rx.recv().await;

        match event {
            Some(CommandEvent::Stdout(line_bytes)) => {
                let line = String::from_utf8_lossy(&line_bytes);

                return Ok(line.to_string());
            }
            _ => return Ok("".to_string()),
        }
    }
}

#[command]
#[cfg(target_os = "linux")]
pub async fn system_ocr<R: Runtime>(
    _app_handle: AppHandle<R>,
    path: &str,
) -> tauri::Result<String> {
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

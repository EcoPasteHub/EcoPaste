use tauri::{
    command, generate_handler,
    plugin::{Builder, TauriPlugin},
    Result, Wry,
};

#[command]
#[cfg(target_os = "windows")]
async fn system_ocr(path: &str) -> Result<String> {
    use tauri::Error;
    use windows::core::HSTRING;
    use windows::Graphics::Imaging::BitmapDecoder;
    use windows::Media::Ocr::OcrEngine;
    use windows::Storage::{FileAccessMode, StorageFile};

    let file = StorageFile::GetFileFromPathAsync(&HSTRING::from(path))
        .unwrap()
        .get()
        .unwrap();

    let bitmap = BitmapDecoder::CreateWithIdAsync(
        BitmapDecoder::PngDecoderId().unwrap(),
        &file.OpenAsync(FileAccessMode::Read).unwrap().get().unwrap(),
    )
    .unwrap()
    .get()
    .unwrap();

    let bitmap = bitmap.GetSoftwareBitmapAsync().unwrap().get().unwrap();

    let engine = OcrEngine::TryCreateFromUserProfileLanguages();

    match engine {
        Ok(v) => Ok(v
            .RecognizeAsync(&bitmap)
            .unwrap()
            .get()
            .unwrap()
            .Text()
            .unwrap()
            .to_string_lossy()),
        Err(e) => {
            if e.to_string().contains("0x00000000") {
                // "Language package not installed!\n\nSee: https://learn.microsoft.com/zh-cn/windows/powertoys/text-extractor#supported-languages"
                Err(Error::InvokeKey)
            } else {
                Err(Error::InvokeKey)
            }
        }
    }
}

#[command]
#[cfg(target_os = "macos")]
async fn system_ocr(path: &str) -> Result<String> {
    use tauri::api::process::{Command, CommandEvent};

    let (mut rx, _child) = Command::new_sidecar("ocr")
        .expect("Failed to find sidecar")
        .args(&[path, "zh"])
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

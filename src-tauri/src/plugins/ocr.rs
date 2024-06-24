use tauri::api::process::{Command, CommandEvent};
use tauri::{
    command, generate_handler,
    plugin::{Builder, TauriPlugin},
    Result, Wry,
};

#[command(async)]
#[cfg(target_os = "windows")]
async fn system_ocr(app_handle: tauri::AppHandle, path: &str) -> Result<String, String> {
    use windows::core::HSTRING;
    use windows::Globalization::Language;
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

    let engine = match lang {
        "auto" => OcrEngine::TryCreateFromUserProfileLanguages(),
        _ => {
            if let Ok(language) = Language::CreateLanguage(&HSTRING::from("zh")) {
                OcrEngine::TryCreateFromLanguage(&language)
            } else {
                return Err("Language Error".to_string());
            }
        }
    };

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
                Err("Language package not installed!\n\nSee: https://learn.microsoft.com/zh-cn/windows/powertoys/text-extractor#supported-languages".to_string())
            } else {
                Err(e.to_string())
            }
        }
    }
}

#[command(async)]
#[cfg(target_os = "macos")]
async fn system_ocr(path: &str) -> Result<String> {
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

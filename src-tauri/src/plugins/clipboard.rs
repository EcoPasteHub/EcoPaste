use clipboard_rs::{
    common::RustImage, Clipboard, ClipboardContent, ClipboardContext, ClipboardHandler,
    ClipboardWatcher, ClipboardWatcherContext, ContentFormat, RustImageData, WatcherShutdown,
};
use std::{
    fs::create_dir_all,
    hash::{DefaultHasher, Hash, Hasher},
    path::PathBuf,
    sync::{Arc, Mutex},
    thread::spawn,
};
use tauri::{
    command, generate_handler,
    plugin::{Builder, TauriPlugin},
    AppHandle, Error, Manager, Result, State, Wry,
};

struct ClipboardManager {
    context: ClipboardContext,
    watcher_shutdown: Arc<Mutex<Option<WatcherShutdown>>>,
}

struct ClipboardListen {
    app_handle: tauri::AppHandle,
}

impl ClipboardManager {
    fn new() -> Self {
        ClipboardManager {
            context: ClipboardContext::new().unwrap(),
            watcher_shutdown: Arc::default(),
        }
    }

    fn has(&self, format: ContentFormat) -> bool {
        self.context.has(format)
    }
}

impl ClipboardListen {
    fn new(app_handle: AppHandle) -> Self {
        Self { app_handle }
    }
}

impl ClipboardHandler for ClipboardListen {
    fn on_clipboard_change(&mut self) {
        self.app_handle
            .emit_all("plugin:clipboard://clipboard_update", "Clipboard updated")
            .unwrap();
    }
}

#[derive(Debug, serde::Serialize)]
struct ReadImage {
    width: u32,
    height: u32,
    image: String,
}

#[command]
async fn start_listen(app_handle: AppHandle, manager: State<'_, ClipboardManager>) -> Result<()> {
    let listener = ClipboardListen::new(app_handle.clone());

    let mut watcher: ClipboardWatcherContext<ClipboardListen> =
        ClipboardWatcherContext::new().unwrap();

    let watcher_shutdown = watcher.add_handler(listener).get_shutdown_channel();

    let mut watcher_shutdown_state = manager.watcher_shutdown.lock().unwrap();

    if (*watcher_shutdown_state).is_some() {
        return Ok(());
    }

    *watcher_shutdown_state = Some(watcher_shutdown);

    spawn(move || {
        watcher.start_watch();
    });

    Ok(())
}

#[command]
async fn stop_listen(manager: State<'_, ClipboardManager>) -> Result<()> {
    let mut watcher_shutdown = manager.watcher_shutdown.lock().unwrap();

    if let Some(watcher_shutdown) = (*watcher_shutdown).take() {
        watcher_shutdown.stop();
    }

    *watcher_shutdown = None;

    Ok(())
}

#[command]
async fn has_files(manager: State<'_, ClipboardManager>) -> Result<bool> {
    Ok(manager.has(ContentFormat::Files))
}

#[command]
async fn has_image(manager: State<'_, ClipboardManager>) -> Result<bool> {
    Ok(manager.has(ContentFormat::Image))
}

#[command]
async fn has_html(manager: State<'_, ClipboardManager>) -> Result<bool> {
    Ok(manager.has(ContentFormat::Html))
}

#[command]
async fn has_rich_text(manager: State<'_, ClipboardManager>) -> Result<bool> {
    Ok(manager.has(ContentFormat::Rtf))
}

#[command]
async fn has_text(manager: State<'_, ClipboardManager>) -> Result<bool> {
    Ok(manager.has(ContentFormat::Text))
}

#[command]
async fn read_files(manager: State<'_, ClipboardManager>) -> Result<Vec<String>> {
    let mut files = manager.context.get_files().unwrap();

    files.iter_mut().for_each(|path| {
        *path = path.replace("file://", "");
    });

    Ok(files)
}

#[command]
async fn read_image(manager: State<'_, ClipboardManager>, dir: PathBuf) -> Result<ReadImage> {
    create_dir_all(&dir).unwrap();

    let image = manager.context.get_image().unwrap();

    let (width, height) = image.get_size();

    let thumbnail_image = image.thumbnail(width / 10, height / 10).unwrap();

    let bytes = thumbnail_image.to_png().unwrap().get_bytes().to_vec();

    let mut hasher = DefaultHasher::new();

    bytes.hash(&mut hasher);

    let hash = hasher.finish();

    let image_path = dir.join(format!("{hash}.png"));

    if let Some(path) = image_path.to_str() {
        image.save_to_path(path).unwrap();

        let image = path.to_string();

        return Ok(ReadImage {
            width,
            height,
            image,
        });
    }

    Err(Error::InvokeKey)
}

#[command]
async fn read_html(manager: State<'_, ClipboardManager>) -> Result<String> {
    Ok(manager.context.get_html().unwrap())
}

#[command]
async fn read_rich_text(manager: State<'_, ClipboardManager>) -> Result<String> {
    Ok(manager.context.get_rich_text().unwrap())
}

#[command]
async fn read_text(manager: State<'_, ClipboardManager>) -> Result<String> {
    Ok(manager.context.get_text().unwrap())
}

#[command]
async fn write_files(manager: State<'_, ClipboardManager>, value: Vec<String>) -> Result<()> {
    manager.context.set_files(value).unwrap();

    Ok(())
}

#[command]
async fn write_image(manager: State<'_, ClipboardManager>, value: String) -> Result<()> {
    let image = RustImageData::from_path(&value).unwrap();

    manager.context.set_image(image).unwrap();

    Ok(())
}

#[command]
async fn write_html(
    manager: State<'_, ClipboardManager>,
    text: String,
    html: String,
) -> Result<()> {
    let contents = vec![ClipboardContent::Text(text), ClipboardContent::Html(html)];

    manager.context.set(contents).unwrap();

    Ok(())
}

#[command]
async fn write_rich_text(manager: State<'_, ClipboardManager>, value: String) -> Result<()> {
    manager.context.set_rich_text(value).unwrap();

    Ok(())
}

#[command]
async fn write_text(manager: State<'_, ClipboardManager>, value: String) -> Result<()> {
    manager.context.set_text(value).unwrap();

    Ok(())
}

pub fn init() -> TauriPlugin<Wry> {
    Builder::new("clipboard")
        .setup(move |app| {
            app.manage(ClipboardManager::new());

            Ok(())
        })
        .invoke_handler(generate_handler![
            start_listen,
            stop_listen,
            has_files,
            has_image,
            has_html,
            has_rich_text,
            has_text,
            read_files,
            read_image,
            read_html,
            read_rich_text,
            read_text,
            write_files,
            write_image,
            write_html,
            write_rich_text,
            write_text,
        ])
        .build()
}

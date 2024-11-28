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
use tauri::{command, AppHandle, Emitter, Runtime, State};

pub struct ClipboardManager {
    context: Arc<Mutex<ClipboardContext>>,
    watcher_shutdown: Arc<Mutex<Option<WatcherShutdown>>>,
}

struct ClipboardListen<R>
where
    R: Runtime,
{
    app_handle: AppHandle<R>,
}

impl ClipboardManager {
    pub fn new() -> Self {
        ClipboardManager {
            context: Arc::new(Mutex::new(ClipboardContext::new().unwrap())),
            watcher_shutdown: Arc::default(),
        }
    }

    fn has(&self, format: ContentFormat) -> bool {
        self.context.lock().unwrap().has(format)
    }
}

impl<R> ClipboardListen<R>
where
    R: Runtime,
{
    fn new(app_handle: AppHandle<R>) -> Self {
        Self { app_handle }
    }
}

impl<R> ClipboardHandler for ClipboardListen<R>
where
    R: Runtime,
{
    fn on_clipboard_change(&mut self) {
        let _ = self
            .app_handle
            .emit("plugin:eco-clipboard://clipboard_update", ())
            .map_err(|err| err.to_string());
    }
}

#[derive(Debug, serde::Serialize)]
pub struct ReadImage {
    width: u32,
    height: u32,
    image: String,
}

#[command]
pub async fn start_listen<R: Runtime>(
    app_handle: AppHandle<R>,
    manager: State<'_, ClipboardManager>,
) -> Result<(), String> {
    let listener = ClipboardListen::new(app_handle.clone());

    let mut watcher: ClipboardWatcherContext<ClipboardListen<R>> =
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
pub async fn stop_listen(manager: State<'_, ClipboardManager>) -> Result<(), String> {
    let mut watcher_shutdown = manager.watcher_shutdown.lock().unwrap();

    if let Some(watcher_shutdown) = (*watcher_shutdown).take() {
        watcher_shutdown.stop();
    }

    *watcher_shutdown = None;

    Ok(())
}

#[command]
pub async fn has_files(manager: State<'_, ClipboardManager>) -> Result<bool, String> {
    Ok(manager.has(ContentFormat::Files))
}

#[command]
pub async fn has_image(manager: State<'_, ClipboardManager>) -> Result<bool, String> {
    Ok(manager.has(ContentFormat::Image))
}

#[command]
pub async fn has_html(manager: State<'_, ClipboardManager>) -> Result<bool, String> {
    Ok(manager.has(ContentFormat::Html))
}

#[command]
pub async fn has_rtf(manager: State<'_, ClipboardManager>) -> Result<bool, String> {
    Ok(manager.has(ContentFormat::Rtf))
}

#[command]
pub async fn has_text(manager: State<'_, ClipboardManager>) -> Result<bool, String> {
    Ok(manager.has(ContentFormat::Text))
}

#[command]
pub async fn read_files(manager: State<'_, ClipboardManager>) -> Result<Vec<String>, String> {
    let mut files = manager
        .context
        .lock()
        .map_err(|err| err.to_string())?
        .get_files()
        .map_err(|err| err.to_string())?;

    files.iter_mut().for_each(|path| {
        *path = path.replace("file://", "");
    });

    Ok(files)
}

#[command]
pub async fn read_image(
    manager: State<'_, ClipboardManager>,
    path: PathBuf,
) -> Result<ReadImage, String> {
    create_dir_all(&path).map_err(|op| op.to_string())?;

    let image = manager
        .context
        .lock()
        .map_err(|err| err.to_string())?
        .get_image()
        .map_err(|err| err.to_string())?;

    let (width, height) = image.get_size();

    let thumbnail_image = image
        .thumbnail(width / 10, height / 10)
        .map_err(|err| err.to_string())?;

    let bytes = thumbnail_image
        .to_png()
        .map_err(|err| err.to_string())?
        .get_bytes()
        .to_vec();

    let mut hasher = DefaultHasher::new();

    bytes.hash(&mut hasher);

    let hash = hasher.finish();

    let image_path = path.join(format!("{hash}.png"));

    if let Some(path) = image_path.to_str() {
        image.save_to_path(path).map_err(|err| err.to_string())?;

        let image = path.to_string();

        return Ok(ReadImage {
            width,
            height,
            image,
        });
    }

    Err("read_image execution error".to_string())
}

#[command]
pub async fn read_html(manager: State<'_, ClipboardManager>) -> Result<String, String> {
    manager
        .context
        .lock()
        .map_err(|err| err.to_string())?
        .get_html()
        .map_err(|err| err.to_string())
}

#[command]
pub async fn read_rtf(manager: State<'_, ClipboardManager>) -> Result<String, String> {
    manager
        .context
        .lock()
        .map_err(|err| err.to_string())?
        .get_rich_text()
        .map_err(|err| err.to_string())
}

#[command]
pub async fn read_text(manager: State<'_, ClipboardManager>) -> Result<String, String> {
    manager
        .context
        .lock()
        .map_err(|err| err.to_string())?
        .get_text()
        .map_err(|err| err.to_string())
}

#[command]
pub async fn write_files(
    manager: State<'_, ClipboardManager>,
    value: Vec<String>,
) -> Result<(), String> {
    manager
        .context
        .lock()
        .map_err(|err| err.to_string())?
        .set_files(value)
        .map_err(|err| err.to_string())
}

#[command]
pub async fn write_image(
    manager: State<'_, ClipboardManager>,
    value: String,
) -> Result<(), String> {
    // 尝试从路径创建 RustImageData，如果失败则返回错误信息
    let image = RustImageData::from_path(&value).map_err(|err| err.to_string())?;

    // 尝试获取锁并设置图像，如果失败则返回错误信息
    manager
        .context
        .lock()
        .map_err(|err| err.to_string())?
        .set_image(image)
        .map_err(|err| err.to_string())
}

#[command]
pub async fn write_html(
    manager: State<'_, ClipboardManager>,
    text: String,
    html: String,
) -> Result<(), String> {
    let contents = vec![ClipboardContent::Text(text), ClipboardContent::Html(html)];

    manager
        .context
        .lock()
        .map_err(|err| err.to_string())?
        .set(contents)
        .map_err(|err| err.to_string())
}

#[command]
pub async fn write_rtf(
    manager: State<'_, ClipboardManager>,
    text: String,
    rtf: String,
) -> Result<(), String> {
    let mut contents = vec![ClipboardContent::Rtf(rtf)];

    if cfg!(not(target_os = "macos")) {
        contents.push(ClipboardContent::Text(text))
    }

    manager
        .context
        .lock()
        .map_err(|err| err.to_string())?
        .set(contents)
        .map_err(|err| err.to_string())
}

#[command]
pub async fn write_text(manager: State<'_, ClipboardManager>, value: String) -> Result<(), String> {
    manager
        .context
        .lock()
        .map_err(|err| err.to_string())?
        .set_text(value)
        .map_err(|err| err.to_string())
}

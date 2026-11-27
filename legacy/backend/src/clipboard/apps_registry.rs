//! 应用注册表：把「运行中应用」「监听过程中捕获的前台应用」「用户手动添加的应用」
//! 统一物化为可展示的应用记录，并维护一份 id → 应用的内存缓存。
//! 运行中应用只进缓存；复制捕获、手动添加和默认忽略物化的应用才写入 `clipboard_apps` 表。

use std::collections::HashMap;
use std::path::{Path, PathBuf};
use std::sync::{Arc, RwLock};

use anyhow::anyhow;
use chrono::Utc;
use sqlx::SqlitePool;
use tauri::{AppHandle, Manager};

use super::app_store::AppIconStore;
use crate::core::{AppError, Result};
use crate::db::apps;
use crate::db::models::{ClipboardApp, Platform};

#[derive(Clone)]
pub struct AppsRegistry {
    app: AppHandle,
    icon_store: AppIconStore,
    cache: Arc<RwLock<HashMap<String, ClipboardApp>>>,
}

impl AppsRegistry {
    /// 创建来源应用注册表，共享 App 句柄、图标仓库和内存缓存。
    pub fn new(app: AppHandle, icon_store: AppIconStore) -> Self {
        Self {
            app,
            icon_store,
            cache: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    /// 读取当前热替换后的数据库连接池。
    async fn pool(&self) -> SqlitePool {
        self.app.state::<crate::db::DatabaseState>().pool().await
    }

    /// 把 DB 中已有的应用全部装进缓存。启动期调用一次，覆盖任何旧缓存内容。
    pub async fn load_from_db(&self) -> Result<()> {
        let pool = self.pool().await;
        let all = apps::list_all_apps(&pool).await?;
        let mut cache = self.cache.write().expect("apps registry cache poisoned");
        cache.clear();
        for app in all {
            cache.insert(app.id.clone(), app);
        }
        Ok(())
    }

    /// 按 id 从内存缓存读取来源应用记录。
    pub fn get(&self, id: &str) -> Option<ClipboardApp> {
        self.cache
            .read()
            .expect("apps registry cache poisoned")
            .get(id)
            .cloned()
    }

    /// 写入或替换内存缓存中的来源应用记录。
    pub fn insert_into_cache(&self, app: ClipboardApp) {
        self.cache
            .write()
            .expect("apps registry cache poisoned")
            .insert(app.id.clone(), app);
    }

    /// 从内存缓存移除来源应用记录。
    pub fn remove_from_cache(&self, id: &str) {
        self.cache
            .write()
            .expect("apps registry cache poisoned")
            .remove(id);
    }
}

/// 刷新当前运行中的用户应用，并返回本次枚举到的应用列表。
pub async fn refresh_running_apps(registry: AppsRegistry) -> Result<Vec<ClipboardApp>> {
    let metas = tauri::async_runtime::spawn_blocking(running_app_metas)
        .await
        .map_err(|err| AppError::Other(anyhow!("running app refresh task join failed: {err}")))?;

    Ok(materialize_metas(&registry, metas))
}

/// 从用户选择的应用路径构建来源应用并写入注册表。
pub async fn add_app_from_path(registry: AppsRegistry, path: String) -> Result<ClipboardApp> {
    let meta =
        tauri::async_runtime::spawn_blocking(move || app_meta_from_path(PathBuf::from(path)))
            .await
            .map_err(|err| AppError::Other(anyhow!("app add task join failed: {err}")))??;
    let mut apps = upsert_metas(&registry, vec![meta]).await?;

    apps.pop()
        .ok_or_else(|| AppError::Clipboard("app metadata is empty".to_owned()))
}

/// 按应用 id 批量补全应用信息，返回成功物化的应用。
pub async fn add_apps_from_ids(
    registry: AppsRegistry,
    ids: Vec<String>,
) -> Result<Vec<ClipboardApp>> {
    let metas = tauri::async_runtime::spawn_blocking(move || {
        ids.into_iter()
            .filter_map(|id| app_meta_from_id(&id))
            .collect::<Vec<_>>()
    })
    .await
    .map_err(|err| AppError::Other(anyhow!("app lookup task join failed: {err}")))?;

    upsert_metas(&registry, metas).await
}

/// 删除未被历史记录引用的来源应用，并同步移除注册表缓存。
pub async fn delete_unreferenced_apps(
    registry: AppsRegistry,
    ids: Vec<String>,
) -> Result<Vec<String>> {
    let pool = registry.pool().await;
    let deleted = apps::delete_unreferenced_apps(&pool, &ids).await?;

    for id in &deleted {
        registry.remove_from_cache(id);
    }

    Ok(deleted)
}

/// 将元数据列表物化为应用记录，写入内存缓存并同步抽取图标。
fn materialize_metas(registry: &AppsRegistry, metas: Vec<ScannedAppMeta>) -> Vec<ClipboardApp> {
    let now = Utc::now();
    let mut apps_out = Vec::with_capacity(metas.len());

    for meta in metas {
        let existing_icon = registry.get(&meta.id).and_then(|app| app.icon_file);
        let icon_file = match existing_icon {
            Some(icon_file) => Some(icon_file),
            None => meta
                .path
                .as_deref()
                .and_then(|path| super::icon::icon_png(path, None))
                .as_deref()
                .and_then(|bytes| match registry.icon_store.store(bytes) {
                    Ok(name) => Some(name),
                    Err(err) => {
                        log::warn!("app icon store failed for {}: {err}", meta.id);
                        None
                    }
                }),
        };
        let app = ClipboardApp {
            id: meta.id,
            name: meta.name,
            icon_file,
            platform: meta.platform,
            created_at: now,
            updated_at: now,
        };
        registry.insert_into_cache(app.clone());
        apps_out.push(app);
    }

    apps_out
}

/// 将元数据列表写入 DB 与缓存。
async fn upsert_metas(
    registry: &AppsRegistry,
    metas: Vec<ScannedAppMeta>,
) -> Result<Vec<ClipboardApp>> {
    let apps = materialize_metas(registry, metas);
    let pool = registry.pool().await;

    for app in &apps {
        apps::upsert_app(&pool, app).await?;
    }

    Ok(apps)
}

pub struct ScannedAppMeta {
    pub id: String,
    pub name: String,
    pub path: Option<PathBuf>,
    pub platform: Platform,
}

fn running_app_metas() -> Vec<ScannedAppMeta> {
    #[cfg(target_os = "macos")]
    {
        macos::running_app_metas()
    }
    #[cfg(target_os = "windows")]
    {
        windows::running_app_metas()
    }
    #[cfg(not(any(target_os = "macos", target_os = "windows")))]
    {
        Vec::new()
    }
}

fn app_meta_from_path(path: PathBuf) -> Result<ScannedAppMeta> {
    #[cfg(target_os = "macos")]
    {
        macos::app_meta_from_path(&path)
    }
    #[cfg(target_os = "windows")]
    {
        windows::app_meta_from_path(&path)
    }
    #[cfg(not(any(target_os = "macos", target_os = "windows")))]
    {
        let _ = path;
        Err(AppError::Clipboard("unsupported platform".to_owned()))
    }
}

fn app_meta_from_id(id: &str) -> Option<ScannedAppMeta> {
    #[cfg(target_os = "macos")]
    {
        macos::app_meta_from_id(id)
    }
    #[cfg(target_os = "windows")]
    {
        let _ = id;
        None
    }
    #[cfg(not(any(target_os = "macos", target_os = "windows")))]
    {
        let _ = id;
        None
    }
}

fn validate_existing_path(path: &Path) -> Result<()> {
    if !path.exists() {
        return Err(AppError::Clipboard("app path does not exist".to_owned()));
    }

    Ok(())
}

#[cfg(target_os = "macos")]
mod macos {
    use super::{validate_existing_path, ScannedAppMeta};
    use crate::core::{AppError, Result};
    use crate::db::models::Platform;
    use objc2::msg_send;
    use objc2::rc::{autoreleasepool, Retained};
    use objc2_app_kit::{NSApplicationActivationPolicy, NSRunningApplication, NSWorkspace};
    use objc2_foundation::{NSString, NSURL};
    use std::collections::HashSet;
    use std::path::{Path, PathBuf};

    pub fn running_app_metas() -> Vec<ScannedAppMeta> {
        autoreleasepool(|_| {
            let workspace = NSWorkspace::sharedWorkspace();
            let apps = workspace.runningApplications();
            let mut out = Vec::new();
            let mut seen = HashSet::new();

            for app in apps.iter() {
                if app.activationPolicy() != NSApplicationActivationPolicy::Regular {
                    continue;
                }
                let Some(id) = app.bundleIdentifier().map(|s| s.to_string()) else {
                    continue;
                };
                if !seen.insert(id.clone()) {
                    continue;
                }
                let name = app
                    .localizedName()
                    .map(|s| s.to_string())
                    .unwrap_or_else(|| id.clone());
                let path = unsafe { bundle_path(&app) };

                out.push(ScannedAppMeta {
                    id,
                    name,
                    path,
                    platform: Platform::Macos,
                });
            }

            out
        })
    }

    pub fn app_meta_from_path(path: &Path) -> Result<ScannedAppMeta> {
        validate_existing_path(path)?;
        if path.extension().and_then(|s| s.to_str()) != Some("app") {
            return Err(AppError::Clipboard(
                "please choose a macOS app bundle".to_owned(),
            ));
        }

        scan_app_bundle(path)
            .ok_or_else(|| AppError::Clipboard("app bundle metadata is invalid".to_owned()))
    }

    pub fn app_meta_from_id(id: &str) -> Option<ScannedAppMeta> {
        for path in known_app_bundle_paths(id) {
            if let Some(meta) = scan_app_bundle(&path) {
                return Some(meta);
            }
        }

        None
    }

    fn scan_app_bundle(path: &Path) -> Option<ScannedAppMeta> {
        let info_path = path.join("Contents/Info.plist");
        let info = match plist::Value::from_file(&info_path) {
            Ok(value) => value,
            Err(err) => {
                log::debug!("app meta: parse {} failed: {err}", info_path.display());
                return None;
            }
        };
        let dict = info.as_dictionary()?;
        let id = dict.get("CFBundleIdentifier")?.as_string()?.to_owned();

        let fallback_name = dict
            .get("CFBundleDisplayName")
            .and_then(|v| v.as_string())
            .or_else(|| dict.get("CFBundleName").and_then(|v| v.as_string()))
            .map(str::to_owned)
            .unwrap_or_else(|| {
                path.file_stem()
                    .and_then(|s| s.to_str())
                    .unwrap_or(&id)
                    .to_owned()
            });
        let name = localized_bundle_name(path).unwrap_or(fallback_name);

        Some(ScannedAppMeta {
            id,
            name,
            path: Some(path.to_path_buf()),
            platform: Platform::Macos,
        })
    }

    /// 通过 NSRunningApplication.bundleURL 拿到 .app 路径。
    unsafe fn bundle_path(app: &NSRunningApplication) -> Option<PathBuf> {
        let url: Option<Retained<NSURL>> = msg_send![app, bundleURL];
        let url = url?;
        let path: Option<Retained<NSString>> = msg_send![&*url, path];
        Some(PathBuf::from(path?.to_string()))
    }

    /// 取 Finder 展示的本地化名称，拿不到时返回 None。
    fn localized_bundle_name(path: &Path) -> Option<String> {
        let output = std::process::Command::new("/usr/bin/mdls")
            .args(["-name", "kMDItemDisplayName", "-raw"])
            .arg(path)
            .output()
            .ok()?;
        if !output.status.success() {
            return None;
        }
        let raw = String::from_utf8_lossy(&output.stdout);
        let text = raw.trim().trim_end_matches('\0').trim();
        if text.is_empty() || text == "(null)" {
            return None;
        }
        let cleaned = text.strip_suffix(".app").unwrap_or(text);
        if cleaned.is_empty() {
            None
        } else {
            Some(cleaned.to_owned())
        }
    }

    fn known_app_bundle_paths(id: &str) -> Vec<PathBuf> {
        if let Some(path) = path_from_spotlight(id) {
            return vec![path];
        }

        match id {
            "com.apple.keychainaccess" => {
                vec![PathBuf::from(
                    "/System/Library/CoreServices/Applications/Keychain Access.app",
                )]
            }
            "com.apple.Passwords" => vec![PathBuf::from("/System/Applications/Passwords.app")],
            _ => Vec::new(),
        }
    }

    fn path_from_spotlight(id: &str) -> Option<PathBuf> {
        let query = format!("kMDItemCFBundleIdentifier == \"{id}\"");
        let output = std::process::Command::new("/usr/bin/mdfind")
            .arg(query)
            .output()
            .ok()?;
        if !output.status.success() {
            return None;
        }

        String::from_utf8_lossy(&output.stdout)
            .lines()
            .map(str::trim)
            .filter(|line| !line.is_empty())
            .map(PathBuf::from)
            .find(|path| path.extension().and_then(|s| s.to_str()) == Some("app"))
    }

    #[cfg(test)]
    mod tests {
        use super::*;

        #[test]
        fn passwords_app_uses_localized_name() {
            let path = std::path::PathBuf::from("/System/Applications/Passwords.app");
            if !path.exists() {
                eprintln!("skip: Passwords.app not present");
                return;
            }
            let localized = localized_bundle_name(&path);
            println!("Passwords localized: {localized:?}");
            assert!(localized.is_some());
        }

        #[test]
        fn keychain_access_can_be_found_by_bundle_id() {
            let Some(meta) = app_meta_from_id("com.apple.keychainaccess") else {
                eprintln!("skip: Keychain Access.app not indexed");
                return;
            };
            let Some(path) = meta.path else {
                panic!("expected Keychain Access path");
            };

            let png = crate::clipboard::icon_png(&path, None).expect("expected app icon");

            assert_eq!(meta.id, "com.apple.keychainaccess");
            assert!(png.len() > 100);
        }
    }
}

#[cfg(target_os = "windows")]
mod windows {
    use super::{validate_existing_path, ScannedAppMeta};
    use crate::core::{AppError, Result};
    use crate::db::models::Platform;
    use std::collections::HashSet;
    use std::ffi::OsString;
    use std::os::windows::ffi::OsStringExt;
    use std::path::{Path, PathBuf};
    use winapi::shared::minwindef::{BOOL, DWORD, FALSE, LPARAM, TRUE};
    use winapi::shared::windef::HWND;
    use winapi::um::handleapi::CloseHandle;
    use winapi::um::processthreadsapi::OpenProcess;
    use winapi::um::winbase::QueryFullProcessImageNameW;
    use winapi::um::winnt::PROCESS_QUERY_LIMITED_INFORMATION;
    use winapi::um::winuser::{
        EnumWindows, GetWindowTextLengthW, GetWindowThreadProcessId, IsWindowVisible,
    };

    pub fn running_app_metas() -> Vec<ScannedAppMeta> {
        let mut paths = Vec::<PathBuf>::new();
        unsafe {
            EnumWindows(
                Some(enum_visible_window_paths),
                &mut paths as *mut Vec<PathBuf> as LPARAM,
            );
        }

        let mut out = Vec::new();
        let mut seen = HashSet::new();
        for path in paths {
            let path = normalize_exe_path(path);
            let id = path.to_string_lossy().to_string();
            if !seen.insert(id.clone()) {
                continue;
            }
            out.push(ScannedAppMeta {
                id,
                name: exe_display_name(&path),
                path: Some(path),
                platform: Platform::Windows,
            });
        }

        out
    }

    pub fn app_meta_from_path(path: &Path) -> Result<ScannedAppMeta> {
        validate_existing_path(path)?;
        let extension = path
            .extension()
            .and_then(|s| s.to_str())
            .map(str::to_ascii_lowercase);
        if extension.as_deref() != Some("exe") {
            return Err(AppError::Clipboard(
                "please choose a Windows executable".to_owned(),
            ));
        }

        let normalized = path
            .canonicalize()
            .map(normalize_verbatim_path)
            .map_err(|_| AppError::Clipboard("app path cannot be resolved".to_owned()))?;
        Ok(ScannedAppMeta {
            id: normalized.to_string_lossy().to_string(),
            name: exe_display_name(&normalized),
            path: Some(normalized),
            platform: Platform::Windows,
        })
    }

    unsafe extern "system" fn enum_visible_window_paths(hwnd: HWND, lparam: LPARAM) -> BOOL {
        if IsWindowVisible(hwnd) == FALSE || GetWindowTextLengthW(hwnd) == 0 {
            return TRUE;
        }

        let mut pid: DWORD = 0;
        GetWindowThreadProcessId(hwnd, &mut pid);
        if pid == 0 {
            return TRUE;
        }
        if let Some(path) = process_exe_path(pid) {
            let paths = &mut *(lparam as *mut Vec<PathBuf>);
            paths.push(path);
        }

        TRUE
    }

    unsafe fn process_exe_path(pid: DWORD) -> Option<PathBuf> {
        let handle = OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, FALSE, pid);
        if handle.is_null() {
            return None;
        }

        let mut buf = [0u16; 1024];
        let mut size: DWORD = buf.len() as DWORD;
        let ok = QueryFullProcessImageNameW(handle, 0, buf.as_mut_ptr(), &mut size);
        CloseHandle(handle);

        if ok == 0 || size == 0 {
            return None;
        }

        Some(PathBuf::from(OsString::from_wide(&buf[..size as usize])))
    }

    fn normalize_exe_path(path: PathBuf) -> PathBuf {
        path.canonicalize()
            .map(normalize_verbatim_path)
            .unwrap_or(path)
    }

    /// Converts Windows verbatim paths back to paths accepted by Shell icon APIs.
    fn normalize_verbatim_path(path: PathBuf) -> PathBuf {
        let raw = path.to_string_lossy();

        if let Some(rest) = raw.strip_prefix(r"\\?\UNC\") {
            return PathBuf::from(format!(r"\\{rest}"));
        }
        if let Some(rest) = raw.strip_prefix(r"\\?\") {
            return PathBuf::from(rest);
        }

        path
    }

    fn exe_display_name(path: &Path) -> String {
        if let Some(name) = path.file_stem().and_then(|s| s.to_str()) {
            return name.to_owned();
        }

        path.to_string_lossy().into_owned()
    }
}

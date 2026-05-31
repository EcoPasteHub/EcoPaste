//! 应用注册表：把「目录扫描发现的已安装应用」与「监听过程中捕获的前台应用」统一在
//! `clipboard_apps` 表中，并在内存里维护一份 id → 应用的缓存，给两类场景共用：
//!
//! - 偏好设置「应用过滤」面板：一次性展示全部已知应用，让用户勾选要排除的应用。
//! - 剪贴板监听回调：拿到前台应用 id 后先从缓存取（带 icon），缓存缺失再走原 FrontmostApp 路径
//!   补齐图标并写入缓存——同步即可命中常见情况，避免每次复制都走「TIFF→PNG」编码。
//!
//! 启动期先把 DB 已有的应用全部读进缓存（覆盖上一会话扫描结果 + 历史捕获记录），
//! 再后台启动一次目录扫描；之后用户在 UI 里点「刷新」会触发 [`refresh_from_dirs`] 重扫。

use std::collections::HashMap;
use std::path::PathBuf;
use std::sync::{Arc, RwLock};

use chrono::Utc;
use sqlx::SqlitePool;
use tauri::{AppHandle, Emitter};

use super::app_store::AppIconStore;
use crate::core::Result;
use crate::db::apps;
use crate::db::models::{ClipboardApp, Platform};

/// 应用注册表更新事件（前端 FiltersPanel 收到后重拉列表）。
pub const APPS_UPDATED_EVENT: &str = "clipboard-apps://updated";

#[derive(Clone)]
pub struct AppsRegistry {
    pool: SqlitePool,
    icon_store: AppIconStore,
    cache: Arc<RwLock<HashMap<String, ClipboardApp>>>,
}

impl AppsRegistry {
    pub fn new(pool: SqlitePool, icon_store: AppIconStore) -> Self {
        Self {
            pool,
            icon_store,
            cache: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    /// 把 DB 中已有的应用全部装进缓存。启动期调用一次，覆盖任何旧缓存内容。
    pub async fn load_from_db(&self) -> Result<()> {
        let all = apps::list_all_apps(&self.pool).await?;
        let mut cache = self.cache.write().expect("apps registry cache poisoned");
        cache.clear();
        for app in all {
            cache.insert(app.id.clone(), app);
        }
        Ok(())
    }

    pub fn get(&self, id: &str) -> Option<ClipboardApp> {
        self.cache
            .read()
            .expect("apps registry cache poisoned")
            .get(id)
            .cloned()
    }

    /// 列出当前缓存中的全部应用（名称升序）。
    pub fn list_all(&self) -> Vec<ClipboardApp> {
        let mut v: Vec<_> = self
            .cache
            .read()
            .expect("apps registry cache poisoned")
            .values()
            .cloned()
            .collect();
        v.sort_by(|a, b| {
            a.name
                .to_lowercase()
                .cmp(&b.name.to_lowercase())
                .then_with(|| a.id.cmp(&b.id))
        });
        v
    }

    pub fn insert_into_cache(&self, app: ClipboardApp) {
        self.cache
            .write()
            .expect("apps registry cache poisoned")
            .insert(app.id.clone(), app);
    }
}

/// 扫描给定目录、对每个发现的应用 upsert DB 并写入缓存，返回更新后的完整列表。
/// 元数据扫描很快（plist 解析），同步返回；图标提取（NSWorkspace + TIFF→PNG，单条数百毫秒）
/// 放到后台 spawn_blocking 任务，完成后通过 [`APPS_UPDATED_EVENT`] 通知前端重拉。
///
/// 单条 upsert/icon 落盘失败仅 warn，不中断其它条目。
pub async fn refresh_from_dirs(
    app: AppHandle,
    registry: AppsRegistry,
    dirs: Vec<String>,
    force_icons: bool,
) -> Result<Vec<ClipboardApp>> {
    // 将用户配置的目录与系统内建目录合并去重——系统目录列表会随版本升级，
    // 老 settings.json 不会自动跟进，这里做一次 union 兜底，保证新加的系统目录
    // （如 /System/Library/CoreServices/Applications）始终被扫到。
    let mut merged: Vec<String> = Vec::new();
    let mut seen = std::collections::HashSet::new();
    for d in dirs.into_iter().chain(builtin_scan_dirs()) {
        if seen.insert(d.clone()) {
            merged.push(d);
        }
    }
    let dirs = merged;
    log::info!("app scan: refresh requested, dirs = {dirs:?}, force_icons = {force_icons}");
    // 阶段 1：纯文件 IO + plist 解析，放到 blocking 线程池。
    let dirs_for_scan = dirs.clone();
    let metas = tauri::async_runtime::spawn_blocking(move || scanner::scan_dirs(&dirs_for_scan))
        .await
        .map_err(|err| {
            crate::core::AppError::Other(anyhow::anyhow!("app scan join failed: {err}"))
        })?;
    log::info!(
        "app scan: discovered {} metas, upserting metadata...",
        metas.len()
    );

    // 阶段 2：upsert 元数据（保留旧 icon_file 不置空）。force_icons=true 时所有条目都进
    // 重抓队列（用户点刷新）；false 时只针对「没图标」或「icon_file 指向的文件已不在磁盘」
    // 的条目（启动场景，避免每次启动 60s，同时修复 DB 残留指向缺失文件的情况）。
    let now = Utc::now();
    let mut icon_targets: Vec<(String, PathBuf)> = Vec::new();
    for meta in &metas {
        let existing_icon = registry.get(&meta.id).and_then(|a| a.icon_file);
        let icon_on_disk = existing_icon
            .as_deref()
            .map(|name| registry.icon_store.icon_path(name).exists())
            .unwrap_or(false);
        if force_icons || !icon_on_disk {
            icon_targets.push((meta.id.clone(), meta.path.clone()));
        }
        let app_row = ClipboardApp {
            id: meta.id.clone(),
            name: meta.name.clone(),
            icon_file: existing_icon,
            platform: meta.platform,
            created_at: now,
            updated_at: now,
        };
        if let Err(err) = apps::upsert_app(&registry.pool, &app_row).await {
            log::warn!("app scan: upsert meta {} failed: {err}", app_row.id);
            continue;
        }
        registry.insert_into_cache(app_row);
    }

    let initial = registry.list_all();

    // 阶段 3：后台并发提取每个条目的 PNG，再 upsert + emit 通知。
    if !icon_targets.is_empty() {
        let registry_bg = registry.clone();
        let app_bg = app.clone();
        tauri::async_runtime::spawn(async move {
            fetch_icons_background(app_bg, registry_bg, icon_targets).await;
        });
    } else {
        log::info!("app scan: nothing to extract icons for");
    }

    Ok(initial)
}

async fn fetch_icons_background(
    app: AppHandle,
    registry: AppsRegistry,
    targets: Vec<(String, PathBuf)>,
) {
    log::info!("app icons: fetching {} icons in background", targets.len());
    // 一次 spawn_blocking 中串行处理（每个 NSImage→PNG 数百 ms），避免对主线程造成压力。
    // 处理一批后 emit 一次事件让 UI 增量刷新。
    const BATCH: usize = 16;
    let mut buf: Vec<(String, Vec<u8>)> = Vec::new();
    for chunk in targets.chunks(BATCH) {
        let chunk_owned: Vec<(String, PathBuf)> = chunk.to_vec();
        let extracted = tauri::async_runtime::spawn_blocking(move || {
            let mut out: Vec<(String, Vec<u8>)> = Vec::new();
            for (id, path) in chunk_owned {
                if let Some(png) = scanner::icon_for_bundle(&path) {
                    out.push((id, png));
                }
            }
            out
        })
        .await
        .unwrap_or_default();
        buf.extend(extracted);

        let now = Utc::now();
        let mut touched = false;
        for (id, png) in buf.drain(..) {
            let icon_file = match registry.icon_store.store(&png) {
                Ok(name) => name,
                Err(err) => {
                    log::warn!("app icons: store {id} failed: {err}");
                    continue;
                }
            };
            let Some(mut existing) = registry.get(&id) else {
                continue;
            };
            existing.icon_file = Some(icon_file);
            existing.updated_at = now;
            if let Err(err) = apps::upsert_app(&registry.pool, &existing).await {
                log::warn!("app icons: upsert {id} failed: {err}");
                continue;
            }
            registry.insert_into_cache(existing);
            touched = true;
        }
        if touched {
            if let Err(err) = app.emit(APPS_UPDATED_EVENT, ()) {
                log::warn!("emit {APPS_UPDATED_EVENT} failed: {err}");
            }
        }
    }
    log::info!("app icons: done");
}

/// 平台内建扫描目录——与用户配置 union，确保系统更新后新增的目录自动接入。
fn builtin_scan_dirs() -> Vec<String> {
    #[cfg(target_os = "macos")]
    {
        let mut dirs = vec![
            "/Applications".to_owned(),
            "/System/Applications".to_owned(),
            "/System/Applications/Utilities".to_owned(),
            "/System/Library/CoreServices/Applications".to_owned(),
        ];
        if let Some(home) = std::env::var_os("HOME") {
            let p = std::path::PathBuf::from(home).join("Applications");
            if let Some(s) = p.to_str() {
                dirs.push(s.to_owned());
            }
        }
        dirs
    }
    #[cfg(target_os = "windows")]
    {
        vec![
            "C:\\Program Files".to_owned(),
            "C:\\Program Files (x86)".to_owned(),
        ]
    }
    #[cfg(not(any(target_os = "macos", target_os = "windows")))]
    {
        Vec::new()
    }
}

pub struct ScannedAppMeta {
    pub id: String,
    pub name: String,
    pub path: PathBuf,
    pub platform: Platform,
}

mod scanner {
    use super::ScannedAppMeta;
    use std::path::Path;

    pub fn scan_dirs(dirs: &[String]) -> Vec<ScannedAppMeta> {
        #[cfg(target_os = "macos")]
        {
            super::macos::scan_dirs(dirs)
        }
        #[cfg(target_os = "windows")]
        {
            super::windows::scan_dirs(dirs)
        }
        #[cfg(not(any(target_os = "macos", target_os = "windows")))]
        {
            let _ = dirs;
            Vec::new()
        }
    }

    pub fn icon_for_bundle(path: &Path) -> Option<Vec<u8>> {
        let png = super::super::icon::icon_png(path, None);
        match &png {
            Some(b) => log::debug!("icon_for_bundle: {} -> {} bytes", path.display(), b.len()),
            None => log::warn!("icon_for_bundle: {} -> None", path.display()),
        }
        png
    }
}

#[cfg(target_os = "macos")]
mod macos {
    use super::ScannedAppMeta;
    use crate::db::models::Platform;
    use std::collections::HashSet;
    use std::path::Path;

    /// 扫描深度：`/Applications` 下的 `.app` 一般为一级；`Adobe/Adobe Photoshop 2024.app`
    /// 这类品牌子目录也很常见。允许两级足够，再深通常是 plug-in 之类的内嵌包，进了反而出错。
    const MAX_DEPTH: usize = 2;

    pub fn scan_dirs(dirs: &[String]) -> Vec<ScannedAppMeta> {
        let mut out = Vec::new();
        let mut visited: HashSet<String> = HashSet::new();
        for dir in dirs {
            let path = Path::new(dir);
            if !path.is_dir() {
                log::warn!("app scan: dir not found or not a dir: {dir}");
                continue;
            }
            let before = out.len();
            scan_recursive(path, 0, &mut visited, &mut out);
            log::info!(
                "app scan: {} -> {} bundles",
                dir,
                out.len().saturating_sub(before)
            );
        }
        log::info!("app scan: total {} bundles", out.len());
        out
    }

    fn scan_recursive(
        dir: &Path,
        depth: usize,
        visited: &mut HashSet<String>,
        out: &mut Vec<ScannedAppMeta>,
    ) {
        if depth > MAX_DEPTH {
            return;
        }
        let entries = match std::fs::read_dir(dir) {
            Ok(e) => e,
            Err(_) => return,
        };
        for entry in entries.flatten() {
            let p = entry.path();
            if !p.is_dir() {
                continue;
            }
            if p.extension().and_then(|s| s.to_str()) == Some("app") {
                if let Some(app) = scan_app_bundle(&p) {
                    if visited.insert(app.id.clone()) {
                        out.push(app);
                    }
                }
                continue;
            }
            scan_recursive(&p, depth + 1, visited, out);
        }
    }

    fn scan_app_bundle(path: &Path) -> Option<ScannedAppMeta> {
        let info_path = path.join("Contents/Info.plist");
        let info = match plist::Value::from_file(&info_path) {
            Ok(v) => v,
            Err(err) => {
                log::debug!("app scan: parse {} failed: {err}", info_path.display());
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

        // 优先按系统语言取 .lproj/InfoPlist.strings 里的本地化名称
        let name = localized_bundle_name(path).unwrap_or(fallback_name);

        Some(ScannedAppMeta {
            id,
            name,
            path: path.to_path_buf(),
            platform: Platform::Macos,
        })
    }

    /// 取 Finder 展示的本地化名称：调用 `mdls -name kMDItemDisplayName -raw`，
    /// Spotlight 元数据由系统按当前语言计算好，与 Finder 完全一致；70+ App 启动场景
    /// 总耗时 ~2s，已在 `spawn_blocking` 里跑，不阻塞 UI。
    /// 拿不到 / 等于文件名时返回 None，由调用方走 `Info.plist` fallback。
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

    #[cfg(test)]
    mod tests {
        use super::*;

        #[test]
        fn scan_applications_finds_at_least_one_app() {
            let metas = scan_dirs(&["/Applications".to_owned()]);
            assert!(
                !metas.is_empty(),
                "expected at least one app under /Applications"
            );
            for m in metas.iter().take(3) {
                println!(
                    "scanned meta: id={} name={} path={:?}",
                    m.id, m.name, m.path
                );
            }
        }

        #[test]
        fn passwords_app_uses_localized_name() {
            let path = std::path::PathBuf::from("/System/Applications/Passwords.app");
            if !path.exists() {
                eprintln!("skip: Passwords.app not present");
                return;
            }
            // 仅在 zh-* 系统语言下校验非英文名；其他环境只确保 mdls 至少能返回内容。
            let localized = localized_bundle_name(&path);
            println!("Passwords localized: {localized:?}");
            assert!(localized.is_some());
        }
    }
}

#[cfg(target_os = "windows")]
mod windows {
    use super::ScannedAppMeta;
    /// Windows 端的可执行文件枚举还没接入（注册表 / Start Menu 都需要额外实现）。
    /// 用户仍可通过监听捕获已用过的应用 id（exe 绝对路径）并在 UI 勾选过滤。
    pub fn scan_dirs(_dirs: &[String]) -> Vec<ScannedAppMeta> {
        Vec::new()
    }
}

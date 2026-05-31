//! 剪贴板相关命令：手动重新读取、解析图片路径。供前端按需触发。

use std::sync::Arc;
use std::time::Duration;

use serde::Serialize;
use sqlx::SqlitePool;
use tauri::{AppHandle, Manager, State};

use crate::clipboard::{
    build_item, detect_frontmost, materialize_source, persist_and_notify, refresh_from_dirs,
    AppIconStore, AppsRegistry, ClipboardReader, ImageStore, WritebackGuard,
};
use crate::core::{AppError, Result};
use crate::db::items::find_item_by_id;
use crate::db::models::{ClipboardGroup, ClipboardItem, ClipboardItemQuery};
use crate::window::{self, MAIN_WINDOW_LABEL};

/// `read_clipboard` 的返回：入库后的记录 + 是否命中去重（前端据此决定提示/滚动行为）。
#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ReadClipboardResult {
    pub item: ClipboardItem,
    pub deduplicated: bool,
    /// 剪贴板为空 / 无可识别内容时为 `false`，此时 `item` 为 `None`。
    pub captured: bool,
}

/// 手动读取当前剪贴板并入库（「重新读取」按钮）。
///
/// 复用监听管线：read_all → build_item（含图片落盘）→ persist_and_notify（去重入库 + emit）。
/// 与 OS 监听走同一条入库路径，语义一致；emit 同样触发前端列表刷新。
///
/// `ClipboardReader` 持有的 `ClipboardContext` 是 `!Send`，故读取与转换全部在
/// await 之前的同步块内完成并 drop，之后才进入异步入库——保证命令 future 满足 `Send`。
#[tauri::command]
pub async fn read_clipboard(
    app: AppHandle,
    pool: State<'_, SqlitePool>,
    store: State<'_, ImageStore>,
    app_icon_store: State<'_, AppIconStore>,
    registry: State<'_, AppsRegistry>,
) -> Result<Option<ReadClipboardResult>> {
    // 先在同步段抓前台应用 + 读取剪贴板。
    // 注意：手动「重新读取」时前台应用就是 EcoPaste 自己，不像 OS 监听场景能拿到原应用——
    // 这里仍保留探测：若用户在外部应用复制后立刻命令式触发，多少能捕获到正确来源；
    // 命中我们自己也无害（apps 表只是多记一条「EcoPaste」）。
    let (item_opt, source) = {
        let source = detect_frontmost();
        let reader = ClipboardReader::new()?;
        let payload = reader.read_all()?;
        let item = match payload {
            Some(payload) => build_item(&store, &payload)?,
            None => None,
        };
        (item, source)
        // reader 在此 drop，!Send 句柄不跨下方 await。
    };

    let Some(mut item) = item_opt else {
        return Ok(None);
    };

    let source_app = source.map(|src| materialize_source(&app_icon_store, Some(&registry), src));
    if let Some(src) = &source_app {
        item.source_app_id = Some(src.id.clone());
    }

    let result = persist_and_notify(&app, &pool, &item, source_app.as_ref()).await?;
    Ok(Some(ReadClipboardResult {
        item,
        deduplicated: result.deduplicated,
        captured: true,
    }))
}

/// 把入库的图片文件名解析为磁盘绝对路径，供前端预览取图。
/// `thumbnail = true` 取缩略图，否则取原图。
///
/// 缩略图按需生成：不存在时在 `spawn_blocking` 里从原图解码 + 缩放 + 编码后落盘，再返回路径。
/// 解码/编码是 CPU 阻塞活，放到阻塞线程池避免占用 async worker；且「返回时文件已确保存在」，
/// 前端拿到路径即可安全 `convertFileSrc` 加载，不会撞到半成品文件。
///
/// `file_name` 来自前端（即记录的 `content`），是唯一的外部输入，需防路径穿越：
/// 仅允许「单层、纯 `<hash>.png` 形态」的文件名，含分隔符 / `..` / 子目录一律拒绝。
#[tauri::command]
pub async fn get_clipboard_image_path(
    store: State<'_, ImageStore>,
    file_name: String,
    thumbnail: bool,
) -> Result<String> {
    validate_image_file_name(&file_name)?;

    let path = if thumbnail {
        let store = store.inner().clone();
        tauri::async_runtime::spawn_blocking(move || store.ensure_thumbnail(&file_name))
            .await
            .map_err(|err| AppError::Clipboard(format!("thumbnail task join failed: {err}")))??
    } else {
        store.origin_path(&file_name)
    };

    path.to_str()
        .map(str::to_owned)
        .ok_or_else(|| AppError::Clipboard("image path is not valid utf-8".to_owned()))
}

/// 把 `clipboard_apps.icon_file`（形如 `<sha256>.png`）解析为磁盘绝对路径，
/// 供前端用 `convertFileSrc` 渲染。文件名复用图片同款防穿越校验。
#[tauri::command]
pub async fn get_clipboard_app_icon_path(
    store: State<'_, AppIconStore>,
    file_name: String,
) -> Result<String> {
    validate_image_file_name(&file_name)?;
    store
        .icon_path(&file_name)
        .to_str()
        .map(str::to_owned)
        .ok_or_else(|| AppError::Clipboard("app icon path is not valid utf-8".to_owned()))
}

/// 把指定历史记录写回系统剪贴板（不触发模拟粘贴，4.2 再补）。
/// `plain = true` 强制纯文本，剥离 HTML/RTF。
///
/// `ClipboardContext` 是 `!Send`，写回逻辑在 `clipboard::write_to_clipboard` 内同步完成，
/// 之后无 await，命令 future 仍满足 `Send`。
#[tauri::command]
pub async fn write_to_clipboard(
    pool: State<'_, SqlitePool>,
    store: State<'_, ImageStore>,
    guard: State<'_, Arc<WritebackGuard>>,
    id: String,
    plain: bool,
) -> Result<()> {
    let item = find_item_by_id(&pool, &id)
        .await?
        .ok_or_else(|| AppError::Clipboard(format!("clipboard item not found: {id}")))?;

    crate::clipboard::write_to_clipboard(&store, guard.inner().as_ref(), &item, plain)?;
    Ok(())
}

/// 「点击列表项 → 自动粘贴」的组合命令：写回剪贴板 + 隐藏主窗口 + 触发系统级粘贴。
///
/// 隐藏窗口是为了让 OS 把焦点交还给上一个应用——这样 ⌘V / Shift+Insert 投递到
/// 的是用户原本工作的窗口而非我们自己。等后续把主窗口改成「不抢占焦点」
/// （NSPanel `NonactivatingPanel` / Windows `WS_EX_NOACTIVATE`）后，这一步可拆除，
/// 但当前先按「显示即夺焦、粘贴时隐藏」的简单模型走，逻辑就在 Rust 一处闭环。
///
/// 隐藏后短暂等待让系统完成焦点切换；50ms 是经验值，旧版用 100ms 保守，
/// 这里取一半够用——超出会让用户感到延迟，过短则可能在前一个窗口还未成为
/// key window 时按键被自身吞掉。
#[tauri::command]
pub async fn paste_clipboard_item(
    app: AppHandle,
    pool: State<'_, SqlitePool>,
    store: State<'_, ImageStore>,
    guard: State<'_, Arc<WritebackGuard>>,
    id: String,
    plain: bool,
) -> Result<()> {
    let item = find_item_by_id(&pool, &id)
        .await?
        .ok_or_else(|| AppError::Clipboard(format!("clipboard item not found: {id}")))?;

    crate::clipboard::write_to_clipboard(&store, guard.inner().as_ref(), &item, plain)?;

    if let Err(err) = window::hide_window(&app, MAIN_WINDOW_LABEL) {
        log::warn!("hide main window before paste failed: {err:?}");
    }
    tokio::time::sleep(Duration::from_millis(50)).await;

    crate::keystroke::simulate_paste()?;
    Ok(())
}

/// 列表查询命令（薄封装）：参数缺省时走 Rust 端默认（limit=50, offset=0, createdAtDesc）；
/// `keyword` 非空时由 `query_items` 内部自动委派 FTS5。
#[tauri::command]
pub async fn list_clipboard_items(
    pool: State<'_, SqlitePool>,
    query: Option<ClipboardItemQuery>,
) -> Result<Vec<ClipboardItem>> {
    let q = query.unwrap_or_default();
    crate::db::items::query_items(&pool, &q).await
}

/// 单条查询命令：监听到 `clipboard://updated` 后前端按 id 拉单条，
/// 避免事件驱动刷新时整页 refetch。不存在返回 `None`，前端按需降级。
#[tauri::command]
pub async fn get_clipboard_item(
    pool: State<'_, SqlitePool>,
    id: String,
) -> Result<Option<ClipboardItem>> {
    find_item_by_id(&pool, &id).await
}

/// 按 id 列表批量取来源应用——前端渲染卡片时一次性补齐图标/名称。
#[tauri::command]
pub async fn list_clipboard_apps(
    pool: State<'_, SqlitePool>,
    ids: Vec<String>,
) -> Result<Vec<crate::db::models::ClipboardApp>> {
    crate::db::apps::list_apps_by_ids(&pool, &ids).await
}

/// 列出全部已知应用（目录扫描发现 + 监听过程捕获的并集）+ 设置里勾选了但 DB 尚无记录的
/// 「占位条目」，确保用户当前的过滤选择在 UI 上始终可见可取消。
/// 直接走 DB 避免与启动期异步装载缓存的竞态。
#[tauri::command]
pub async fn list_all_apps(
    app: AppHandle,
    pool: State<'_, SqlitePool>,
) -> Result<Vec<crate::db::models::ClipboardApp>> {
    use crate::db::models::{ClipboardApp, Platform};
    use std::collections::HashSet;

    let mut apps = crate::db::apps::list_all_apps(&pool).await?;
    let known: HashSet<String> = apps.iter().map(|a| a.id.clone()).collect();

    let excluded = app
        .state::<crate::settings::SettingsStore>()
        .snapshot()
        .clipboard
        .filters
        .excluded_app_ids;

    let now = chrono::Utc::now();
    let platform = if cfg!(target_os = "macos") {
        Platform::Macos
    } else {
        Platform::Windows
    };
    for id in excluded {
        if known.contains(&id) {
            continue;
        }
        apps.push(ClipboardApp {
            name: id.clone(),
            id,
            icon_file: None,
            platform,
            created_at: now,
            updated_at: now,
        });
    }

    // 保持按名称升序，让占位项也归入正确位置。
    apps.sort_by(|a, b| {
        a.name
            .to_lowercase()
            .cmp(&b.name.to_lowercase())
            .then_with(|| a.id.cmp(&b.id))
    });
    Ok(apps)
}

/// 手动触发一次目录扫描：按当前 `settings.clipboard.filters.scanDirs` 重新枚举
/// 已安装应用，更新 DB + 内存缓存，返回最新完整列表。耗时较长（取决于目录条目数），
/// 前端按钮调用时建议带 loading 态。
#[tauri::command]
pub async fn refresh_apps(
    app: AppHandle,
    registry: State<'_, AppsRegistry>,
) -> Result<Vec<crate::db::models::ClipboardApp>> {
    let dirs = app
        .state::<crate::settings::SettingsStore>()
        .snapshot()
        .clipboard
        .filters
        .scan_dirs
        .clone();
    refresh_from_dirs(app, registry.inner().clone(), dirs, true).await
}

/// 列出全部分组（薄封装），供前端构建分组 tab 栏。
/// 按 `sort_order` 升序，同序按 `created_at` 兜底。
#[tauri::command]
pub async fn list_clipboard_groups(pool: State<'_, SqlitePool>) -> Result<Vec<ClipboardGroup>> {
    crate::db::groups::list_groups(&pool).await
}

/// 翻转收藏状态（薄封装）。前端在调用前/后做乐观更新；这里不返回新状态，
/// 失败时前端按需回滚 / 重拉单条。
#[tauri::command]
pub async fn toggle_clipboard_item_favorite(pool: State<'_, SqlitePool>, id: String) -> Result<()> {
    crate::db::items::toggle_item_favorite(&pool, &id).await
}

/// 删除单条记录（薄封装）。若删的是图片记录，连带删除其落盘文件（原图 + 缩略图）。
/// 行已删成功，删文件失败仅记日志、不回滚——残留文件最坏占用磁盘，不影响功能。
#[tauri::command]
pub async fn delete_clipboard_item(
    pool: State<'_, SqlitePool>,
    store: State<'_, ImageStore>,
    id: String,
) -> Result<()> {
    if let Some(file_name) = crate::db::items::delete_item(&pool, &id).await? {
        if let Err(err) = store.remove(&file_name) {
            log::warn!("remove deleted image {file_name} failed: {err}");
        }
    }
    Ok(())
}

/// 更新备注（薄封装）。`note = None` 或空串清空备注；空串归一化为 None，
/// 保证「无备注」在库里只有一种表示（NULL），避免后续筛选/展示判别两套逻辑。
///
/// auto-favorite：写入非空备注时，若 `settings.clipboard.content.autoFavorite` 开启，
/// 顺带把 `is_favorite` 置为 true（已收藏的无变化；清空备注不触发）。返回值表示本次
/// 是否触发了 auto-favorite，供前端把乐观更新里的 `isFavorite` 一并设为 true。
#[tauri::command]
pub async fn update_clipboard_item_note(
    app: AppHandle,
    pool: State<'_, SqlitePool>,
    id: String,
    note: Option<String>,
) -> Result<bool> {
    let normalized = note.as_deref().and_then(|s| {
        let trimmed = s.trim();
        if trimmed.is_empty() {
            None
        } else {
            Some(trimmed)
        }
    });
    crate::db::items::update_item_note(&pool, &id, normalized).await?;

    let mut auto_favorited = false;
    if normalized.is_some() {
        let auto_favorite = app
            .try_state::<crate::settings::SettingsStore>()
            .map(|s| s.snapshot().clipboard.content.auto_favorite)
            .unwrap_or(false);
        if auto_favorite {
            crate::db::items::mark_item_favorite(&pool, &id).await?;
            auto_favorited = true;
        }
    }
    Ok(auto_favorited)
}

/// 校验图片文件名：必须是单层 `<name>.png`，不含路径分隔符 / 父目录引用。
fn validate_image_file_name(file_name: &str) -> Result<()> {
    let invalid = file_name.is_empty()
        || file_name.contains('/')
        || file_name.contains('\\')
        || file_name.contains("..")
        || !file_name.ends_with(".png");

    if invalid {
        return Err(AppError::Clipboard(format!(
            "invalid image file name: {file_name:?}"
        )));
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn accepts_plain_png_file_name() {
        assert!(validate_image_file_name("abcdef0123.png").is_ok());
    }

    #[test]
    fn rejects_traversal_and_subpaths() {
        for bad in [
            "",
            "evil.txt",
            "../secret.png",
            "..\\secret.png",
            "sub/dir.png",
            "a/b.png",
            "/abs.png",
            "name..png", // 含 ".." 序列，保守拒绝
        ] {
            assert!(
                validate_image_file_name(bad).is_err(),
                "should reject: {bad:?}"
            );
        }
    }
}

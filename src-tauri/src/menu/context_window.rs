//! Windows 自定义右键菜单窗（webview）。
//!
//! 用 `focusable: false` 的 webview 窗替代 muda：muda 的 `TrackPopupMenu` 必须把
//! 菜单 owner 拉到前台，会把用户原本聚焦的目标 App（资源管理器重命名编辑框、
//! 浏览器地址栏 IME 等）挤掉焦点。本窗口不抢焦，跟主窗口一样隐形挂在桌面上。
//!
//! - 启动期建窗一次（`init`），常驻隐藏；右键时 `set_size + set_position + emit + show`。
//! - 外部点击关闭：复用 [`crate::mouse`] 的全局鼠标钩子，菜单可见且光标在矩形外
//!   即 hide（不用轮询）。
//! - 菜单项点击：前端直接 emit `clipboard://menu-action` 给主窗，业务派发与
//!   macOS 路径走同一套。

use serde::Serialize;
use tauri::{
    AppHandle, Emitter, LogicalPosition, LogicalSize, Manager, WebviewUrl, WebviewWindowBuilder,
};

use crate::core::{AppError, Result};
use crate::settings::Language;

use super::clipboard_item::{ClipboardMenuAction, ACTION_GROUPS};

pub const CONTEXT_MENU_WINDOW_LABEL: &str = "context-menu";

/// 前端订阅事件：菜单窗收到后渲染并 `show`。
const CONTEXT_MENU_SHOW_EVENT: &str = "context-menu://show";

// 几何常量（logical px）：跟前端 `ContextMenu` 的 CSS 必须一致，否则 hit-test
// 与裁切会错位。前端那侧用同名 token 写在 `ContextMenu/index.tsx` 头部。
const MENU_WIDTH: u32 = 220;
const ITEM_HEIGHT: u32 = 32;
const SEPARATOR_HEIGHT: u32 = 9;
const MENU_PADDING_Y: u32 = 8; // 上下各 4

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
struct MenuItemPayload {
    action: ClipboardMenuAction,
    label: String,
    accelerator: Option<String>,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
struct ContextMenuShowPayload {
    item_id: String,
    is_favorite: bool,
    /// 已按 `ACTION_GROUPS` 过滤排序好的分组；前端按二维结构渲染 + 自动加分隔符。
    groups: Vec<Vec<MenuItemPayload>>,
}

/// setup 阶段建窗（仅 Windows）。失败只 log，不影响主流程：菜单首次右键时
/// `get_webview_window` 拿不到会返回错误，前端走 toast。
pub fn init(app: &AppHandle) {
    if let Err(err) = build_window(app) {
        log::error!("init context-menu window failed: {err:?}");
    }
}

fn build_window(app: &AppHandle) -> Result<()> {
    if app.get_webview_window(CONTEXT_MENU_WINDOW_LABEL).is_some() {
        return Ok(());
    }

    WebviewWindowBuilder::new(
        app,
        CONTEXT_MENU_WINDOW_LABEL,
        WebviewUrl::App("index.html/#/context-menu".into()),
    )
    .inner_size(MENU_WIDTH as f64, 240.0)
    .decorations(false)
    .transparent(true)
    .resizable(false)
    .maximizable(false)
    .minimizable(false)
    .always_on_top(true)
    .focusable(false)
    .visible(false)
    .skip_taskbar(true)
    .drag_and_drop(false)
    .build()
    .map_err(|err| AppError::Other(anyhow::anyhow!("build context-menu window: {err}")))?;

    Ok(())
}

/// 在当前光标处弹出列表项右键菜单。算好 size + position → `set_size`
/// → `set_position` → emit 数据 → `show`。
pub fn show_for_clipboard_item(
    app: &AppHandle,
    item_id: String,
    available_actions: &[ClipboardMenuAction],
    is_favorite: bool,
) -> Result<()> {
    let lang = crate::i18n::current_language(app);
    let groups = build_groups(available_actions, lang, is_favorite);
    if groups.is_empty() {
        return Ok(());
    }

    let window = app
        .get_webview_window(CONTEXT_MENU_WINDOW_LABEL)
        .ok_or_else(|| AppError::Other(anyhow::anyhow!("context-menu window missing")))?;

    let (width, height) = compute_size(&groups);
    let (x, y) = compute_position(&window, width, height)?;

    window
        .set_size(LogicalSize::new(width as f64, height as f64))
        .map_err(|err| AppError::Other(anyhow::anyhow!("context-menu set_size: {err}")))?;
    window
        .set_position(LogicalPosition::new(x as f64, y as f64))
        .map_err(|err| AppError::Other(anyhow::anyhow!("context-menu set_position: {err}")))?;

    let payload = ContextMenuShowPayload {
        item_id,
        is_favorite,
        groups,
    };
    window
        .emit(CONTEXT_MENU_SHOW_EVENT, payload)
        .map_err(|err| AppError::Other(anyhow::anyhow!("context-menu emit show: {err}")))?;
    window
        .show()
        .map_err(|err| AppError::Other(anyhow::anyhow!("context-menu show: {err}")))?;

    Ok(())
}

fn build_groups(
    available: &[ClipboardMenuAction],
    lang: Language,
    is_favorite: bool,
) -> Vec<Vec<MenuItemPayload>> {
    ACTION_GROUPS
        .iter()
        .map(|group| {
            group
                .iter()
                .filter(|a| available.contains(a))
                .map(|a| MenuItemPayload {
                    action: *a,
                    label: a.label(lang, is_favorite).into(),
                    accelerator: a.accelerator().map(String::from),
                })
                .collect::<Vec<_>>()
        })
        .filter(|g| !g.is_empty())
        .collect()
}

fn compute_size(groups: &[Vec<MenuItemPayload>]) -> (u32, u32) {
    let item_count: u32 = groups.iter().map(|g| g.len() as u32).sum();
    let separator_count = groups.len().saturating_sub(1) as u32;
    let h = MENU_PADDING_Y + item_count * ITEM_HEIGHT + separator_count * SEPARATOR_HEIGHT;
    (MENU_WIDTH, h)
}

/// 取光标所在显示器，把菜单矩形 clamp 在显示器内（鼠标处尽量为菜单左上角，
/// 右 / 下越界时翻到屏幕另一边）。返回 logical 坐标。
fn compute_position(window: &tauri::WebviewWindow, width: u32, height: u32) -> Result<(i32, i32)> {
    let cursor = window
        .cursor_position()
        .map_err(|err| AppError::Other(anyhow::anyhow!("cursor_position: {err}")))?;
    let scale = window
        .scale_factor()
        .map_err(|err| AppError::Other(anyhow::anyhow!("scale_factor: {err}")))?;
    let monitor = window
        .monitor_from_point(cursor.x, cursor.y)
        .map_err(|err| AppError::Other(anyhow::anyhow!("monitor_from_point: {err}")))?
        .ok_or_else(|| AppError::Other(anyhow::anyhow!("no monitor under cursor")))?;

    let cx = (cursor.x / scale) as i32;
    let cy = (cursor.y / scale) as i32;
    let mon_x = (monitor.position().x as f64 / scale) as i32;
    let mon_y = (monitor.position().y as f64 / scale) as i32;
    let mon_w = (monitor.size().width as f64 / scale) as i32;
    let mon_h = (monitor.size().height as f64 / scale) as i32;

    let max_x = mon_x + mon_w - width as i32;
    let max_y = mon_y + mon_h - height as i32;

    // 右 / 下越界时翻到光标左 / 上侧，避免菜单贴边裁切。
    let x = if cx > max_x { cx - width as i32 } else { cx }.clamp(mon_x, max_x.max(mon_x));
    let y = if cy > max_y { cy - height as i32 } else { cy }.clamp(mon_y, max_y.max(mon_y));

    Ok((x, y))
}

/// 隐藏菜单窗。给鼠标钩子 / 主窗隐藏等外部触发处用。
pub fn hide(app: &AppHandle) {
    let Some(window) = app.get_webview_window(CONTEXT_MENU_WINDOW_LABEL) else {
        return;
    };
    if let Err(err) = window.hide() {
        log::warn!("hide context-menu window failed: {err}");
    }
}

/// 菜单窗当前是否可见。给鼠标钩子判断「是否需要做外部点击关闭」。
pub fn is_visible(app: &AppHandle) -> bool {
    app.get_webview_window(CONTEXT_MENU_WINDOW_LABEL)
        .and_then(|w| w.is_visible().ok())
        .unwrap_or(false)
}

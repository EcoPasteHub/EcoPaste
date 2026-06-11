use tauri::AppHandle;

use crate::core::Result;
use crate::settings::WindowPosition;
use crate::window;

pub use window::lifecycle::LifecycleSnapshot;
pub use window::preview::{ClipboardPreviewState, PreviewAnchorRect};

#[tauri::command]
pub async fn show_window(app: AppHandle, label: String) -> Result<()> {
    window::show_window(&app, &label)
}

#[tauri::command]
pub async fn hide_window(app: AppHandle, label: String) -> Result<()> {
    window::hide_window(&app, &label)
}

#[tauri::command]
pub async fn toggle_window(app: AppHandle, label: String) -> Result<()> {
    window::toggle_window(&app, &label)
}

/// 前端 WebView 完成基础初始化后回报 ready，由生命周期管理器把窗口推进到 `Ready` 阶段。
/// label 由前端取自身窗口传入，避免硬编码。
#[tauri::command]
pub async fn notify_window_ready(app: AppHandle, label: String) {
    window::lifecycle::on_ready(&app, &label);
}

/// 标记当前窗口是否存在未保存草稿。相同 owner 可重复设置；所有 owner 清除后才允许销毁。
#[tauri::command]
pub async fn set_window_dirty(app: AppHandle, label: String, owner: String, dirty: bool) {
    if label.is_empty() || owner.is_empty() {
        return;
    }

    window::lifecycle::set_dirty(&app, &label, &owner, dirty);
}

/// 为当前窗口申请短期保活租约，防止原生对话框或后台动作进行中被 idle destroy。
#[tauri::command]
pub async fn acquire_window_keepalive(
    app: AppHandle,
    label: String,
    owner: String,
    reason: String,
    timeout_ms: Option<u64>,
) {
    if label.is_empty() || owner.is_empty() {
        return;
    }

    window::lifecycle::acquire_keepalive(&app, &label, &owner, &reason, timeout_ms);
}

/// 释放窗口保活租约。不存在时 no-op。
#[tauri::command]
pub async fn release_window_keepalive(app: AppHandle, label: String, owner: String) {
    if label.is_empty() || owner.is_empty() {
        return;
    }

    window::lifecycle::release_keepalive(&app, &label, &owner);
}

/// 返回生命周期调试快照，供偏好页诊断面板展示。
#[tauri::command]
pub async fn get_window_lifecycle_snapshot(app: AppHandle) -> Vec<LifecycleSnapshot> {
    window::lifecycle::snapshot(&app)
}

/// 打开偏好窗口并定位到指定设置项。偏好已空闲销毁时也能正确重建后跳转，
/// 替代前端 `show_window` 后直接 `emitTo`（重建异步会丢事件）。
#[tauri::command]
pub async fn open_preference_with_highlight(app: AppHandle, setting_id: String) -> Result<()> {
    window::open_preference_with_highlight(&app, setting_id)
}

/// 取走偏好窗口重建前暂存的高亮目标设置项；无暂存时返回 `null`。
#[tauri::command]
pub async fn take_pending_preference_highlight() -> Option<String> {
    window::take_pending_preference_highlight()
}

#[tauri::command]
pub async fn show_taskbar_icon(app: AppHandle, visible: bool) -> Result<()> {
    window::show_taskbar_icon(&app, visible)
}

#[tauri::command]
pub async fn position_window(
    app: AppHandle,
    label: String,
    position: WindowPosition,
) -> Result<()> {
    window::position_window(&app, &label, position)
}

#[tauri::command]
pub async fn set_main_window_pinned(pinned: bool) {
    window::set_main_window_pinned(pinned);
}

/// 临时暂停主窗口自动隐藏，避免系统文件选择等原生交互触发失焦收窗。
#[tauri::command]
pub async fn set_main_window_auto_hide_suspended(suspended: bool) {
    window::set_main_window_auto_hide_suspended(suspended);
}

#[tauri::command]
pub async fn show_clipboard_preview(
    app: AppHandle,
    item_id: String,
    anchor: PreviewAnchorRect,
) -> Result<Option<ClipboardPreviewState>> {
    window::preview::show_clipboard_preview(&app, item_id, anchor)
}

#[tauri::command]
pub async fn close_clipboard_preview(app: AppHandle) -> Result<()> {
    window::preview::close_clipboard_preview(&app)
}

#[tauri::command]
pub async fn get_clipboard_preview_state() -> Result<Option<ClipboardPreviewState>> {
    window::preview::get_clipboard_preview_state()
}

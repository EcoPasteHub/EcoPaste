use tauri::AppHandle;

use crate::core::Result;
use crate::settings::WindowPosition;
use crate::window;

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

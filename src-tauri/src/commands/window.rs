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

#[tauri::command]
pub async fn show_taskbar_icon(app: AppHandle, visible: bool) -> Result<()> {
    window::show_taskbar_icon(&app, visible)
}

/// 让主窗口进入文本输入模式，供 Windows `focusable=false` 主窗口里的搜索框使用。
#[tauri::command]
pub async fn focus_main_window_for_text_input(app: AppHandle) -> Result<()> {
    window::focus_main_window_for_text_input(&app)
}

/// 让主窗口退出文本输入模式，恢复 Windows 主窗口不抢焦点的导航模式。
#[tauri::command]
pub async fn restore_main_window_non_focusable(app: AppHandle) -> Result<()> {
    window::restore_main_window_non_focusable(&app)
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

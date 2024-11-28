use super::{shared_hide_window, shared_show_taskbar_icon, shared_show_window};
use tauri::{command, AppHandle, Runtime, WebviewWindow};

// 显示窗口
#[command]
pub async fn show_window<R: Runtime>(app_handle: AppHandle<R>, window: WebviewWindow<R>) {
    shared_show_window(&app_handle, &window);
}

// 隐藏窗口
#[command]
pub async fn hide_window<R: Runtime>(app_handle: AppHandle<R>, window: WebviewWindow<R>) {
    shared_hide_window(&app_handle, &window);
}

// 显示任务栏图标
#[command]
pub async fn show_taskbar_icon<R: Runtime>(
    app_handle: AppHandle<R>,
    window: WebviewWindow<R>,
    show: bool,
) {
    shared_show_taskbar_icon(&app_handle, &window, show);
}

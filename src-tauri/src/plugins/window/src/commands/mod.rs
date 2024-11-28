use tauri::{async_runtime::spawn, AppHandle, Manager, Runtime, WebviewWindow};

// 主窗口的label
pub static MAIN_WINDOW_LABEL: &str = "main";
// 偏好设置窗口的label
pub static PREFERENCE_WINDOW_LABEL: &str = "preference";
// 主窗口的title
pub static MAIN_WINDOW_TITLE: &str = "EcoPaste";

#[cfg(target_os = "macos")]
mod macos;

#[cfg(target_os = "windows")]
mod windows;

#[cfg(target_os = "linux")]
mod linux;

#[cfg(target_os = "macos")]
pub use macos::*;

#[cfg(target_os = "windows")]
pub use windows::*;

#[cfg(target_os = "linux")]
pub use linux::*;

// 是否为主窗口
pub fn is_main_window<R: Runtime>(window: &WebviewWindow<R>) -> bool {
    window.label() == MAIN_WINDOW_LABEL
}

// 共享显示窗口的方法
fn shared_show_window<R: Runtime>(app_handle: &AppHandle<R>, window: &WebviewWindow<R>) {
    let _ = window.show();
    let _ = window.unminimize();
    let _ = window.set_focus();

    let _ = app_handle;
}

// 共享隐藏窗口的方法
fn shared_hide_window<R: Runtime>(app_handle: &AppHandle<R>, window: &WebviewWindow<R>) {
    let _ = window.hide();

    let _ = app_handle;
}

// 共享显示任务栏图标的方法
#[cfg(not(target_os = "macos"))]
fn shared_show_taskbar_icon<R: Runtime>(
    app_handle: &AppHandle<R>,
    window: &WebviewWindow<R>,
    show: bool,
) {
    let _ = window.set_skip_taskbar(!show);

    let _ = app_handle;
}

// 显示主窗口
pub fn show_main_window(app_handle: &AppHandle) {
    show_window_by_label(app_handle, MAIN_WINDOW_LABEL);
}

// 显示偏好设置窗口
pub fn show_preference_window(app_handle: &AppHandle) {
    show_window_by_label(app_handle, PREFERENCE_WINDOW_LABEL);
}

// 显示指定 label 的窗口
fn show_window_by_label(app_handle: &AppHandle, label: &str) {
    if let Some(window) = app_handle.get_webview_window(label) {
        let app_handle_clone = app_handle.clone();

        spawn(async move {
            show_window(app_handle_clone, window).await;
        });
    }
}

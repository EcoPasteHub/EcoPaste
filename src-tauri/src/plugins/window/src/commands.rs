use tauri::{async_runtime::spawn, command, AppHandle, Manager, Runtime, WebviewWindow};

// 主窗口的label
pub static MAIN_WINDOW_LABEL: &str = "main";
// 偏好设置窗口的label
pub static PREFERENCE_WINDOW_LABEL: &str = "preference";
// 主窗口的title
pub static MAIN_WINDOW_TITLE: &str = "EcoPaste";

// 显示窗口（非linux）
#[cfg(not(target_os = "linux"))]
#[command]
pub async fn show_window<R: Runtime>(window: WebviewWindow<R>) {
    window.show().unwrap();
    window.unminimize().unwrap();
    window.set_focus().unwrap();
}

// 显示窗口（linux）
#[cfg(target_os = "linux")]
#[command]
pub async fn show_window<R: Runtime>(window: WebviewWindow<R>) {
    let position = window.outer_position().unwrap();
    let physical_position = tauri::PhysicalPosition::new(position.x, position.y);

    window.hide().unwrap();
    window.set_position(physical_position).unwrap();
    window.show().unwrap();
}

#[command]
// 隐藏窗口
pub async fn hide_window<R: Runtime>(window: WebviewWindow<R>) {
    window.hide().unwrap();
}

// 显示任务栏图标
#[command]
pub fn show_taskbar_icon<R: Runtime>(
    app_handle: AppHandle<R>,
    window: WebviewWindow<R>,
    show: bool,
) {
    #[cfg(not(target_os = "macos"))]
    window.set_skip_taskbar(!show).unwrap();

    #[cfg(target_os = "macos")]
    {
        use tauri::ActivationPolicy::*;

        let policy = if show { Regular } else { Accessory };
        let _ = app_handle.set_activation_policy(policy);
    }

    let _ = (app_handle, window);
}

// 显示主窗口
pub fn show_main_window(app_handle: &AppHandle) {
    let window = app_handle.get_webview_window(MAIN_WINDOW_LABEL).unwrap();

    spawn(async move {
        show_window(window).await;
    });
}

// 显示偏好设置窗口
pub fn show_preference_window(app_handle: &AppHandle) {
    let window = app_handle
        .get_webview_window(PREFERENCE_WINDOW_LABEL)
        .unwrap();

    spawn(async move {
        show_window(window).await;
    });
}

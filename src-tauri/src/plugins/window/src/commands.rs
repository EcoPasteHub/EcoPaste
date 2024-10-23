use tauri::{async_runtime::spawn, command, AppHandle, Manager, Runtime, WebviewWindow};
use tauri_nspanel::ManagerExt;

// 主窗口的label
pub static MAIN_WINDOW_LABEL: &str = "main";
// 偏好设置窗口的label
pub static PREFERENCE_WINDOW_LABEL: &str = "preference";
// 主窗口的title
pub static MAIN_WINDOW_TITLE: &str = "EcoPaste";

// 显示窗口
#[command]
pub async fn show_window<R: Runtime>(app_handle: AppHandle<R>, window: WebviewWindow<R>) {
    if cfg!(target_os = "macos") && window.label() == MAIN_WINDOW_LABEL {
        let handle = app_handle.clone();

        let _ = app_handle.run_on_main_thread(move || {
            let panel = handle.get_webview_panel(MAIN_WINDOW_LABEL).unwrap();

            panel.show();
        });
    } else {
        window.show().unwrap();
        window.unminimize().unwrap();
        window.set_focus().unwrap();
    }
}

// 隐藏窗口
#[command]
pub async fn hide_window<R: Runtime>(app_handle: AppHandle<R>, window: WebviewWindow<R>) {
    if cfg!(target_os = "macos") && window.label() == MAIN_WINDOW_LABEL {
        let handle = app_handle.clone();

        let _ = app_handle.run_on_main_thread(move || {
            let panel = handle.get_webview_panel(MAIN_WINDOW_LABEL).unwrap();

            panel.order_out(None);
        });
    } else {
        window.hide().unwrap();
    }
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

    // spawn(async move {
    //     show_window(window).await;
    // });
}

// 显示偏好设置窗口
pub fn show_preference_window(app_handle: &AppHandle) {
    let window = app_handle
        .get_webview_window(PREFERENCE_WINDOW_LABEL)
        .unwrap();

    // spawn(async move {
    //     show_window(window).await;
    // });
}

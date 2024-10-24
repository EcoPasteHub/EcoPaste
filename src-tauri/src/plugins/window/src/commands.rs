use tauri::{async_runtime::spawn, command, AppHandle, Manager, Runtime, WebviewWindow};
use tauri_nspanel::ManagerExt;

// 主窗口的label
pub static MAIN_WINDOW_LABEL: &str = "main";
// 偏好设置窗口的label
pub static PREFERENCE_WINDOW_LABEL: &str = "preference";
// 主窗口的title
pub static MAIN_WINDOW_TITLE: &str = "EcoPaste";

#[cfg(target_os = "macos")]
pub enum MacOSPanelStatus {
    Show,
    Hide,
    Resign,
}

// 显示窗口
#[command]
pub async fn show_window<R: Runtime>(app_handle: AppHandle<R>, window: WebviewWindow<R>) {
    #[cfg(target_os = "macos")]
    set_macos_panel(app_handle, &window, MacOSPanelStatus::Show);

    if !is_macos_panel(&window) {
        window.show().unwrap();
        window.unminimize().unwrap();
        window.set_focus().unwrap();
    }
}

// 隐藏窗口
#[command]
pub async fn hide_window<R: Runtime>(app_handle: AppHandle<R>, window: WebviewWindow<R>) {
    #[cfg(target_os = "macos")]
    set_macos_panel(app_handle, &window, MacOSPanelStatus::Hide);

    if !is_macos_panel(&window) {
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
    let app_handle_clone = app_handle.clone();

    let window = app_handle.get_webview_window(MAIN_WINDOW_LABEL).unwrap();

    spawn(async move {
        show_window(app_handle_clone, window).await;
    });
}

// 显示偏好设置窗口
pub fn show_preference_window(app_handle: &AppHandle) {
    let app_handle_clone = app_handle.clone();

    let window = app_handle
        .get_webview_window(PREFERENCE_WINDOW_LABEL)
        .unwrap();

    spawn(async move {
        show_window(app_handle_clone, window).await;
    });
}

// 是否是 macos 的 ns_panel 窗口
pub fn is_macos_panel<R: Runtime>(window: &WebviewWindow<R>) -> bool {
    return cfg!(target_os = "macos") && window.label() == MAIN_WINDOW_LABEL;
}

// 设置 macos 的 ns_panel 的状态
#[cfg(target_os = "macos")]
pub fn set_macos_panel<R: Runtime>(
    app_handle: AppHandle<R>,
    window: &WebviewWindow<R>,
    status: MacOSPanelStatus,
) {
    if is_macos_panel(window) {
        let handle = app_handle.clone();

        let _ = app_handle.run_on_main_thread(move || {
            let panel = handle.get_webview_panel(MAIN_WINDOW_LABEL).unwrap();

            match status {
                MacOSPanelStatus::Show => {
                    panel.show();
                }
                MacOSPanelStatus::Hide => {
                    panel.order_out(None);
                }
                MacOSPanelStatus::Resign => {
                    panel.resign_key_window();
                }
            }
        });
    }
}

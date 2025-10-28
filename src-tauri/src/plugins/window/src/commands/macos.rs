use super::is_main_window;
use crate::MAIN_WINDOW_LABEL;
use tauri::{command, AppHandle, Runtime, WebviewWindow};
use tauri_nspanel::{CollectionBehavior, ManagerExt};

pub enum NsPanelStatus {
    Show,
    Hide,
    Resign,
}

// 显示窗口
#[command]
pub async fn show_window<R: Runtime>(app_handle: AppHandle<R>, window: WebviewWindow<R>) {
    if is_main_window(&window) {
        set_ns_panel(&app_handle, &window, NsPanelStatus::Show);
    } else {
        let _ = window.show();
        let _ = window.unminimize();
        let _ = window.set_focus();
    }
}

// 隐藏窗口
#[command]
pub async fn hide_window<R: Runtime>(app_handle: AppHandle<R>, window: WebviewWindow<R>) {
    if is_main_window(&window) {
        set_ns_panel(&app_handle, &window, NsPanelStatus::Hide);
    } else {
        let _ = window.hide();
    }
}

// 显示任务栏图标
#[command]
pub async fn show_taskbar_icon<R: Runtime>(
    app_handle: AppHandle<R>,
    _window: WebviewWindow<R>,
    visible: bool,
) {
    let _ = app_handle.set_dock_visibility(visible);
}

// 设置 macos 的 ns_panel 的状态
pub fn set_ns_panel<R: Runtime>(
    app_handle: &AppHandle<R>,
    window: &WebviewWindow<R>,
    status: NsPanelStatus,
) {
    if is_main_window(window) {
        let app_handle_clone = app_handle.clone();

        let _ = app_handle.run_on_main_thread(move || {
            if let Ok(panel) = app_handle_clone.get_webview_panel(MAIN_WINDOW_LABEL) {
                match status {
                    NsPanelStatus::Show => {
                        panel.show_and_make_key();

                        panel.set_collection_behavior(
                            CollectionBehavior::new()
                                .stationary()
                                .can_join_all_spaces()
                                .full_screen_auxiliary()
                                .into(),
                        );
                    }
                    NsPanelStatus::Hide => {
                        panel.hide();

                        panel.set_collection_behavior(
                            CollectionBehavior::new()
                                .stationary()
                                .move_to_active_space()
                                .full_screen_auxiliary()
                                .into(),
                        );
                    }
                    NsPanelStatus::Resign => {
                        panel.resign_key_window();
                    }
                }
            }
        });
    }
}

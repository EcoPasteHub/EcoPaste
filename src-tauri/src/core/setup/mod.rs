use super::{app::observe_app, error::redirect_panic_to_log, info};
use crate::{plugins::window::show_preference_window, AUTO_LAUNCH_ARG};
use std::env;
use tauri::{App, Manager, Window};

#[cfg(target_os = "macos")]
mod mac;

#[cfg(target_os = "windows")]
mod win;

#[cfg(target_os = "linux")]
mod linux;

#[cfg(target_os = "macos")]
pub use mac::*;

#[cfg(target_os = "windows")]
pub use win::*;

#[cfg(target_os = "linux")]
pub use linux::*;

pub fn default(app: &mut App, main_window: Window, preference_window: Window) {
    let app_handle = app.app_handle();

    redirect_panic_to_log();

    // 获取命令行参数，检查是否有 `info` 或 `i` 参数
    match app.get_cli_matches() {
        Ok(matches) => {
            if matches.args["info"].value.as_bool().expect("参数错误") {
                info::print_app_info(app.app_handle());
                info::print_build_info();
                info::print_system_info();

                std::process::exit(0);
            }
        }
        Err(_) => {}
    }

    // 判断是否为自动启动
    let args: Vec<String> = env::args().collect();
    if !args.contains(&AUTO_LAUNCH_ARG.to_string()) {
        show_preference_window(&app_handle);
    }

    // 在开发环境中或者打包时加上 `--features=devtools` 启动时自动打开控制台：https://tauri.app/v1/guides/debugging/application/#opening-devtools-programmatically
    #[cfg(any(debug_assertions, feature = "devtools"))]
    main_window.open_devtools();

    // 给窗口添加阴影
    #[cfg(not(target_os = "linux"))]
    window_shadows::set_shadow(&main_window, true).unwrap();

    platform(app, main_window.clone(), preference_window.clone());

    observe_app();
}

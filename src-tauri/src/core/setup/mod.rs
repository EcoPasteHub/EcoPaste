use super::info;
use crate::AUTO_LAUNCH_ARG;
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
        preference_window.show().unwrap();
    }

    // 在开发环境中或者打包时加上 `--features=devtools` 启动时自动打开控制台：https://tauri.app/v1/guides/debugging/application/#opening-devtools-programmatically
    #[cfg(any(debug_assertions, feature = "devtools"))]
    main_window.open_devtools();
}

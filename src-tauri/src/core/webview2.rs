//! WebView2 运行时版本固定（仅 Windows）。
//!
//! wry 每次建窗都会新建一个 WebView2 environment，并让 loader 解析「当前最新」的 Evergreen
//! 运行时。进程长期常驻期间若系统把运行时升级（如 154 → 155），而本进程的浏览器进程仍在旧版本上
//! 持有同一个 user data 目录，之后重建窗口（偏好窗口空闲销毁后再打开）就会创建失败
//! （`0x80010108`）。Tauri 已经登记了该窗口，失败后留下无法再重建的僵尸窗口，托盘 / 快捷键
//! 都唤不起界面，只能重启应用。
//!
//! 进程启动时把 `WEBVIEW2_BROWSER_EXECUTABLE_FOLDER` 指向当时的运行时目录，使本进程生命周期内
//! 所有 environment 都使用同一版本；旧版本目录在仍被占用期间不会被 Edge 更新器删除。

use std::path::{Path, PathBuf};

use windows_registry::{CURRENT_USER, LOCAL_MACHINE};

const BROWSER_FOLDER_ENV: &str = "WEBVIEW2_BROWSER_EXECUTABLE_FOLDER";
const RUNTIME_EXE: &str = "msedgewebview2.exe";
const RUNTIME_CLIENT_KEYS: [&str; 2] = [
    r"SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}",
    r"SOFTWARE\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}",
];

/// 固定本进程使用的 WebView2 运行时目录，返回固定到的目录。
///
/// 用户或部署方已设置环境变量（固定版本运行时）时保持不变；找不到已安装的 Evergreen 运行时
/// 则不做任何事，交回 loader 默认行为。必须在创建任何 WebView 之前调用。
pub fn pin_runtime_version() -> Option<PathBuf> {
    if std::env::var_os(BROWSER_FOLDER_ENV).is_some_and(|value| !value.is_empty()) {
        return None;
    }

    let folder = installed_runtime_folder()?;
    std::env::set_var(BROWSER_FOLDER_ENV, &folder);
    Some(folder)
}

fn installed_runtime_folder() -> Option<PathBuf> {
    RUNTIME_CLIENT_KEYS.iter().find_map(|subkey| {
        runtime_folder_from_key(LOCAL_MACHINE, subkey)
            .or_else(|| runtime_folder_from_key(CURRENT_USER, subkey))
    })
}

fn runtime_folder_from_key(root: &windows_registry::Key, subkey: &str) -> Option<PathBuf> {
    let key = root.open(subkey).ok()?;
    let version = key.get_string("pv").ok()?;
    let location = key.get_string("location").ok()?;

    let folder = Path::new(&location).join(&version);
    folder.join(RUNTIME_EXE).is_file().then_some(folder)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn resolved_runtime_folder_contains_runtime_exe() {
        // 未安装 Evergreen 运行时的机器上解析为 None，不视为失败。
        if let Some(folder) = installed_runtime_folder() {
            assert!(folder.join(RUNTIME_EXE).is_file());
        }
    }
}

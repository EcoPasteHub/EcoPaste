use crate::i18n::keys::TrayKey as Key;

/// 返回美式英文系统托盘菜单文案。
pub fn label(key: Key) -> &'static str {
    match key {
        Key::Preference => "Preference",
        Key::StartListening => "Start Listening",
        Key::StopListening => "Stop Listening",
        Key::OpenSourceAddress => "Open Source Address",
        Key::CheckForUpdates => "Check for Updates",
        Key::Version => "Version",
        Key::Relaunch => "Relaunch",
        Key::Exit => "Exit",
    }
}

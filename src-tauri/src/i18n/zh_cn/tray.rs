use crate::i18n::keys::TrayKey as Key;

/// 返回简体中文系统托盘菜单文案。
pub fn label(key: Key) -> &'static str {
    match key {
        Key::Preference => "偏好设置",
        Key::StartListening => "开启监听",
        Key::StopListening => "停止监听",
        Key::OpenSourceAddress => "开源地址",
        Key::Version => "版本",
        Key::Relaunch => "重启应用",
        Key::Exit => "退出应用",
    }
}

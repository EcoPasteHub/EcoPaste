use crate::settings::Language;

#[derive(Debug, Clone, Copy)]
pub enum Key {
    Preference,
    StartListening,
    StopListening,
    OpenSourceAddress,
    Version,
    Relaunch,
    Exit,
}

pub fn label(lang: Language, key: Key) -> &'static str {
    match (lang, key) {
        (Language::ZhCN, Key::Preference) => "偏好设置",
        (Language::ZhCN, Key::StartListening) => "开启监听",
        (Language::ZhCN, Key::StopListening) => "停止监听",
        (Language::ZhCN, Key::OpenSourceAddress) => "开源地址",
        (Language::ZhCN, Key::Version) => "版本",
        (Language::ZhCN, Key::Relaunch) => "重启应用",
        (Language::ZhCN, Key::Exit) => "退出应用",
        (Language::EnUS, Key::Preference) => "Preference",
        (Language::EnUS, Key::StartListening) => "Start Listening",
        (Language::EnUS, Key::StopListening) => "Stop Listening",
        (Language::EnUS, Key::OpenSourceAddress) => "Open Source Address",
        (Language::EnUS, Key::Version) => "Version",
        (Language::EnUS, Key::Relaunch) => "Relaunch",
        (Language::EnUS, Key::Exit) => "Exit",
    }
}

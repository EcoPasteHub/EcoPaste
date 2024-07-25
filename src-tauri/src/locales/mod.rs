pub mod en_us;
pub mod zh_cn;
pub mod zh_tw;
pub mod ja_jp;

pub const ZH_CN: &str = "zh-CN";
pub const ZH_TW: &str = "zh-TW";
pub const EN_US: &str = "en-US";
pub const JA_JP: &str = "ja-JP";

pub const LANGUAGES: [&str; 4] = [ZH_CN, ZH_TW, EN_US, JA_JP];

pub struct Locale {
    pub preference: &'static str,
    pub language: &'static str,
    pub about: &'static str,
    pub update: &'static str,
    pub github: &'static str,
    pub version: &'static str,
    pub exit: &'static str,
}

pub fn get_locale(language: &str) -> &'static Locale {
    match language {
        ZH_CN => &zh_cn::LOCALE,
        ZH_TW => &zh_tw::LOCALE,
        EN_US => &en_us::LOCALE,
        JA_JP => &ja_jp::LOCALE,
        _ => &zh_cn::LOCALE,
    }
}

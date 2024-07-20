pub mod en_us;
pub mod zh_cn;

pub const ZH_CN: &str = "zh-CN";
pub const EN_US: &str = "en-US";

pub struct Locale {
    pub preference: &'static str,
    pub language: &'static str,
    pub zh_cn: &'static str,
    pub en_us: &'static str,
    pub about: &'static str,
    pub update: &'static str,
    pub github: &'static str,
    pub version: &'static str,
    pub exit: &'static str,
}

pub fn get_locale(language: &str) -> &'static Locale {
    match language {
        ZH_CN => &zh_cn::LOCALE,
        EN_US => &en_us::LOCALE,
        _ => &zh_cn::LOCALE,
    }
}

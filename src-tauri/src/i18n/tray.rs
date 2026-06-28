pub use crate::i18n::keys::TrayKey as Key;

use crate::settings::Language;

/// 返回系统托盘菜单文案。
pub fn label(lang: Language, key: Key) -> &'static str {
    match lang {
        Language::ZhCN => crate::i18n::zh_cn::tray::label(key),
        Language::EnUS => crate::i18n::en_us::tray::label(key),
    }
}

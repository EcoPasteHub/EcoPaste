pub use crate::i18n::keys::ClipboardMenuKey as Key;

use crate::settings::Language;

/// 返回剪贴板列表项右键菜单文案。
pub fn label(lang: Language, key: Key) -> &'static str {
    match lang {
        Language::ZhCN => crate::i18n::zh_cn::clipboard_menu::label(key),
        Language::EnUS => crate::i18n::en_us::clipboard_menu::label(key),
    }
}

pub use crate::i18n::keys::CommandKey as Key;

use crate::settings::Language;

/// 返回 Tauri 命令错误中会展示给用户的根因文案。
pub fn label(lang: Language, key: Key) -> &'static str {
    match lang {
        Language::ZhCN => crate::i18n::zh_cn::commands::label(key),
        Language::EnUS => crate::i18n::en_us::commands::label(key),
    }
}

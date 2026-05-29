//! 文本子类型识别与富文本转纯文本（纯逻辑，便于单测）。
//!
//! 对应旧项目前端的 `getClipboardTextSubtype` / `utils/is`，本次下沉到 Rust。
//! 判定顺序与旧项目一致：`url` > `email` > `color` > `path`。

use std::path::Path;
use std::sync::LazyLock;

use regex::Regex;

use crate::db::models::ClipboardSubKind;

/// URL：要求带协议头（http/https/ftp/file）或 `www.` 开头的单行串。
/// 比旧项目 `is-url` 收紧，避免把任意带点的词误判为链接。
static URL_RE: LazyLock<Regex> = LazyLock::new(|| {
    Regex::new(r"(?i)^(https?|ftp|file)://[^\s]+$|^www\.[^\s]+\.[^\s]+$").unwrap()
});

/// Email：沿用旧项目 `utils/is.ts` 的正则（允许中文用户名）。
static EMAIL_RE: LazyLock<Regex> = LazyLock::new(|| {
    Regex::new(r"^[A-Za-z0-9\u{4e00}-\u{9fa5}]+@[a-zA-Z0-9_-]+(\.[a-zA-Z0-9_-]+)+$").unwrap()
});

/// 颜色：hex(#RGB/#RGBA/#RRGGBB/#RRGGBBAA) 或 rgb()/rgba()/hsl()/hsla() 函数式。
/// 只匹配这三类语法，故 CSS 关键字（none/inherit…）与含 `url(...)` 的值天然不命中，
/// 无需旧项目的关键字排除表。
static COLOR_RE: LazyLock<Regex> = LazyLock::new(|| {
    Regex::new(
        r"(?i)^#([0-9a-f]{3}|[0-9a-f]{4}|[0-9a-f]{6}|[0-9a-f]{8})$|^(rgb|rgba|hsl|hsla)\([^)]*\)$",
    )
    .unwrap()
});

/// 识别纯文本的子类型。判定顺序：url > email > color > path。
/// 均不命中返回 `None`（普通文本）。
///
/// 注意：`path` 分支会触碰文件系统（`exists`），且仅认**绝对路径**——
/// 相对路径的存在性取决于进程 cwd（监听线程下不可控），收紧以避免误判。
pub fn detect_text_sub_kind(text: &str) -> Option<ClipboardSubKind> {
    let value = text.trim();
    if value.is_empty() {
        return None;
    }

    if URL_RE.is_match(value) {
        return Some(ClipboardSubKind::Url);
    }
    if EMAIL_RE.is_match(value) {
        return Some(ClipboardSubKind::Email);
    }
    if COLOR_RE.is_match(value) {
        return Some(ClipboardSubKind::Color);
    }
    if is_existing_absolute_path(value) {
        return Some(ClipboardSubKind::Path);
    }
    None
}

fn is_existing_absolute_path(value: &str) -> bool {
    let path = Path::new(value);
    path.is_absolute() && path.exists()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn detects_url() {
        for s in [
            "https://example.com",
            "http://a.b/c?d=1",
            "ftp://host/file",
            "www.example.com",
            "  https://trimmed.com  ",
        ] {
            assert_eq!(
                detect_text_sub_kind(s),
                Some(ClipboardSubKind::Url),
                "should be url: {s:?}"
            );
        }
        // 不带协议的裸域名/普通词不判 url。
        assert_eq!(detect_text_sub_kind("example.com"), None);
        assert_eq!(detect_text_sub_kind("hello world"), None);
    }

    #[test]
    fn detects_email() {
        assert_eq!(
            detect_text_sub_kind("user@example.com"),
            Some(ClipboardSubKind::Email)
        );
        assert_eq!(
            detect_text_sub_kind("张三@example.com.cn"),
            Some(ClipboardSubKind::Email)
        );
        assert_eq!(detect_text_sub_kind("not@an@email"), None);
    }

    #[test]
    fn detects_color() {
        for s in [
            "#fff",
            "#FFFF",
            "#ffffff",
            "#ffffffff",
            "rgb(1,2,3)",
            "rgba(1,2,3,0.5)",
            "hsl(0, 100%, 50%)",
            "hsla(0,100%,50%,.5)",
        ] {
            assert_eq!(
                detect_text_sub_kind(s),
                Some(ClipboardSubKind::Color),
                "should be color: {s:?}"
            );
        }
        // CSS 关键字 / 含 url 的值不判 color。
        assert_eq!(detect_text_sub_kind("inherit"), None);
        assert_eq!(detect_text_sub_kind("url(#abc)"), None);
        assert_eq!(detect_text_sub_kind("#xyz"), None);
    }

    #[test]
    fn detects_existing_absolute_path_only() {
        let dir = std::env::temp_dir();
        let file = dir.join(format!("ecopaste-detect-{}.txt", uuid::Uuid::new_v4()));
        std::fs::write(&file, b"x").unwrap();

        assert_eq!(
            detect_text_sub_kind(file.to_str().unwrap()),
            Some(ClipboardSubKind::Path)
        );
        // 不存在的绝对路径、相对路径都不判 path。
        assert_eq!(detect_text_sub_kind("/nope/does/not/exist/xyz"), None);
        assert_eq!(detect_text_sub_kind("src"), None);

        std::fs::remove_file(&file).ok();
    }
}

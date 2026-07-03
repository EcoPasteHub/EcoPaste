//! 文本子类型识别与富文本转纯文本（纯逻辑，便于单测）。
//!
//! 判定顺序：`url` > `email` > `color` > `path`。

use std::path::Path;
use std::sync::LazyLock;

use regex::Regex;

use crate::db::models::ClipboardSubKind;

/// URL：要求带协议头（http/https/ftp/file）或 `www.` 开头的单行串。
/// 规则保持收紧，避免把任意带点的词误判为链接。
static URL_RE: LazyLock<Regex> = LazyLock::new(|| {
    Regex::new(r"(?i)^(https?|ftp|file)://[^\s]+$|^www\.[^\s]+\.[^\s]+$")
        .expect("invalid URL regex")
});

/// Email：允许中文用户名。
static EMAIL_RE: LazyLock<Regex> = LazyLock::new(|| {
    Regex::new(r"^[A-Za-z0-9\u{4e00}-\u{9fa5}]+@[a-zA-Z0-9_-]+(\.[a-zA-Z0-9_-]+)+$")
        .expect("invalid email regex")
});

/// Hex 颜色：#RGB / #RGBA / #RRGGBB / #RRGGBBAA。
static HEX_COLOR_RE: LazyLock<Regex> = LazyLock::new(|| {
    Regex::new(r"(?i)^#([0-9a-f]{3}|[0-9a-f]{4}|[0-9a-f]{6}|[0-9a-f]{8})$")
        .expect("invalid hex color regex")
});

/// 颜色函数：覆盖经典 `rgb()/rgba()/hsl()/hsla()`（含逗号 / 空格语法）
/// 以及 CSS Color 4/5：`hwb() / lab() / lch() / oklab() / oklch() / color() / color-mix()`。
/// 括号内不限定（含嵌套函数），由 [`is_safe_css_value`] 做安全 + 括号平衡校验。
static COLOR_FN_RE: LazyLock<Regex> = LazyLock::new(|| {
    Regex::new(r"(?i)^(rgba?|hsla?|hwb|lab|lch|oklab|oklch|color|color-mix)\(.+\)$")
        .expect("invalid color fn regex")
});

/// CSS 渐变：`linear-gradient(...)` / `radial-gradient(...)` / `conic-gradient(...)`
/// 及其 `repeating-*` 变体。括号内不限定内容，由 [`is_safe_css_value`] 做安全/平衡校验。
static GRADIENT_RE: LazyLock<Regex> = LazyLock::new(|| {
    Regex::new(r"(?i)^(repeating-)?(linear|radial|conic)-gradient\(.+\)$")
        .expect("invalid gradient regex")
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
    if is_css_color_value(value) {
        return Some(ClipboardSubKind::Color);
    }
    if is_existing_absolute_path(value) {
        return Some(ClipboardSubKind::Path);
    }
    None
}

/// 把任意字符串规范化为可信的 CSS 颜色串：trim 后必须命中颜色 / 渐变规则
/// 且通过 [`is_safe_css_value`] 才返回 `Some`。命令层用它给前端 `colorPreview` 兜底，
/// 避免前端把任意文本塞进 CSS `background` 触发样式注入。
pub fn sanitize_css_color(text: &str) -> Option<String> {
    let value = text.trim();

    if value.is_empty() || !is_css_color_value(value) {
        return None;
    }

    Some(value.to_owned())
}

/// 统一判定：hex 直接命中即可（无括号无注入面）；
/// 函数式 / 渐变需要再过 [`is_safe_css_value`] 防注入与括号平衡。
fn is_css_color_value(value: &str) -> bool {
    if HEX_COLOR_RE.is_match(value) {
        return true;
    }

    if (COLOR_FN_RE.is_match(value) || GRADIENT_RE.is_match(value)) && is_safe_css_value(value) {
        return true;
    }

    false
}

/// CSS background 值安全校验：用于 gradient 这类括号内容自由的语法。
/// - 拒绝 `;` / `<` / `>` 等可能跳出声明 / 注入标签的字符；
/// - 拒绝 `url(` / `expression(` / `javascript:` / `@import` 等已知风险 token；
/// - 要求括号平衡，避免半截语法被浏览器宽松解析后吞掉后续 CSS。
fn is_safe_css_value(value: &str) -> bool {
    if value.contains(';') || value.contains('<') || value.contains('>') {
        return false;
    }

    let lower = value.to_ascii_lowercase();
    if lower.contains("url(")
        || lower.contains("expression(")
        || lower.contains("javascript:")
        || lower.contains("@import")
    {
        return false;
    }

    let mut depth: i32 = 0;
    for ch in value.chars() {
        match ch {
            '(' => depth += 1,
            ')' => {
                depth -= 1;
                if depth < 0 {
                    return false;
                }
            }
            _ => {}
        }
    }

    depth == 0
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
    fn detects_modern_color_functions() {
        for s in [
            // rgb() / hsl() 现代空格语法 + slash alpha
            "rgb(255 87 51)",
            "rgb(255 87 51 / 0.5)",
            "hsl(0 100% 50% / 80%)",
            // CSS Color 4 现代色彩空间
            "hwb(120 10% 20%)",
            "lab(50% 40 30)",
            "lch(50% 40 30)",
            "oklab(0.7 0.1 0.05)",
            "oklch(0.7 0.15 30)",
            "color(display-p3 1 0 0)",
            "color(rec2020 0.5 0.2 0.8 / 0.6)",
            // CSS Color 5 color-mix（含嵌套函数）
            "color-mix(in srgb, #fff 50%, #000)",
            "color-mix(in oklch, oklch(0.7 0.15 30), red 20%)",
        ] {
            assert_eq!(
                detect_text_sub_kind(s),
                Some(ClipboardSubKind::Color),
                "should be color (modern fn): {s:?}"
            );
            assert_eq!(sanitize_css_color(s).as_deref(), Some(s));
        }
    }

    #[test]
    fn detects_gradient_as_color() {
        for s in [
            "linear-gradient(to right, #ffdde1, #ee9ca7)",
            "radial-gradient(circle, rgba(0,0,0,0.5) 0%, #fff 100%)",
            "conic-gradient(from 45deg, red, blue)",
            "repeating-linear-gradient(45deg, #000 0 10px, #fff 10px 20px)",
        ] {
            assert_eq!(
                detect_text_sub_kind(s),
                Some(ClipboardSubKind::Color),
                "should be color (gradient): {s:?}"
            );
            assert_eq!(sanitize_css_color(s).as_deref(), Some(s));
        }

        // 含注入风险或括号不平衡的渐变串：识别命中但 sanitize 拒绝。
        for bad in [
            "linear-gradient(to right, url(http://x))",
            "linear-gradient(to right, #fff;color:red)",
            "linear-gradient(to right, #fff",
        ] {
            assert_eq!(sanitize_css_color(bad), None, "should reject: {bad:?}");
        }
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

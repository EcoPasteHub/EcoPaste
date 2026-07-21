/// Quotes one Windows command-line argument when it contains characters that
/// would otherwise split or terminate the argument.
pub fn quote_arg(value: impl AsRef<str>) -> String {
    let value = value.as_ref();
    if value.is_empty() {
        return "\"\"".to_owned();
    }

    if !value
        .bytes()
        .any(|byte| matches!(byte, b' ' | b'\t' | b'\n' | b'\r' | b'"'))
    {
        return value.to_owned();
    }

    let mut result = String::from("\"");
    let mut backslashes = 0;
    for ch in value.chars() {
        if ch == '\\' {
            backslashes += 1;
            continue;
        }

        if ch == '"' {
            result.push_str(&"\\".repeat(backslashes * 2 + 1));
            result.push('"');
            backslashes = 0;
            continue;
        }

        result.push_str(&"\\".repeat(backslashes));
        backslashes = 0;
        result.push(ch);
    }

    result.push_str(&"\\".repeat(backslashes * 2));
    result.push('"');
    result
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn quote_arg_keeps_simple_values_plain() {
        assert_eq!(quote_arg("--auto-launch"), "--auto-launch");
    }

    #[test]
    fn quote_arg_wraps_spaces() {
        assert_eq!(
            quote_arg(r"C:\Program Files\EcoPaste\EcoPaste.exe"),
            r#""C:\Program Files\EcoPaste\EcoPaste.exe""#
        );
    }

    #[test]
    fn quote_arg_escapes_quotes() {
        assert_eq!(quote_arg(r#"value"tail"#), r#""value\"tail""#);
    }

    #[test]
    fn quote_arg_escapes_trailing_backslash() {
        assert_eq!(quote_arg(r"C:\Users\A Yan\"), r#""C:\Users\A Yan\\""#);
    }
}

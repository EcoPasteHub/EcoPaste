use crate::i18n::keys::CommandKey as Key;

/// 返回美式英文 Tauri 命令错误根因文案。
pub fn label(key: Key) -> &'static str {
    match key {
        Key::DragSourceFilesMissing => "The dragged source files no longer exist",
        Key::DragImageMissing => "The image file no longer exists",
        Key::DragTextEmpty => "Text content is empty",
        Key::ExternalUrlUnsupported => "Only links starting with http or https can be opened",
        Key::PasteAccessibilityDenied => {
            "Accessibility permission is not granted; the content has been copied to the clipboard, please paste manually"
        }
        Key::PasteTargetUnavailable => {
            if cfg!(target_os = "windows") {
                "Windows could not focus the destination or send the paste keys. The content is copied; select the destination and paste manually. If the destination runs as administrator, check the apps' permission levels"
            } else {
                "The destination app could not receive keyboard focus. The content is copied; select the destination and paste manually, or reopen EcoPaste from that app"
            }
        }
    }
}

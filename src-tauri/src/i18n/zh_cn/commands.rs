use crate::i18n::keys::CommandKey as Key;

/// 返回简体中文 Tauri 命令错误根因文案。
pub fn label(key: Key) -> &'static str {
    match key {
        Key::DragSourceFilesMissing => "拖拽源文件已不存在",
        Key::DragImageMissing => "图片文件已不存在",
        Key::DragTextEmpty => "文本内容为空",
        Key::ExternalUrlUnsupported => "只能打开 http 或 https 开头的链接",
        Key::PasteAccessibilityDenied => "未获得辅助功能权限，内容已复制到剪贴板，请手动粘贴",
        Key::PasteTargetUnavailable => {
            if cfg!(target_os = "windows") {
                "Windows 未能切换到目标窗口或发送粘贴按键。内容已复制，请选中目标窗口手动粘贴；若目标以管理员身份运行，请检查两端权限级别"
            } else {
                "未能将键盘焦点交给目标应用。内容已复制，请选中目标应用手动粘贴，或从目标应用重新打开 EcoPaste"
            }
        }
    }
}

use crate::i18n::keys::CommandKey as Key;

/// 返回简体中文 Tauri 命令错误根因文案。
pub fn label(key: Key) -> &'static str {
    match key {
        Key::DragSourceFilesMissing => "拖拽源文件已不存在",
        Key::DragImageMissing => "图片文件已不存在",
        Key::DragTextEmpty => "文本内容为空",
        Key::ExternalUrlUnsupported => "只能打开 http 或 https 开头的链接",
        Key::SyncNotRunning => "请先开启跨设备同步",
        Key::SyncNoDevice => "没有发现正在显示匹配码的设备，可填写对方 IP 后重试",
        Key::SyncCodeInvalid => "匹配码错误或已过期",
        Key::SyncAddressInvalid => "IP 地址格式不正确",
        Key::CloudUnreachable => "无法连接云端服务器，请检查地址、端口映射和网络",
        Key::CloudPinMismatch => "服务器证书与已确认的指纹不一致，连接已拒绝",
        Key::CloudBadCredentials => "用户名或密码错误",
        Key::CloudLocked => "失败次数过多，已被暂时锁定，请稍后再试",
        Key::CloudNotSignedIn => "登录已失效，请重新登录",
        Key::CloudBadAddress => "服务器地址格式不正确",
        Key::SyncUnreachable => "无法连接该 IP，请确认两台设备在同一网络且对方已开启同步",
    }
}

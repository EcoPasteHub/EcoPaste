//! 一次剪贴板读取的「带类型标记」结果。
//!
//! 这是阶段 2.1 的产物：仅做原始读取与种类归类（text / image / files），
//! 映射到 [`crate::db::ClipboardKind`]。子类型识别（url / email / color / path）、
//! 图片落盘与缩略图属于阶段 2.3，不在此处。

/// 剪贴板内容按种类归类后的载荷。读取优先级：files > image > text。
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ClipboardPayload {
    Text(TextPayload),
    Image(ImagePayload),
    /// 文件路径列表（裸路径，已由底层库去掉 `file://` 前缀并解码）。
    Files(Vec<String>),
}

/// 文本载荷。同一次复制可能同时存在多种富文本表示，这里全部保留，
/// 由下游（2.3）决定 `content` 与 `search_text` 的取舍。
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct TextPayload {
    /// 纯文本表示，始终尝试读取；富文本场景下可作为降级 / 检索文本。
    pub text: String,
    /// HTML 表示，仅当剪贴板提供且非空时存在。
    pub html: Option<String>,
    /// RTF 表示，仅当剪贴板提供且非空时存在。
    pub rtf: Option<String>,
}

/// 图片载荷。底层 macOS 剪贴板原始为 TIFF，库已统一解码后重新编码为 PNG。
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ImagePayload {
    /// PNG 编码字节。用于去重哈希与 2.3 的落盘，不直接回传前端。
    pub bytes: Vec<u8>,
    pub width: u32,
    pub height: u32,
}

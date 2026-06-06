//! 设置数据模型。
//!
//! 每个字段都 `#[serde(default)]`，缺字段时回落到 `Default`，这样新增字段不破坏旧配置文件。
//! 不为旧版本数据做迁移——本项目是重写，没有兼容包袱。

use serde::{Deserialize, Serialize};

use crate::db::models::ClipboardItemSort;

#[derive(Debug, Clone, Default, Serialize, Deserialize, PartialEq, Eq)]
#[serde(default, rename_all = "camelCase")]
pub struct Settings {
    pub general: General,
    pub appearance: Appearance,
    pub shortcuts: Shortcuts,
    pub clipboard: Clipboard,
    pub update: Update,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(default, rename_all = "camelCase")]
pub struct General {
    pub auto_start: bool,
    /// 启动后不显示主窗（配合 `auto_start` 用，开机静默驻留托盘）。
    pub silent_start: bool,
    /// macOS 菜单栏 / Windows 系统托盘图标。
    pub tray_icon: bool,
    /// macOS Dock / Windows 任务栏图标。
    pub dock_icon: bool,
}

impl Default for General {
    fn default() -> Self {
        Self {
            auto_start: false,
            silent_start: false,
            tray_icon: true,
            dock_icon: false,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(default, rename_all = "camelCase")]
pub struct Appearance {
    pub theme: Theme,
    pub language: Language,
}

impl Default for Appearance {
    fn default() -> Self {
        Self {
            theme: Theme::Auto,
            language: Language::ZhCN,
        }
    }
}

#[derive(Debug, Clone, Copy, Default, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "lowercase")]
pub enum Theme {
    #[default]
    Auto,
    Light,
    Dark,
}

#[derive(Debug, Clone, Copy, Default, Serialize, Deserialize, PartialEq, Eq)]
pub enum Language {
    #[default]
    #[serde(rename = "zh-CN")]
    ZhCN,
    #[serde(rename = "en-US")]
    EnUS,
}

impl Language {
    /// 把系统 locale（如 `zh_CN.UTF-8` / `en-US` / `ja-JP`）映射到支持的语言；
    /// 任何 zh-* 都归到 zh-CN，其余一律 en-US。
    pub fn from_system_locale(tag: &str) -> Self {
        let lower = tag.to_ascii_lowercase();
        if lower.starts_with("zh") {
            Self::ZhCN
        } else {
            Self::EnUS
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(default, rename_all = "camelCase")]
pub struct Shortcuts {
    /// 全局：唤起剪贴板主窗。
    pub open_clipboard: String,
    /// 全局：打开偏好窗。
    pub open_preference: String,
    /// 主窗内局部：粘贴时强制走纯文本（不在 OS 级注册）。
    pub paste_plain: String,
    pub quick_paste: QuickPaste,
}

impl Default for Shortcuts {
    fn default() -> Self {
        Self {
            open_clipboard: "Alt+C".into(),
            open_preference: "Alt+X".into(),
            paste_plain: String::new(),
            quick_paste: QuickPaste::default(),
        }
    }
}

/// 「按住修饰键 + 数字键」粘贴第 N 条历史，配合主窗显示序号用。
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(default, rename_all = "camelCase")]
pub struct QuickPaste {
    pub enabled: bool,
    pub modifier: String,
}

impl Default for QuickPaste {
    fn default() -> Self {
        Self {
            enabled: false,
            modifier: "Cmd+Shift".into(),
        }
    }
}

#[derive(Debug, Clone, Default, Serialize, Deserialize, PartialEq, Eq)]
#[serde(default, rename_all = "camelCase")]
pub struct Clipboard {
    pub capture: Capture,
    pub content: Content,
    pub display: Display,
    pub sensitive: Sensitive,
    pub history: History,
    pub search: Search,
    pub window: Window,
    pub preview: Preview,
    pub feedback: Feedback,
    pub filters: Filters,
}

/// 剪贴板内容类型采集开关。关闭后监听与手动读取都不入库对应类型。
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(default, rename_all = "camelCase")]
pub struct Capture {
    pub text: bool,
    pub html: bool,
    pub rtf: bool,
    pub image: bool,
    pub files: bool,
}

impl Default for Capture {
    fn default() -> Self {
        Self {
            text: true,
            html: true,
            rtf: true,
            image: true,
            files: true,
        }
    }
}

/// 隐私保护开关。命中规则的内容在入库前直接跳过。
#[derive(Debug, Clone, Copy, Default, Serialize, Deserialize, PartialEq, Eq)]
#[serde(default, rename_all = "camelCase")]
pub struct Sensitive {
    pub secret_detection: bool,
}

/// 应用过滤规则。
/// `excluded_app_ids` 命中复制来源时，对应剪贴板内容不入库。
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(default, rename_all = "camelCase")]
pub struct Filters {
    pub excluded_app_ids: Vec<String>,
}

impl Default for Filters {
    fn default() -> Self {
        Self {
            excluded_app_ids: default_excluded_app_ids(),
        }
    }
}

fn default_excluded_app_ids() -> Vec<String> {
    #[cfg(target_os = "macos")]
    {
        // 系统级密码 / 密钥工具：用户从这里复制的几乎都是敏感凭据，默认不入库。
        // - com.apple.keychainaccess：钥匙串访问
        // - com.apple.Passwords：macOS 15 起的「密码」App
        vec![
            "com.apple.keychainaccess".to_owned(),
            "com.apple.Passwords".to_owned(),
        ]
    }
    #[cfg(target_os = "windows")]
    {
        // Windows 无系统内置的密码管理 App（凭据管理器是 Control Panel 子项，不会作为复制来源）。
        // 第三方密码管理器（1Password / Bitwarden / KeePass 等）因人而异，留给用户在 UI 勾选。
        Vec::new()
    }
    #[cfg(not(any(target_os = "macos", target_os = "windows")))]
    {
        Vec::new()
    }
}

#[cfg(all(test, target_os = "macos"))]
mod tests {
    use super::*;

    #[test]
    fn macos_defaults_keep_sensitive_system_apps_excluded() {
        let ids = default_excluded_app_ids();

        assert!(ids.contains(&"com.apple.keychainaccess".to_owned()));
        assert!(ids.contains(&"com.apple.Passwords".to_owned()));
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(default, rename_all = "camelCase")]
pub struct Content {
    /// 点击列表项时的自动粘贴行为。
    pub auto_paste: AutoPaste,
    /// 中键点击列表项时执行的动作。
    pub middle_click: MiddleClickAction,
    /// 复制（写回剪贴板）时去除格式。
    pub copy_plain: bool,
    /// 粘贴时去除格式（与全局快捷键 `paste_plain` 等价的默认行为）。
    pub paste_plain: bool,
    /// 粘贴文件记录时，默认写入路径文本而不是文件本身。
    pub paste_files_as_path: bool,
    /// 鼠标悬停时显示原始内容预览（HTML/RTF 渲染前的原文）。
    pub show_original_preview: bool,
    pub delete_confirm: bool,
    pub auto_favorite: bool,
    /// 从历史中复制 / 粘贴时，是否刷新使用次数与 `updated_at`。
    pub update_on_reuse: bool,
    /// 历史列表默认排序，和 `ClipboardItemQuery.sort` 使用同一套契约字面量。
    pub sort: ClipboardItemSort,
    /// 列表项悬停操作按钮 (顺序即显示顺序)。
    pub item_actions: Vec<ItemAction>,
}

impl Default for Content {
    fn default() -> Self {
        Self {
            auto_paste: AutoPaste::DoubleClickPaste,
            middle_click: MiddleClickAction::Disabled,
            copy_plain: false,
            paste_plain: false,
            paste_files_as_path: false,
            show_original_preview: false,
            delete_confirm: true,
            auto_favorite: false,
            update_on_reuse: true,
            sort: ClipboardItemSort::CreatedAt,
            item_actions: vec![ItemAction::Copy, ItemAction::Star, ItemAction::Delete],
        }
    }
}

/// 历史列表里不同内容类型的展示上限。
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(default, rename_all = "camelCase")]
pub struct Display {
    /// 文本摘要最多显示行数。
    pub text_max_lines: u8,
    /// 图片缩略图显示高度，单位 px。
    pub image_max_height: u16,
    /// 文件列表最多返回并显示的条目数。
    pub file_max_count: u8,
}

impl Default for Display {
    fn default() -> Self {
        Self {
            text_max_lines: 3,
            image_max_height: 64,
            file_max_count: 3,
        }
    }
}

impl Display {
    /// 返回主列表文件条目上限，并夹在 UI 支持的范围内控制 IPC 与 icon 抽取成本。
    pub fn file_entry_limit(self) -> usize {
        usize::from(self.file_max_count.clamp(1, 5))
    }
}

#[derive(Debug, Clone, Copy, Default, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "camelCase")]
pub enum AutoPaste {
    /// 点击只选中，不自动执行动作。
    Disabled,
    SingleClickPaste,
    #[default]
    DoubleClickPaste,
    SingleClickCopy,
    DoubleClickCopy,
}

#[derive(Debug, Clone, Copy, Default, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "camelCase")]
pub enum MiddleClickAction {
    /// 中键点击仅选中，不自动执行动作。
    #[default]
    Disabled,
    SingleClickPaste,
    SingleClickPastePlain,
    SingleClickCopy,
    SingleClickCopyPlain,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "camelCase")]
pub enum ItemAction {
    Copy,
    PastePlain,
    Note,
    Star,
    Delete,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(default, rename_all = "camelCase")]
pub struct Preview {
    pub hover_enabled: bool,
    pub hover_delay_ms: PreviewHoverDelayMs,
    pub space_enabled: bool,
}

impl Default for Preview {
    fn default() -> Self {
        Self {
            hover_enabled: false,
            hover_delay_ms: PreviewHoverDelayMs::Ms500,
            space_enabled: true,
        }
    }
}

#[derive(Debug, Clone, Copy, Default, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "camelCase")]
pub enum PreviewHoverDelayMs {
    Ms300,
    #[default]
    Ms500,
    Ms1000,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq, Default)]
#[serde(default, rename_all = "camelCase")]
pub struct History {
    pub retention: Retention,
    /// 最多保留条数。`0` = 不限。
    pub max_count: u32,
}

/// 历史保留时长。`unit = Forever` 时忽略 `value`。
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(default, rename_all = "camelCase")]
pub struct Retention {
    pub value: u32,
    pub unit: RetentionUnit,
}

impl Default for Retention {
    fn default() -> Self {
        Self {
            value: 0,
            unit: RetentionUnit::Forever,
        }
    }
}

#[derive(Debug, Clone, Copy, Default, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "camelCase")]
pub enum RetentionUnit {
    Hours,
    Days,
    Weeks,
    Months,
    #[default]
    Forever,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(default, rename_all = "camelCase")]
pub struct Search {
    /// 主窗每次显示时自动聚焦搜索框。
    pub default_focus: bool,
    /// 主窗隐藏时清空搜索关键词。
    pub clear_on_hide: bool,
}

impl Default for Search {
    fn default() -> Self {
        Self {
            default_focus: false,
            clear_on_hide: true,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(default, rename_all = "camelCase")]
pub struct Window {
    pub position: WindowPosition,
    pub always_on_top: bool,
    /// 在所有桌面/工作区可见（macOS Spaces / Windows 虚拟桌面）。
    pub all_workspaces: bool,
}

impl Default for Window {
    fn default() -> Self {
        Self {
            position: WindowPosition::FollowCursor,
            always_on_top: true,
            all_workspaces: true,
        }
    }
}

#[derive(Debug, Clone, Copy, Default, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "camelCase")]
pub enum WindowPosition {
    Remember,
    #[default]
    FollowCursor,
    Center,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize, PartialEq, Eq)]
#[serde(default, rename_all = "camelCase")]
pub struct Feedback {
    pub copy_sound: bool,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize, PartialEq, Eq)]
#[serde(default, rename_all = "camelCase")]
pub struct Update {
    pub auto_check: bool,
    pub frequency: UpdateFrequency,
    pub include_beta: bool,
}

#[derive(Debug, Clone, Copy, Default, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "camelCase")]
pub enum UpdateFrequency {
    Daily,
    #[default]
    Weekly,
    Monthly,
}

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::FromRow;

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, sqlx::Type)]
#[sqlx(rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum ClipboardKind {
    Text,
    Image,
    Files,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, sqlx::Type)]
#[sqlx(rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum ClipboardSubKind {
    Rtf,
    Html,
    Url,
    Email,
    Color,
    Path,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, sqlx::Type)]
#[sqlx(rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum Platform {
    Macos,
    Windows,
}

#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
#[serde(rename_all = "camelCase")]
pub struct ClipboardItem {
    pub id: String,
    pub kind: ClipboardKind,
    pub sub_kind: Option<ClipboardSubKind>,
    pub group_id: Option<String>,
    /// 复制时的来源应用 id（macOS = bundle id，Windows = 可执行文件绝对路径）；
    /// 监听 / 命令链路若未能取到前台应用则为 `None`。引用 `clipboard_apps(id)`，
    /// 删除应用记录时置 NULL，不会级联删条目。
    pub source_app_id: Option<String>,
    pub content: String,
    /// 去重指纹：`blake3(kind:content)`，由 `db::items::content_hash` 计算并在入库前比对。
    pub content_hash: String,
    pub search_text: Option<String>,
    /// 列表渲染用的纯文本摘要（最多 512 字符）。HTML/RTF 也只存纯文本截断
    /// （来源是 OS 同时提供的纯文本，不解析富文本）；Image/Files 为 `None`。
    /// 完整内容仍在 `content`，预览/写回时再读。
    pub summary: Option<String>,
    /// Files 类型专用：紧凑格式记录每个路径的类型，如 "d,f,f" 表示 [dir, file, file]。
    /// d=directory, f=file。用于删除文件后仍能准确显示 icon。
    pub file_types: Option<String>,
    pub size: Option<i64>,
    pub width: Option<i64>,
    pub height: Option<i64>,
    pub use_count: i64,
    pub is_favorite: bool,
    pub is_pinned: bool,
    /// 命中敏感内容规则且被收录的条目；展示是否脱敏由当前设置决定。
    pub is_sensitive: bool,
    pub platform: Platform,
    pub note: Option<String>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,

    /// 来源应用名称。仅 list 查询通过 LEFT JOIN `clipboard_apps` 填充；
    /// 单条 `SELECT_ITEM` 路径与 `INSERT` 不读不写，`#[sqlx(default)]` 保证缺列时为 `None`。
    #[sqlx(default)]
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub source_app_name: Option<String>,
    /// 来源应用图标文件名（`<hash>.png`）。同 [`source_app_name`] 由 list 查询补齐，
    /// 命令层据此解析为绝对路径写入 [`source_app_icon_path`]。
    #[sqlx(default)]
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub source_app_icon_file: Option<String>,
    /// 命令层用 `AppIconStore` 把 `icon_file` 解析后的磁盘绝对路径；
    /// 前端可直接 `convertFileSrc` 渲染。来源不在 SQL，纯后置填充。
    #[sqlx(default)]
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub source_app_icon_path: Option<String>,
    /// Image 类型条目的缩略图绝对路径；命令层按需确保缩略图存在后回填，
    /// 前端可直接 `convertFileSrc` 渲染，避免逐条再发取图命令。
    #[sqlx(default)]
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub image_thumbnail_path: Option<String>,
    /// Files 类型条目对应的预处理后的文件条目，数量由命令层按设置截断。
    /// 前端 `FilesCard` 直接 map 渲染，无需再解析 `content` / `file_types`。
    #[sqlx(skip)]
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub file_entries: Option<Vec<FileEntry>>,
    /// Files 卡片渲染模式：单文件且为图片走 `ImagePreview`，其它走 `List`。
    /// 命令层在填充 `file_entries` 时一并计算，前端无需再判定。
    #[sqlx(skip)]
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub files_preview_kind: Option<FilesPreviewKind>,
    /// 当前条目右键菜单可用的动作列表（按 kind / sub_kind 计算），按建议展示顺序排列。
    /// 前端只负责把动作映射成菜单项 + 文案 + 快捷键，不再判定「能否打开链接」等业务条件。
    #[sqlx(skip)]
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub available_actions: Vec<ClipboardAction>,
    /// `sub_kind = Color` 时的规范化 CSS 颜色串（命令层用 [`crate::clipboard::sanitize_css_color`] 校验后填充）。
    /// 前端可直接塞 `style.background`，无需自行判断 `summary` 是否合法。
    #[sqlx(skip)]
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub color_preview: Option<String>,
    /// 按本地时区对 `created_at` 做三档格式化，命令层填充供前端直接渲染：
    /// 今天 → `HH:mm`，今年内 → `MM-DD HH:mm`，跨年 → `YYYY-MM-DD HH:mm`。
    /// 前端不再需要引入 dayjs。
    #[sqlx(skip)]
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub display_created_at: String,
}

/// 右键菜单可执行的动作种类。
/// 顺序约定见 [`crate::commands::clipboard::compute_available_actions`]。
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum ClipboardAction {
    /// 普通粘贴（恒在）。
    Paste,
    /// 文本条目额外提供「粘贴为纯文本」。
    PasteAsPlainText,
    /// 文件条目额外提供「粘贴为路径」（写回纯文本路径列表）。
    PasteAsPath,
    /// 复制回剪贴板（恒在）。
    Copy,
    /// 将图片条目另存到本地文件（`kind = image`）。
    SaveImage,
    /// 在浏览器打开链接（`sub_kind = url`）。
    OpenLink,
    /// 调起邮件客户端（`sub_kind = email`）。
    SendEmail,
    /// 在 Finder 中显示（macOS，`sub_kind = path` 或 `kind = files`）。
    RevealInFinder,
    /// 在资源管理器中显示（Windows，`sub_kind = path` 或 `kind = files`）。
    RevealInExplorer,
    /// 切换收藏（恒在；前端按 `is_favorite` 切「收藏 / 取消收藏」文案）。
    ToggleFavorite,
    /// 切换置顶（恒在；前端按 `is_pinned` 切「置顶 / 取消置顶」文案）。
    TogglePinned,
    /// 编辑备注（恒在）。
    EditNote,
    /// 删除条目（恒在）。
    Delete,
}

/// Files 类型条目里的单条文件/目录元信息，由命令层组装后返回前端。
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct FileEntry {
    pub path: String,
    pub name: String,
    pub is_dir: bool,
    pub is_image: bool,
    /// 命令层组装时实时检测：路径是否仍存在于磁盘，前端据此对失效条目划删除线并回退预览。
    pub exists: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub icon_path: Option<String>,
}

/// Files 卡片的渲染模式（命令层计算后塞给前端）。
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum FilesPreviewKind {
    /// 单文件且为图片，直接渲染图片预览。
    ImagePreview,
    /// 多文件或非图片，渲染图标 + 文件名列表。
    List,
}

/// 剪贴板来源应用（macOS bundle id / Windows exe 路径作主键），
/// 名称与图标按 id 去重共享，单条剪贴板记录通过 `source_app_id` 引用。
#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
#[serde(rename_all = "camelCase")]
pub struct ClipboardApp {
    pub id: String,
    pub name: String,
    /// `app-icons/<hash>.png` 形式的文件名（无分片目录前缀）；无图标则 `None`。
    pub icon_file: Option<String>,
    pub platform: Platform,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
#[serde(rename_all = "camelCase")]
pub struct ClipboardGroup {
    pub id: String,
    pub name: String,
    pub icon: String,
    pub is_hidden: bool,
    pub sort_order: i64,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, Default)]
#[serde(rename_all = "camelCase")]
pub enum ClipboardItemSort {
    #[serde(rename = "createdAtDesc")]
    CreatedAt,
    #[default]
    #[serde(rename = "updatedAtDesc")]
    UpdatedAt,
    #[serde(rename = "useCountDesc")]
    UseCount,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", default)]
pub struct ClipboardItemQuery {
    pub kind: Option<ClipboardKind>,
    pub group_id: Option<String>,
    pub favorite: Option<bool>,
    pub pinned: Option<bool>,
    /// 列表顶部 Tab 过滤（前端只需传这一个；Rust 侧翻译成 kind / favorite）。
    /// 显式设置时覆盖 `kind` / `favorite`；为 None 时走显式字段（保留给单测）。
    pub group: Option<ClipboardGroupFilter>,
    pub keyword: Option<String>,
    pub sort: ClipboardItemSort,
    pub limit: i64,
    pub offset: i64,
}

/// 列表顶部分组 Tab：UI 概念，与 `ClipboardGroup`（用户自建分组）不同。
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum ClipboardGroupFilter {
    All,
    Text,
    Image,
    Files,
    Favorite,
}

impl Default for ClipboardItemQuery {
    fn default() -> Self {
        Self {
            kind: None,
            group_id: None,
            favorite: None,
            pinned: None,
            group: None,
            keyword: None,
            sort: ClipboardItemSort::UpdatedAt,
            limit: 20,
            offset: 0,
        }
    }
}

/// 列表查询的一页结果：项 + 当前过滤下的总数 + 是否还有下一页。
/// `total` 让 Footer 等 UI 无需再单独 IPC `count_clipboard_items`，
/// `has_more` 由 Rust 用 `offset + list.len() < total` 精确计算，
/// 避免前端用 `len == page_size` 近似（恰好整除时多一次空请求）。
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ClipboardItemPage {
    pub list: Vec<ClipboardItem>,
    pub total: i64,
    pub has_more: bool,
}

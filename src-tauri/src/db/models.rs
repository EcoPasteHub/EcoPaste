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
    /// 去重指纹：`sha256(kind:content)`，由 `db::items::content_hash` 计算并在入库前比对。
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
    pub platform: Platform,
    pub note: Option<String>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,

    /// 来源应用名称。仅 list 查询通过 LEFT JOIN `clipboard_apps` 填充；
    /// 单条 `SELECT_ITEM` 路径与 `INSERT` 不读不写，`#[sqlx(default)]` 保证缺列时为 `None`。
    #[sqlx(default)]
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub source_app_name: Option<String>,
    /// 来源应用图标文件名（`<sha256>.png`）。同 [`source_app_name`] 由 list 查询补齐，
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
    /// Files 类型条目对应的文件 icon 路径 JSON（与 `content` 里的路径顺序一致）；
    /// 命令层在列表查询后按需填充，最多返回前 3 个，用于列表轻量渲染。
    /// 形如 `["/abs/a.png", null, "/abs/c.png"]`。
    #[sqlx(default)]
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub file_icon_paths: Option<String>,
}

/// 剪贴板来源应用（macOS bundle id / Windows exe 路径作主键），
/// 名称与图标按 id 去重共享，单条剪贴板记录通过 `source_app_id` 引用。
#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
#[serde(rename_all = "camelCase")]
pub struct ClipboardApp {
    pub id: String,
    pub name: String,
    /// `app-icons/<sha256>.png` 形式的文件名（无分片目录前缀）；无图标则 `None`。
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
    pub sort_order: i64,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum ClipboardItemSort {
    CreatedAtDesc,
    UseCountDesc,
}

impl Default for ClipboardItemSort {
    fn default() -> Self {
        Self::CreatedAtDesc
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", default)]
pub struct ClipboardItemQuery {
    pub kind: Option<ClipboardKind>,
    pub group_id: Option<String>,
    pub favorite: Option<bool>,
    pub pinned: Option<bool>,
    pub keyword: Option<String>,
    pub sort: ClipboardItemSort,
    pub limit: i64,
    pub offset: i64,
}

impl Default for ClipboardItemQuery {
    fn default() -> Self {
        Self {
            kind: None,
            group_id: None,
            favorite: None,
            pinned: None,
            keyword: None,
            sort: ClipboardItemSort::CreatedAtDesc,
            limit: 20,
            offset: 0,
        }
    }
}

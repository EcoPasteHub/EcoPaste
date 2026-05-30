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
            limit: 50,
            offset: 0,
        }
    }
}

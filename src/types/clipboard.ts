// 与 src-tauri/src/db/models.rs 对齐。
// Rust 侧 serde 全部 camelCase（枚举值用 lowercase），改 Rust 时记得同步这里。

export type ClipboardKind = "text" | "image" | "files";

export type ClipboardSubKind =
  | "rtf"
  | "html"
  | "url"
  | "email"
  | "color"
  | "path";

export type Platform = "macos" | "windows";

export type ClipboardItemSort = "createdAtDesc" | "useCountDesc";

export interface ClipboardItem {
  id: string;
  kind: ClipboardKind;
  subKind?: ClipboardSubKind | null;
  groupId?: string | null;
  // 复制时的前台应用 id（macOS bundle id / Windows exe 路径）；未识别为 null。
  sourceAppId?: string | null;
  content: string;
  contentHash: string;
  searchText?: string | null;
  size?: number | null;
  width?: number | null;
  height?: number | null;
  useCount: number;
  isFavorite: boolean;
  isPinned: boolean;
  platform: Platform;
  note?: string | null;
  // DateTime<Utc> 序列化为 ISO8601 字符串。
  createdAt: string;
  updatedAt: string;
}

export interface ClipboardApp {
  id: string;
  name: string;
  // `<sha256>.png` 文件名；无图标为 null。配合 `get_clipboard_app_icon_path` 命令解析为绝对路径。
  iconFile?: string | null;
  platform: Platform;
  createdAt: string;
  updatedAt: string;
}

export interface ClipboardGroup {
  id: string;
  name: string;
  sortOrder: number;
  createdAt: string;
}

// 对齐 Rust `#[serde(default)]`：全 optional，前端不传时由 Rust 侧填默认值
// （limit=50, offset=0, sort=createdAtDesc）。
export interface ClipboardItemQuery {
  kind?: ClipboardKind;
  groupId?: string;
  favorite?: boolean;
  pinned?: boolean;
  keyword?: string;
  sort?: ClipboardItemSort;
  limit?: number;
  offset?: number;
}

export interface ReadClipboardResult {
  item: ClipboardItem;
  deduplicated: boolean;
  captured: boolean;
}

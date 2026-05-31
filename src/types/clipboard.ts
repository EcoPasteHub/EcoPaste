// 与 Rust 端 db::models 对齐。serde 全部 camelCase（枚举值用 lowercase），改 Rust 时记得同步这里。

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
  /**
   * 复制时的前台应用 id（macOS bundle id / Windows exe 路径）；未识别为 null。
   */
  sourceAppId?: string | null;
  content: string;
  contentHash: string;
  searchText?: string | null;
  /**
   * 列表渲染用的纯文本摘要（最多 512 字符）。HTML/RTF 也只存纯文本截断；
   * Image/Files 为 null。完整内容在 `content`，预览/写回时再读。
   */
  summary?: string | null;
  /**
   * Files 类型专用：紧凑格式记录每个路径的类型，如 "d,f,f" 表示 [dir, file, file]。
   */
  fileTypes?: string | null;
  size?: number | null;
  width?: number | null;
  height?: number | null;
  useCount: number;
  isFavorite: boolean;
  isPinned: boolean;
  platform: Platform;
  note?: string | null;
  /**
   * DateTime<Utc> 序列化为 ISO8601 字符串。
   */
  createdAt: string;
  updatedAt: string;
}

export interface ClipboardApp {
  id: string;
  name: string;
  /**
   * `<sha256>.png` 文件名；无图标为 null。配合 `get_clipboard_app_icon_path` 命令解析为绝对路径。
   */
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
  updatedAt: string;
}

/**
 * 对齐 Rust `#[serde(default)]`：全 optional，前端不传时由 Rust 侧填默认值
 * （limit=50, offset=0, sort=createdAtDesc）。
 */
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

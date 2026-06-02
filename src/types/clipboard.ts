/**
 * 与 Rust `db::models::ClipboardItem` 对应（`serde(rename_all = "camelCase")`）。
 * 列表视图下 text 类型条目的 `content` / `searchText` 由 Rust 端置空，渲染走 `summary`。
 */

export type ClipboardKind = "text" | "image" | "files";

export type ClipboardSubKind =
  | "rtf"
  | "html"
  | "url"
  | "email"
  | "color"
  | "path";

export type ClipboardPlatform = "macos" | "windows";

export interface ClipboardItem {
  id: string;
  kind: ClipboardKind;
  subKind: ClipboardSubKind | null;
  groupId: string | null;
  sourceAppId: string | null;
  content: string;
  contentHash: string;
  searchText: string | null;
  summary: string | null;
  fileTypes: string | null;
  size: number | null;
  width: number | null;
  height: number | null;
  useCount: number;
  isFavorite: boolean;
  isPinned: boolean;
  platform: ClipboardPlatform;
  note: string | null;
  createdAt: string;
  updatedAt: string;
  /**
   * 后端 LEFT JOIN `clipboard_apps` 顺带返回的来源应用元信息。
   * `sourceAppIconPath` 已由 Rust 命令层用 `AppIconStore` 解析为绝对路径，
   * 前端直接 `convertFileSrc(sourceAppIconPath)` 渲染。
   * `sourceAppIconFile` 是 Rust 内部解析用的原始文件名，前端通常不需要。
   */
  sourceAppName?: string;
  sourceAppIconFile?: string;
  sourceAppIconPath?: string;
  /** image 条目的缩略图绝对路径（后端预处理返回）。 */
  imageThumbnailPath?: string;
  /** files 条目的预处理条目（与 content 中路径顺序一致，最多 3 项）。 */
  fileEntries?: FileEntry[];
}

export interface FileEntry {
  path: string;
  name: string;
  isDir: boolean;
  isImage: boolean;
  iconPath?: string;
}

export type ClipboardItemSort = "createdAtDesc" | "useCountDesc";

export type ClipboardGroup = "all" | "text" | "image" | "files" | "favorite";

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

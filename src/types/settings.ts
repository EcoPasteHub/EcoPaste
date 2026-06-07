import type { ClipboardItemSort } from "./clipboard";

/**
 * 设置数据契约，镜像 `src-tauri/src/settings/model.rs::Settings`。
 *
 * 字段命名、枚举字面量必须与 Rust 的 `serde` 序列化输出严格一致：
 * - struct 默认走 `rename_all = "camelCase"`；
 * - enum 默认走 `rename_all = "camelCase"`，特例已在每个类型上方注明。
 *
 * Rust 端新增/重命名字段时，**本文件必须同步修改**，否则前端读到的是空字段，组件渲染会失真而无报错。
 */

/** Rust enum `Theme`（`rename_all = "lowercase"`）。 */
export type Theme = "auto" | "light" | "dark";

/** Rust enum `Language`（手动 `serde(rename)`）。 */
export type Language = "zh-CN" | "en-US";

export type AutoPaste =
  | "disabled"
  | "singleClickPaste"
  | "doubleClickPaste"
  | "singleClickCopy"
  | "doubleClickCopy";

export type MiddleClickAction =
  | "disabled"
  | "singleClickPaste"
  | "singleClickPastePlain"
  | "singleClickCopy"
  | "singleClickCopyPlain";

export type ItemAction =
  | "paste"
  | "pastePlain"
  | "pastePath"
  | "copy"
  | "copyPlain"
  | "openLink"
  | "sendEmail"
  | "reveal"
  | "note"
  | "star"
  | "delete";

export type CaptureKind = "files" | "image" | "html" | "rtf" | "text";

export type RetentionUnit = "hours" | "days" | "weeks" | "months" | "forever";

export type WindowPosition = "remember" | "followCursor" | "center";

export type PreviewHoverDelayMs = "ms300" | "ms500" | "ms1000";

export type UpdateFrequency = "daily" | "weekly" | "monthly";

export interface General {
  autoStart: boolean;
  silentStart: boolean;
  trayIcon: boolean;
  dockIcon: boolean;
}

export interface Appearance {
  theme: Theme;
  language: Language;
}

export interface QuickPaste {
  enabled: boolean;
  modifier: string;
}

export interface Shortcuts {
  openClipboard: string;
  openPreference: string;
  pastePlain: string;
  quickPaste: QuickPaste;
}

export interface Content {
  autoPaste: AutoPaste;
  middleClick: MiddleClickAction;
  copyPlain: boolean;
  copyThenHideWindow: boolean;
  pastePlain: boolean;
  pasteFilesAsPath: boolean;
  showOriginalPreview: boolean;
  deleteConfirm: boolean;
  autoFavorite: boolean;
  updateOnReuse: boolean;
  sort: ClipboardItemSort;
  itemActions: ItemAction[];
  itemActionOrder: ItemAction[];
}

export interface Display {
  textMaxLines: number;
  imageMaxHeight: number;
  fileMaxCount: number;
}

export interface Capture {
  text: boolean;
  html: boolean;
  rtf: boolean;
  image: boolean;
  files: boolean;
  order: CaptureKind[];
}

export interface Sensitive {
  secretDetection: boolean;
}

export interface Retention {
  value: number;
  unit: RetentionUnit;
}

export interface History {
  retention: Retention;
  maxCount: number;
  cleanupIntervalHours: number;
}

export interface Search {
  defaultFocus: boolean;
  clearOnHide: boolean;
}

export interface Window {
  position: WindowPosition;
  scrollToTopOnOpen: boolean;
  selectAllGroupOnOpen: boolean;
  alwaysOnTop: boolean;
  allWorkspaces: boolean;
}

export interface Preview {
  hoverEnabled: boolean;
  hoverDelayMs: PreviewHoverDelayMs;
  spaceEnabled: boolean;
}

export interface Feedback {
  copySound: boolean;
}

export interface Filters {
  excludedAppIds: string[];
}

export interface Clipboard {
  capture: Capture;
  content: Content;
  display: Display;
  sensitive: Sensitive;
  history: History;
  search: Search;
  window: Window;
  preview: Preview;
  feedback: Feedback;
  filters: Filters;
}

export interface Update {
  autoCheck: boolean;
  frequency: UpdateFrequency;
  includeBeta: boolean;
}

export interface Settings {
  general: General;
  appearance: Appearance;
  shortcuts: Shortcuts;
  clipboard: Clipboard;
  update: Update;
}

/**
 * 任意层级可选的设置补丁，与 Rust 端 `update_settings` 的 `serde_json::Value` 深度合并语义对齐。
 * 数组字段按整体替换处理（与 Rust 的 deep_merge 行为一致），调用方需要传完整数组。
 */
export type SettingsPatch = DeepPartial<Settings>;

type DeepPartial<T> =
  T extends ReadonlyArray<infer U>
    ? ReadonlyArray<U>
    : T extends object
      ? { [K in keyof T]?: DeepPartial<T[K]> }
      : T;

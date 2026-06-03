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

export type AutoPaste = "disabled" | "singleClick" | "doubleClick";

export type ItemAction = "copy" | "pastePlain" | "note" | "star" | "delete";

export type RetentionUnit = "hours" | "days" | "weeks" | "months" | "forever";

export type SearchPosition = "top" | "bottom";

export type WindowPosition = "remember" | "followCursor" | "center";

export type PreviewHoverDelayMs = "ms300" | "ms500" | "ms1000";

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
  copyPlain: boolean;
  pastePlain: boolean;
  showOriginalPreview: boolean;
  deleteConfirm: boolean;
  autoFavorite: boolean;
  autoSortByFrequency: boolean;
  itemActions: ItemAction[];
}

export interface Retention {
  value: number;
  unit: RetentionUnit;
}

export interface History {
  retention: Retention;
  maxCount: number;
}

export interface Search {
  position: SearchPosition;
  defaultFocus: boolean;
  clearOnHide: boolean;
}

export interface Window {
  position: WindowPosition;
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
  scanDirs: string[];
}

export interface Clipboard {
  content: Content;
  history: History;
  search: Search;
  window: Window;
  preview: Preview;
  feedback: Feedback;
  filters: Filters;
}

export interface Update {
  autoCheck: boolean;
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

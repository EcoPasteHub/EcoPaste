// 与 src-tauri/src/settings/model.rs 对齐。
// Rust 侧 serde 全部 camelCase（枚举亦同），改 Rust 时记得同步这里。

export type Theme = "auto" | "light" | "dark";

export type Language = "zh-CN" | "zh-TW" | "en-US" | "ja-JP";

export type AutoPaste = "disabled" | "singleClick" | "doubleClick";

export type ItemAction = "copy" | "pastePlain" | "note" | "star" | "delete";

export type RetentionUnit = "hours" | "days" | "weeks" | "months" | "forever";

export type SearchPosition = "top" | "bottom";

export type WindowStyle = "standard" | "dock";

export type WindowPosition = "remember" | "followCursor" | "center";

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
  style: WindowStyle;
  position: WindowPosition;
  alwaysOnTop: boolean;
  allWorkspaces: boolean;
}

export interface Feedback {
  copySound: boolean;
}

export interface Clipboard {
  content: Content;
  history: History;
  search: Search;
  window: Window;
  feedback: Feedback;
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

export type SettingsPatch = DeepPartial<Settings>;

type DeepPartial<T> = T extends (infer U)[]
  ? U[]
  : T extends object
    ? { [K in keyof T]?: DeepPartial<T[K]> }
    : T;

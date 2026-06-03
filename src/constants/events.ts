/**
 * 与 Rust `app.emit` 事件名一一对应的常量表。
 * 命名采用 `domain://action` 形式（见 AGENTS.md）。
 */
export const TAURI_EVENT = {
  CLIPBOARD_MENU_ACTION: "clipboard://menu-action",
  CLIPBOARD_UPDATED: "clipboard://updated",
  CONTEXT_MENU_SHOW: "context-menu://show",
  KEYBOARD_NAV: "keyboard://nav",
  PREVIEW_UPDATED: "preview://updated",
  SETTINGS_UPDATED: "settings://updated",
  WINDOW_VISIBILITY: "window://visibility",
} as const;

export type TauriEvent = (typeof TAURI_EVENT)[keyof typeof TAURI_EVENT];

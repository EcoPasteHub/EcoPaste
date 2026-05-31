/**
 * 与 Rust `app.emit` 事件名一一对应的常量表。
 * 命名采用 `domain://action` 形式（见 AGENTS.md）。
 */
export const TAURI_EVENT = {
  SETTINGS_UPDATED: "settings://updated",
} as const;

export type TauriEvent = (typeof TAURI_EVENT)[keyof typeof TAURI_EVENT];

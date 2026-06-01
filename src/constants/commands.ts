/**
 * 与 Rust `#[tauri::command]` 函数名一一对应的常量表。
 * 改名时两侧同步，且严禁在调用处直接写字面量。
 */
export const TAURI_COMMAND = {
  GET_SETTINGS: "get_settings",
  SHOW_WINDOW: "show_window",
  UPDATE_SETTINGS: "update_settings",
} as const;

export type TauriCommand = (typeof TAURI_COMMAND)[keyof typeof TAURI_COMMAND];

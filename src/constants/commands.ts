/**
 * 与 Rust `#[tauri::command]` 函数名一一对应的常量表。
 * 改名时两侧同步，且严禁在调用处直接写字面量。
 */
export const TAURI_COMMAND = {
  COUNT_CLIPBOARD_ITEMS: "count_clipboard_items",
  GET_CLIPBOARD_IMAGE_PATH: "get_clipboard_image_path",
  GET_CLIPBOARD_ITEM: "get_clipboard_item",
  GET_SETTINGS: "get_settings",
  LIST_CLIPBOARD_ITEMS: "list_clipboard_items",
  PASTE_CLIPBOARD_ITEM: "paste_clipboard_item",
  SHOW_WINDOW: "show_window",
  UPDATE_SETTINGS: "update_settings",
} as const;

export type TauriCommand = (typeof TAURI_COMMAND)[keyof typeof TAURI_COMMAND];

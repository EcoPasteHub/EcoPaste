/**
 * 与 Rust `#[tauri::command]` 函数名一一对应的常量表。
 * 仅在 `src/commands/index.ts` 内使用——业务代码请直接 import 各命令包装函数，
 * 不要再裸写命令名字面量或在调用处引用这个表。
 */
export const TAURI_COMMAND = {
  COUNT_CLIPBOARD_ITEMS: "count_clipboard_items",
  DELETE_CLIPBOARD_ITEM: "delete_clipboard_item",
  GET_CLIPBOARD_IMAGE_PATH: "get_clipboard_image_path",
  GET_CLIPBOARD_ITEM: "get_clipboard_item",
  GET_CLIPBOARD_ITEM_CONTENT: "get_clipboard_item_content",
  GET_SETTINGS: "get_settings",
  LIST_CLIPBOARD_ITEMS: "list_clipboard_items",
  PASTE_CLIPBOARD_ITEM: "paste_clipboard_item",
  SET_MAIN_WINDOW_PINNED: "set_main_window_pinned",
  SHOW_WINDOW: "show_window",
  TOGGLE_CLIPBOARD_ITEM_FAVORITE: "toggle_clipboard_item_favorite",
  UPDATE_CLIPBOARD_ITEM_NOTE: "update_clipboard_item_note",
  UPDATE_SETTINGS: "update_settings",
  WRITE_TO_CLIPBOARD: "write_to_clipboard",
} as const;

export type TauriCommand = (typeof TAURI_COMMAND)[keyof typeof TAURI_COMMAND];

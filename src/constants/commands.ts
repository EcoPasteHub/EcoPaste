// Tauri 命令名常量。与 Rust 端 `#[tauri::command]` 函数名一一对应，
// 注册位置见 `src-tauri/src/lib.rs::invoke_handler`；改名需同步两侧。
// 仅收录前端实际调用的命令，未使用的命令不预先列入，避免悬挂常量。
export const TAURI_COMMAND = {
  DELETE_CLIPBOARD_ITEM: "delete_clipboard_item",
  GET_CLIPBOARD_APP_ICON_PATH: "get_clipboard_app_icon_path",
  GET_CLIPBOARD_IMAGE_PATH: "get_clipboard_image_path",
  GET_CLIPBOARD_ITEM: "get_clipboard_item",
  GET_SETTINGS: "get_settings",
  HIDE_WINDOW: "hide_window",
  LIST_ALL_APPS: "list_all_apps",
  LIST_CLIPBOARD_APPS: "list_clipboard_apps",
  LIST_CLIPBOARD_GROUPS: "list_clipboard_groups",
  LIST_CLIPBOARD_ITEMS: "list_clipboard_items",
  PASTE_CLIPBOARD_ITEM: "paste_clipboard_item",
  REFRESH_APPS: "refresh_apps",
  TOGGLE_CLIPBOARD_ITEM_FAVORITE: "toggle_clipboard_item_favorite",
  UPDATE_CLIPBOARD_ITEM_NOTE: "update_clipboard_item_note",
  UPDATE_SETTINGS: "update_settings",
  WRITE_TO_CLIPBOARD: "write_to_clipboard",
} as const;

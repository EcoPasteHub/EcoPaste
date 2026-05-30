// Tauri 事件名常量。与 Rust 端 emit 的字符串字面量一一对应；改动需同步 Rust：
// - WINDOW_VISIBILITY → src-tauri/src/window/mod.rs
// - CLIPBOARD_UPDATED → src-tauri/src/clipboard/ingest.rs（或检索同名常量）
// - KEYBOARD_NAV      → src-tauri/src/keyboard/windows.rs
export const TAURI_EVENT = {
  CLIPBOARD_UPDATED: "clipboard://updated",
  KEYBOARD_NAV: "keyboard://nav",
  WINDOW_VISIBILITY: "window://visibility",
} as const;

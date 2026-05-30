// 窗口 label 常量。与 Rust 端 `src-tauri/src/window/mod.rs` 的
// `MAIN_WINDOW_LABEL` / `PREFERENCE_WINDOW_LABEL` 一一对应；改名需同步。
export const WINDOW_LABEL = {
  MAIN: "main",
  PREFERENCE: "preference",
} as const;

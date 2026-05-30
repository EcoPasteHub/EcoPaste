//! `#[tauri::command]` 入口层（薄封装）：参数校验 + 调用 `clipboard` / `db` 下层，不写业务逻辑。
//!
//! awesome-rpc 只替换 IPC 传输层，命令仍走标准 `#[tauri::command]` + `generate_handler!`。

mod clipboard;
mod shortcut;
mod window;

// glob 再导出：`#[tauri::command]` 会在函数旁生成隐藏辅助项（`__cmd__*`），
// `generate_handler!` 依赖它们，故用 `*` 一并带出，而非逐个具名 re-export。
pub use clipboard::*;
pub use shortcut::*;
pub use window::*;

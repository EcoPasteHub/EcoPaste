//! `#[tauri::command]` 入口层（薄封装）：参数校验 + 调用 `clipboard` / `db` 下层，不写业务逻辑。

mod autostart;
mod clipboard;
mod drag;
mod link;
mod settings;
mod storage;
mod window;

// glob 再导出：`#[tauri::command]` 会在函数旁生成隐藏辅助项（`__cmd__*`），
// `generate_handler!` 依赖它们，故用 `*` 一并带出，而非逐个具名 re-export。
pub use autostart::*;
pub use clipboard::*;
pub use drag::*;
pub use link::*;
pub use settings::*;
pub use storage::*;
pub use window::*;

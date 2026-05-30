mod app_store;
mod cleanup;
mod detect;
mod guard;
mod ingest;
mod payload;
mod read;
mod sound;
mod source;
mod storage;
mod watcher;
mod write;

pub use app_store::AppIconStore;
pub use guard::WritebackGuard;
pub use ingest::build_item;
pub use payload::{ClipboardPayload, ImagePayload, TextPayload};
pub use read::ClipboardReader;
pub use source::detect_frontmost;
pub use storage::ImageStore;
pub use watcher::{init, materialize_source, persist_and_notify, CLIPBOARD_UPDATED_EVENT};
pub use write::write_to_clipboard;

#[cfg(test)]
pub(crate) mod test_lock {
    use std::sync::{Mutex, MutexGuard};

    /// 系统剪贴板是单一全局资源，触碰它的测试不能并行（否则相互覆盖内容而 flaky）。
    /// 这些测试统一持有此锁串行执行，即使用默认多线程 runner 也稳定。
    static LOCK: Mutex<()> = Mutex::new(());

    /// 获取串行锁。即便上一个持锁测试 panic 导致锁中毒，也恢复使用——
    /// 锁仅用于串行化，内部 `()` 无状态可破坏。
    pub fn serial() -> MutexGuard<'static, ()> {
        LOCK.lock().unwrap_or_else(|poisoned| poisoned.into_inner())
    }
}

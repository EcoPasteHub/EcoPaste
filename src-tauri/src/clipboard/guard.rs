//! 写回回环抑制。
//!
//! 应用自身写回剪贴板会触发 OS 监听，进而被当成一次新的
//! 复制再次入库，形成回环。写回前调用 [`WritebackGuard::suppress`] 登记将写入内容的
//! `content_hash`；监听回调读到内容后调用 [`WritebackGuard::should_skip`]，命中则跳过本次入库。
//!
//! 用 `content_hash` 比对而非简单布尔标记：避免「写回事件尚未到达就来了一次真实复制」
//! 误伤真实复制；同时带 TTL 兜底——若写回的内容与剪贴板现状完全相同（OS 可能不发变更事件），
//! 登记的指纹不会永久滞留导致后续同内容复制被吞。HTML/RTF 写回会同时写入纯文本回退，
//! 因此 guard 支持短期登记多个指纹。

use std::sync::Mutex;
use std::time::{Duration, Instant};

/// 登记的写回指纹在多久内有效。写回后监听事件通常在毫秒级到达，
/// 给足冗余但不至于长到误吞后续的真实复制。
const SUPPRESS_TTL: Duration = Duration::from_secs(2);

pub struct WritebackGuard {
    pending: Mutex<Vec<Pending>>,
}

struct Pending {
    content_hash: String,
    at: Instant,
}

impl Default for WritebackGuard {
    fn default() -> Self {
        Self {
            pending: Mutex::new(Vec::new()),
        }
    }
}

impl WritebackGuard {
    pub fn new() -> Self {
        Self::default()
    }

    /// 写回剪贴板前登记将写入内容的 `content_hash`。
    pub fn suppress(&self, content_hash: String) {
        let mut pending = self.pending.lock().expect("writeback guard poisoned");
        pending.retain(|p| p.at.elapsed() <= SUPPRESS_TTL);
        pending.push(Pending {
            content_hash,
            at: Instant::now(),
        });
    }

    /// 监听回调判断本次变更是否为自身写回所致：命中登记指纹（且未过期）则返回 `true`
    /// 并消费掉登记；否则返回 `false`。过期的登记顺带清理。
    pub fn should_skip(&self, content_hash: &str) -> bool {
        let mut pending = self.pending.lock().expect("writeback guard poisoned");
        pending.retain(|p| p.at.elapsed() <= SUPPRESS_TTL);

        let Some(index) = pending.iter().position(|p| p.content_hash == content_hash) else {
            return false;
        };
        pending.remove(index);

        true
    }

    /// 单测用：直接塞一条已过期登记。
    #[cfg(test)]
    fn suppress_expired_for_test(&self, content_hash: String) {
        let mut pending = self.pending.lock().expect("writeback guard poisoned");
        pending.push(Pending {
            content_hash,
            at: Instant::now() - SUPPRESS_TTL - Duration::from_millis(1),
        });
    }

    /// 单测用：返回当前登记数量。
    #[cfg(test)]
    fn pending_len_for_test(&self) -> usize {
        let pending = self.pending.lock().expect("writeback guard poisoned");

        pending.len()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn skips_once_then_resets() {
        let guard = WritebackGuard::new();
        guard.suppress("hash-a".to_owned());

        assert!(guard.should_skip("hash-a"));
        // 登记已消费，同内容的下一次（真实复制）不再被吞。
        assert!(!guard.should_skip("hash-a"));
    }

    #[test]
    fn supports_multiple_pending_hashes() {
        let guard = WritebackGuard::new();
        guard.suppress("hash-a".to_owned());
        guard.suppress("hash-b".to_owned());

        assert!(guard.should_skip("hash-a"));
        assert!(guard.should_skip("hash-b"));
        assert_eq!(guard.pending_len_for_test(), 0);
    }

    #[test]
    fn does_not_skip_unrelated_content() {
        let guard = WritebackGuard::new();
        guard.suppress("hash-a".to_owned());

        // 写回事件未到，先来了一次别的真实复制 → 不该被吞，登记仍在。
        assert!(!guard.should_skip("hash-b"));
        assert!(guard.should_skip("hash-a"));
    }

    #[test]
    fn expired_suppression_is_ignored() {
        let guard = WritebackGuard::new();
        guard.suppress_expired_for_test("hash-a".to_owned());

        assert!(!guard.should_skip("hash-a"));
    }
}

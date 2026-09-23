//! Pure destination identity and handoff policy for the Windows adapter.

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(super) struct WindowTarget {
    pub(super) hwnd: isize,
    pub(super) process_id: u32,
}

/// An HWND is only a destination while it still belongs to the captured external process.
pub(super) fn is_valid_target(target: WindowTarget, owner: Option<u32>, own_process: u32) -> bool {
    target.hwnd != 0
        && target.process_id != 0
        && target.process_id != own_process
        && owner == Some(target.process_id)
}

pub(super) fn select_target(
    current: Option<WindowTarget>,
    origin: Option<WindowTarget>,
    valid: impl Fn(WindowTarget) -> bool,
) -> Option<WindowTarget> {
    current
        .filter(|target| valid(*target))
        .or_else(|| origin.filter(|target| valid(*target)))
}

/// Do not inject until focusability reset and optional hiding have actually reached the native window.
pub(super) fn ready_to_paste(
    live: bool,
    foreground: bool,
    nonfocusable: bool,
    visible: bool,
    pinned: bool,
) -> bool {
    live && foreground && nonfocusable && (pinned || !visible)
}

#[derive(Default)]
pub(super) struct PasteSession {
    pub(super) origin: Option<WindowTarget>,
    pub(super) generation: u64,
    open: bool,
}

impl PasteSession {
    /// Repeated showing from an edit control keeps the same session; reopening starts fresh.
    pub(super) fn show(&mut self, current: Option<WindowTarget>, visible: bool) {
        if !visible || current.is_some() {
            self.origin = current;
        }
        self.open = true;
        self.generation = self.generation.wrapping_add(1);
    }

    /// Redundant hide callbacks must not invalidate a handoff already acknowledged after hiding.
    pub(super) fn close(&mut self) {
        if !self.open {
            return;
        }
        self.open = false;
        self.origin = None;
        self.generation = self.generation.wrapping_add(1);
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    const OWN: u32 = 10;
    const A: WindowTarget = WindowTarget {
        hwnd: 100,
        process_id: 20,
    };
    const B: WindowTarget = WindowTarget {
        hwnd: 200,
        process_id: 30,
    };

    #[test]
    fn rejects_missing_self_and_reused_handles() {
        assert!(!is_valid_target(A, None, OWN));
        assert!(!is_valid_target(A, Some(30), OWN));
        assert!(!is_valid_target(
            WindowTarget {
                hwnd: 300,
                process_id: OWN
            },
            Some(OWN),
            OWN
        ));
        assert!(!is_valid_target(
            WindowTarget {
                hwnd: 0,
                process_id: 20
            },
            Some(20),
            OWN
        ));
        assert!(is_valid_target(A, Some(20), OWN));
    }

    #[test]
    fn current_external_target_wins_but_invalid_current_uses_retained_origin() {
        assert_eq!(select_target(Some(B), Some(A), |_| true), Some(B));
        assert_eq!(
            select_target(Some(B), Some(A), |target| target == A),
            Some(A)
        );
        assert_eq!(select_target(None, Some(A), |_| false), None);
        assert_eq!(select_target(None, None, |_| true), None);
    }

    #[test]
    fn visible_editing_keeps_origin_and_fresh_show_clears_stale_origin() {
        let mut session = PasteSession::default();
        session.show(Some(A), false);
        session.show(None, true);
        assert_eq!(session.origin, Some(A));
        session.show(Some(B), true);
        assert_eq!(session.origin, Some(B));
        session.show(None, false);
        assert_eq!(session.origin, None);
    }

    #[test]
    fn close_invalidates_session_once_and_new_show_cancels_handoff() {
        let mut session = PasteSession::default();
        session.show(Some(A), false);
        let shown = session.generation;
        session.close();
        assert_ne!(shown, session.generation);
        assert_eq!(session.origin, None);
        let handoff = session.generation;
        session.close();
        assert_eq!(handoff, session.generation);
        session.show(None, false);
        assert_ne!(handoff, session.generation);
    }

    #[test]
    fn dispatch_requires_verified_destination_and_finished_window_reset() {
        assert!(!ready_to_paste(false, true, true, false, false));
        assert!(!ready_to_paste(true, false, true, false, false));
        assert!(!ready_to_paste(true, true, false, true, true));
        assert!(!ready_to_paste(true, true, true, true, false));
        assert!(ready_to_paste(true, true, true, true, true));
        assert!(ready_to_paste(true, true, true, false, false));
    }
}

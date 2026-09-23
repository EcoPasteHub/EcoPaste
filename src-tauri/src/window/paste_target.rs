//! Platform-independent policy for retaining a paste destination through delayed panel showing.

pub(super) struct PasteSession<T> {
    origin: Option<T>,
    pending_show: bool,
    closed: bool,
    generation: u64,
}

impl<T> Default for PasteSession<T> {
    fn default() -> Self {
        Self {
            origin: None,
            pending_show: false,
            closed: true,
            generation: 0,
        }
    }
}

impl<T> PasteSession<T> {
    /// Capture before delayed showing; only an existing visible/pending session may keep its origin.
    pub(super) fn begin_show(&mut self, external: Option<T>, visible: bool) -> u64 {
        if (!visible && !self.pending_show) || external.is_some() {
            self.origin = external;
        }
        self.pending_show = true;
        self.closed = false;
        self.generation = self.generation.wrapping_add(1);
        self.generation
    }

    pub(super) fn close(&mut self) {
        if self.closed {
            return;
        }
        self.closed = true;
        self.origin = None;
        self.cancel_pending_show();
    }

    pub(super) fn mark_shown(&mut self) {
        self.pending_show = false;
    }

    pub(super) fn cancel_pending_show(&mut self) {
        self.pending_show = false;
        self.generation = self.generation.wrapping_add(1);
    }

    pub(super) fn generation(&self) -> u64 {
        self.generation
    }

    pub(super) fn is_current(&self, generation: u64) -> bool {
        self.generation == generation
    }

    pub(super) fn origin(&self) -> Option<&T> {
        self.origin.as_ref()
    }
}

/// Prefer a live current destination, then a live retained origin; never infer destination from content.
pub(super) fn select_target<T>(
    current: Option<T>,
    origin: Option<T>,
    valid: impl Fn(&T) -> bool,
) -> Option<T> {
    current.filter(&valid).or_else(|| origin.filter(valid))
}

/// App activation alone is insufficient while a nonactivating panel still owns keyboard focus.
pub(super) fn ready_to_paste(
    live: bool,
    frontmost: bool,
    panel_key: bool,
    panel_visible: bool,
    pinned: bool,
) -> bool {
    live && frontmost && !panel_key && (pinned || !panel_visible)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn repeated_show_from_self_preserves_origin_but_new_session_clears_it() {
        let mut session = PasteSession::default();
        session.begin_show(Some("external"), false);
        session.begin_show(None, false);
        assert_eq!(session.origin(), Some(&"external"));
        session.mark_shown();
        session.begin_show(None, true);
        assert_eq!(session.origin(), Some(&"external"));
        session.close();
        session.begin_show(None, false);
        assert_eq!(session.origin(), None);
    }

    #[test]
    fn hide_and_new_show_invalidate_queued_show_callbacks() {
        let mut session = PasteSession::default();
        let first = session.begin_show(Some("first"), false);
        let second = session.begin_show(Some("second"), false);
        assert!(!session.is_current(first));
        assert!(session.is_current(second));
        session.close();
        assert!(!session.is_current(second));
    }

    #[test]
    fn duplicate_resign_hide_does_not_cancel_an_acknowledged_paste_handoff() {
        let mut session = PasteSession::default();
        session.begin_show(Some("external"), false);
        session.mark_shown();
        session.close();
        let handoff = session.generation();
        session.close();
        assert!(session.is_current(handoff));
        session.begin_show(None, false);
        session.close();
        assert!(!session.is_current(handoff));
    }

    #[test]
    fn hidden_window_starts_a_fresh_session_even_without_a_hide_callback() {
        let mut session = PasteSession::default();
        session.begin_show(Some("old"), false);
        session.mark_shown();
        session.begin_show(None, false);
        assert_eq!(session.origin(), None);
    }

    #[test]
    fn pinned_switch_prefers_the_current_external_application() {
        assert_eq!(
            select_target(Some("new"), Some("old"), |_| true),
            Some("new")
        );
        assert_eq!(
            select_target(Some("self"), Some("old"), |app| *app != "self"),
            Some("old")
        );
    }

    #[test]
    fn missing_self_and_terminated_targets_never_receive_paste() {
        let live = |app: &&str| *app != "self" && *app != "terminated";
        assert_eq!(select_target(Some("self"), None, live), None);
        assert_eq!(select_target(None, Some("terminated"), live), None);
        assert_eq!(select_target(None, Some("live"), live), Some("live"));
    }

    #[test]
    fn dispatch_requires_live_frontmost_target_and_completed_panel_handoff() {
        assert!(!ready_to_paste(false, true, false, false, false));
        assert!(!ready_to_paste(true, false, false, false, false));
        assert!(!ready_to_paste(true, true, true, true, true));
        assert!(!ready_to_paste(true, true, false, true, false));
        assert!(ready_to_paste(true, true, false, true, true));
        assert!(ready_to_paste(true, true, false, false, false));
    }
}

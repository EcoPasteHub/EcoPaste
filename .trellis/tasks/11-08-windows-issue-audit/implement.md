# Implementation Plan

## Ordered Changes

1. Add regression tests and logic for the Win+V chord latch.
2. Ignore IME composition events in the shared keyboard hook and verify editable
   focus handoff behavior.
3. Bind Windows keyboard navigation activation to clicks inside/outside the
   pinned clipboard window.
4. Add a shared bounded WebView readiness probe and use it in context-menu and
   preview rebuild paths.
5. Preserve StartupApproved state with an exact Run-value no-op check.
6. Change the admin task to `ONLOGON`, coordinate it with `autoStart`, and avoid
   duplicate Run/task login launch paths.
7. Make storage-directory adoption non-destructive and preserve a temporarily
   missing custom manifest.
8. Retry Windows empty clipboard reads and add source-aware Office/WPS text
   priority.
9. Make the preference sidebar navigation scrollable.
10. Update project specs for the new native contracts.
11. Add scale-aware window geometry persistence and Remote Desktop clamping.
12. Retain context-menu/submenu WebViews after first creation.
13. Review and update concise core memory specs.
14. Build the NSIS installer, replace the installed app, and verify launch.
15. Commit on a new `codex/` branch and push to `origin`.

## Validation

- Focused Rust unit tests for pure decisions and parsing.
- `cargo fmt --check`
- `cargo clippy -- -D warnings`
- `cargo test`
- `pnpm lint`
- `pnpm tsc`
- Build the Windows application and manually exercise:
  - Win+V hold/repress behavior;
  - search IME Enter;
  - pinned window then Space in another app;
  - 200% preference layout;
  - context menu/preview after idle destroy;
  - storage adoption with a temporary test directory;
  - autostart/task registry state where elevation is available.
  - simulated saved-geometry scale changes and monitor-bound clamping.
  - installed application launch after replacement.

## Risk and Rollback Points

- Keyboard hooks and autostart are Windows-native and require manual validation
  beyond compilation.
- Scheduled-task and registry tests must not alter a user's existing `EcoPaste`
  task or Run entry during automated tests.
- If native validation cannot be completed because elevation/WebView2 is
  unavailable, report the exact unverified path instead of claiming completion.

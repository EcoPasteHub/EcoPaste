# Toggle shortcut hint overlays

## Goal

Add a persisted preference that hides Command/Ctrl key-hint overlays without disabling shortcuts.

## Requirements

- Add a persisted `shortcuts.showHints` boolean setting that defaults to `true`.
- Existing settings files that omit the field must keep shortcut hints enabled.
- Add a Preferences switch with matching zh-CN and en-US copy.
- When disabled, `KeyHint` must keep its normal icon visible and suppress only
  the Command/Ctrl overlay.
- Disabling the overlay must not disable or change any keyboard shortcut.
- The behavior must remain consistent in the macOS and Windows clipboard
  windows.

## Acceptance Criteria

- [ ] Holding Command on macOS or Ctrl on Windows shows hints when enabled.
- [ ] Holding the modifier shows no hint overlay when disabled.
- [ ] Modifier shortcuts still call their existing actions when hints are off.
- [ ] Missing-field and explicit-false settings cases are covered by Rust tests.
- [ ] Frontend lint/type-check and Rust format/Clippy/tests pass.

## Notes

- This is a lightweight settings/UI task; no database or command contract changes.

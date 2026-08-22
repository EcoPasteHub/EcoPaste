# Bottom dock clipboard history

## Goal

Add a persisted full-width horizontal clipboard shelf at the bottom of the active display while retaining the existing panel layout.

## Requirements

- Persist a `bottom` clipboard-window position while preserving the existing panel position.
- Show the dock on the active monitor, spanning its work-area width at the bottom edge.
- Persist a dock-specific resizable height independently from the panel window state.
- Render the complete clipboard history left-to-right with virtualization and preserve search, groups, selection, paste, context menus, and drag behavior.
- Use unmodified Left/Right for dock-card navigation; preserve panel category navigation and modifier-based category shortcuts.
- Apply position changes immediately and deserialize settings written by older versions.
- Provide matching Chinese and English settings/help copy.
- Support the existing macOS panel and Windows window-hook implementations.

## Acceptance Criteria

- [ ] Switching between panel and bottom layouts applies immediately and survives restart.
- [ ] The dock matches the active monitor work area, restores its own height, and stays between 220 logical pixels and half the work-area height.
- [ ] Horizontal history remains virtualized and all existing clipboard actions still work.
- [ ] Keyboard navigation behaves correctly in both layouts.
- [ ] The panel layout remains unchanged.
- [ ] Frontend and Rust formatting, lint, type, build, clippy, and test checks pass.
- [ ] The primary macOS dock flow is verified in the packaged app.

## Notes

- Fresh in-code defaults may use the bottom layout, but released settings files must keep their persisted position.

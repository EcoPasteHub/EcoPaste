# Design

## Settings and geometry

- Extend `WindowPosition` with `Bottom`; keep missing-field defaults compatible with existing settings.
- Keep separate `clipboard` and `clipboard-dock` window-state labels.
- Resolve the monitor under the cursor, falling back to the primary monitor.
- Span the monitor work area and clamp height to 220 logical pixels through half the work-area height, defaulting to 320 logical pixels.
- Keep the panel at its current 360×600 geometry.

## UI and interaction

- Reuse the existing Virtuoso list with horizontal mode and fixed 240-pixel card width.
- Reuse existing cards, actions, groups, and data loading; only adapt layout classes and navigation by position.
- Emit a prepare-hide event so the bottom shelf can finish its reduced-motion-aware slide before Rust hides the window.
- Keep platform-specific behavior behind the existing macOS and Windows window modules.

## Data flow

1. The preference writes the persisted window position through the existing settings command.
2. Rust applies the matching window-state label and monitor-relative geometry.
3. The frontend settings mirror selects panel or dock layout without duplicating clipboard data.

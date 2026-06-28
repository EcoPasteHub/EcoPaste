# Thinking Guides

Use these guides when a change could drift across EcoPaste's Rust, React,
database, settings, or native platform boundaries.

## Guides

| Guide | Use When |
| --- | --- |
| [Code Reuse Thinking Guide](./code-reuse-thinking-guide.md) | Adding constants, wrappers, helpers, settings, commands, actions, or repeated UI |
| [Cross-Layer Thinking Guide](./cross-layer-thinking-guide.md) | Changing data that moves between Rust, SQLite, Tauri events, TypeScript, and UI |

## Fast Routing

- Clipboard capture/writeback/search/persistence: start in
  [backend/clipboard-pipeline](../backend/clipboard-pipeline.md) and
  [backend/database-and-storage](../backend/database-and-storage.md).
- Commands/events/settings mirrors: read
  [backend/commands-and-events](../backend/commands-and-events.md) and
  [frontend/state-management](../frontend/state-management.md).
- UI components/hooks: read the frontend guide for that topic.
- Window, shortcut, tray, autostart, or platform behavior: read
  [backend/settings-window-platform](../backend/settings-window-platform.md).

## Search Habit

Use `rg` before adding a new contract or value. Most cross-layer bugs here come
from updating one mirror and missing another.

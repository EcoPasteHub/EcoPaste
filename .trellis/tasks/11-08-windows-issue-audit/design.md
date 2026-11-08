# Technical Design

## Architecture Boundaries

- Rust owns clipboard reads, keyboard hooks, autostart/task scheduling, storage
  directory switching, and WebView readiness.
- React only changes layout and ignores IME composition events before forwarding
  keyboard actions.
- Existing Tauri lifecycle state remains the authority for destroy/rebuild
  readiness.

## Keyboard Hooks

### Win+V

Replace the per-V-key consumed flag with a chord latch. The low-level hook resets
the latch whenever Win is not physically down; while Win stays down, only the
first V press toggles the clipboard window and all repeated V events remain
consumed.

### Pinned clipboard window

The Windows keyboard hook is only active while the clipboard window is the
intended keyboard target. The existing global mouse hook already receives clicks
outside the window. When a pinned window receives an outside click, disable the
navigation keyboard hook; when a click lands inside the clipboard window,
reenable it. This preserves normal typing in other apps without changing the
window's pinned visibility.

### IME

`useKeyboardEvent` drops browser and synthesized keydown events marked as IME
composition (`isComposing`, Chromium key code 229) before either editable handoff
or global shortcut dispatch. `SearchInput` continues to handle composition
boundaries itself.

## Destroyable WebViews

Context-menu and preview windows can be recreated after idle destruction. Tauri's
window methods that proxy into the WebView may fail before its runtime is ready.
Add one shared async readiness probe in the window layer that polls the rebuilt
window's scale factor with a short bounded timeout. Menu and preview show paths
await that probe before sizing, positioning, emitting, or showing.

## Windows Autostart

### StartupApproved

When the current user's Run value already matches the exact executable and
`--auto-launch` arguments, autostart enable is a no-op. The app therefore stops
rewriting a stable Run value on every startup and preserves Windows'
StartupApproved disabled bit.

### Administrator login task

The highest-privilege task is created with `ONLOGON`, not `ONCE`. It exists only
when both `runAsAdmin` and `autoStart` are enabled. In that state the normal Run
entry is removed to avoid two login launches. If the task cannot be created
(for example, the current process is not elevated), the Run entry remains as the
fallback.

Early startup settings parsing reads both flags so an admin-only configuration
that does not enable autostart does not recreate a login task.

## Storage Safety

- If a manifest points to a temporarily missing directory, keep the manifest and
  fall back to the default data root for the current process.
- Before moving data, detect whether the validated target already contains
  EcoPaste data. Adopt such a target directly instead of deleting each content
  directory and copying the current/default payload into it.
- Keep the current data root when adopting an existing target. Only the copy
  path removes the previous root after a successful migration.

## Clipboard Capture

- On Windows, an empty first read is retried with the existing bounded backoff
  because delayed-rendering applications can announce a clipboard update before
  all formats are ready. Errors continue to retry on both platforms.
- The reader receives the source executable path as a hint. For known Office/WPS
  executables, when both image and text-family formats exist, text-family capture
  is considered before image capture. Mixed representations from browsers and
  other apps retain the configured priority.

## Preference Layout

Make the sidebar navigation its own vertical scroll area while keeping the logo
and storage summary fixed. This prevents the bottom navigation item from being
clipped at high DPI or short logical viewport heights.

## Remote Desktop DPI and Resolution Changes

- Persist physical position as Windows uses it, but persist the source scale
  factor and derive logical window width/height from the saved physical size.
- On restore, choose the monitor containing the saved position, convert logical
  size to that monitor's current scale, clamp to the monitor bounds, then clamp
  position so the full window remains visible.
- Treat `monitor_from_point` coordinates as physical coordinates. Converting the
  cursor to logical coordinates before the lookup picks the wrong monitor under
  mixed DPI.
- Handle live `ScaleFactorChanged` events by converting from the last known
  logical size, clamping to the current monitor, and refreshing the in-memory
  geometry snapshot.

## Warm Context Menus

The context-menu and submenu WebViews are created lazily on first use but retained
after hiding. `DestroyWhenIdle` remains for preference and preview windows only.
This trades a small amount of idle memory for predictable right-click latency,
especially after long idle or Remote Desktop reconnection.

## Local Installation

- Build an NSIS installer from the current tree.
- Stop the running installed instance before replacement.
- Install silently into the existing Program Files location when the NSIS
  installer recognizes the prior machine-wide install.
- Verify executable path, timestamp/version, installed files, and successful app
  launch. Keep existing user data and settings.

## Compatibility and Rollback

- No settings or database schema migration is required.
- Autostart and scheduled-task changes are reversible when their settings are
  turned off.
- Storage adoption keeps persisted data intact; rollback is restoring the
  bootstrap manifest to the previous root.
- New behavior is narrowly gated to Windows where platform-specific.
- Window state JSON remains backward compatible: missing scale metadata is read
  as unknown and falls back to current scale.

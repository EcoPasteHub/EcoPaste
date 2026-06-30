# Architecture

## Startup and Managed State

`src-tauri/src/lib.rs` is the composition root. It registers plugins, commands,
menu handlers, managed state, and run-loop event handlers. The setup order
matters:

- `settings::init` runs before shortcut/tray setup so those modules receive the
  current settings snapshot.
- `db::init` creates the SQLite pool and runs migrations before
  `clipboard::init`, because the watcher needs `DatabaseState`.
- `clipboard::init` manages `WritebackGuard`, `ImageStore`, `AppIconStore`,
  `FileIconStore`, `AppsRegistry`, and `WatcherPause`.
- `WindowLifecycleManager` is managed before windows report readiness through
  `notify_window_ready`.

Do not create ad hoc SQLite pools or independent settings readers in feature
modules. Use `State<DatabaseState>` and `State<SettingsStore>` at command
boundaries, then pass snapshots or cloned pools to lower layers.

Reference files:
- `src-tauri/src/lib.rs`
- `src-tauri/src/db/state.rs`
- `src-tauri/src/settings/store.rs`
- `src-tauri/src/clipboard/watcher.rs`

## Module Ownership

Keep commands thin but not empty. A command may validate external input,
coordinate Tauri state, and shape frontend-ready view models. Business rules
should live below `commands/` when they are reusable by watcher, tests, menu, or
storage flows.

Examples:
- `commands::clipboard::list_clipboard_items` reads settings, queries
  `db::items::query_items_page`, attaches thumbnails, file entries, source app
  icons, color previews, display timestamps, redaction, and action lists.
- `clipboard::persist_and_notify` is shared by the OS watcher and
  `read_clipboard`, so manual and automatic capture have the same dedupe and
  event semantics.
- `settings::SettingsStore::update` performs deep merge, validation, and atomic
  persistence; `commands::settings::update_settings` adds side effects and emits
  `settings://updated`.

Avoid moving these decisions into React. If the frontend starts checking whether
an item can reveal in Finder, parsing file entries, or deciding sensitive
redaction, the boundary has drifted.

## Platform Boundary

EcoPaste supports macOS and Windows only. Platform code uses
`#[cfg(target_os = "macos")]` and `#[cfg(target_os = "windows")]` modules:

- `window/macos.rs` turns the clipboard window into an NSPanel and uses main-thread
  panel operations.
- `window/windows.rs` keeps the clipboard window non-focusable and enables keyboard
  and mouse hooks while visible.
- `keystroke/macos.rs` and `keystroke/windows.rs` simulate paste.
- `drag_out/macos.rs` and `drag_out/windows.rs` implement platform drag-out.

For cross-platform features, implement both branches or make the unsupported
branch explicit in code and UI copy. Do not add Linux cfgs, dependencies, or
documentation promises.

## Error and Logging Shape

Commands return `crate::core::Result<T>`, where `AppError` serializes as
`{ kind, message }` in `src-tauri/src/core/error.rs`. Keep `message` focused on
the root cause because the frontend command wrapper adds the action label in the
toast.

Use `anyhow::Context` for technical context and `log::*` for diagnostics.
Prefer user-readable messages at conversion points such as filename validation
or missing clipboard items.

## Async and Threading Constraints

Some OS handles are not `Send`. The clipboard code keeps those handles inside a
synchronous section or a dedicated thread:

- `commands::clipboard::read_clipboard` reads the clipboard and drops
  `ClipboardReader` before awaiting database writes.
- `clipboard::watcher` creates `ClipboardReader` on the watcher thread and only
  moves `Send` data into `tauri::async_runtime::spawn`.
- `clipboard::write::write_to_clipboard` completes the `ClipboardContext`
  writeback synchronously before the command awaits anything else.

If a future command touches clipboard, native windows, or OS hooks, check whether
the handle can cross `await`. When in doubt, keep native handle work in a
bounded synchronous block and pass owned data into async code.

## Comments

Rust code in this repository often uses `///` and module docs to record hidden
constraints, such as NSPanel focus timing or clipboard loop suppression. Add a
short comment when a behavior depends on platform timing, external OS quirks, or
cross-layer contracts. Do not restate obvious assignments.

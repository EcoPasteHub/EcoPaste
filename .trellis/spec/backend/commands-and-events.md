# Commands and Events

## Command Registration

Every public Tauri command must be registered in `tauri::generate_handler!` in
`src-tauri/src/lib.rs`, re-exported by `src-tauri/src/commands/mod.rs`, mirrored
in `src/constants/commands.ts`, and wrapped once in `src/commands/index.ts`.

The frontend rule is strict: callers import wrappers from `@/commands`. They do
not call `invoke` directly or import `TAURI_COMMAND` outside `src/commands`.

Reference files:
- `src-tauri/src/lib.rs`
- `src-tauri/src/commands/mod.rs`
- `src/constants/commands.ts`
- `src/commands/index.ts`

## Command Boundary Responsibilities

Commands may:

- Validate external strings, paths, IDs, and enum inputs.
- Acquire `State<DatabaseState>`, `State<SettingsStore>`, resource stores, and
  `AppHandle`.
- Coordinate one-off side effects such as hiding a window before paste.
- Convert repository models into frontend-ready response models.
- Emit events after successful persistence.

Commands should not:

- Open new database connections.
- Reimplement repository filtering or persistence rules inline.
- Let native non-`Send` handles cross an `await`.
- Return raw file names when the frontend needs an absolute path. See
  `get_clipboard_image_path`, `get_clipboard_app_icon_path`, and list item
  enrichment.

Good local examples:
- `commands::clipboard::paste_clipboard_item` handles the writeback, hide/resign
  timing, paste simulation, and reuse update.
- `commands::settings::update_settings` applies `SettingsStore::update`, then
  re-registers shortcuts and tray behavior when relevant sections changed.
- `commands::window::open_preference_with_highlight` delegates to the Rust
  pending-slot mechanism instead of asking React to emit into a rebuilding
  window.

## Event Naming and Payloads

Event names use `domain://action`. Keep Rust constants close to emit sites and
mirror them in `src/constants/events.ts`.

Current high-value events:

| Event | Rust Emitters | Frontend Consumers |
| --- | --- | --- |
| `clipboard://updated` | `clipboard/watcher.rs`, `commands/clipboard.rs`, `commands/storage.rs`, `backup/mod.rs` | Clipboard list refresh and deferred reload logic |
| `clipboard-groups://updated` | `commands/clipboard.rs` | Group lists and empty state labels |
| `settings://updated` | `commands/settings.rs`, `commands/storage.rs`, `backup/mod.rs` | `src/stores/settings.ts` mirror |
| `window://visibility` | `window/mod.rs` | Clipboard focus/search and dormant refresh gates |
| `window://lifecycle` and `window://before-destroy` | `window/lifecycle/mod.rs` | Window lifecycle mirror and destroy protection |
| `keyboard://nav` | `keyboard/windows.rs` | `useKeyboardEvent` on Windows main window |
| `backup://received` | `backup/mod.rs` | Preference backup import flow |

When adding an event, define the payload shape in Rust and in TypeScript near
the consumer. Prefer small payloads that let the frontend fetch the latest state
with an existing command when payloads would otherwise grow large.

## Error Surface

`src/commands/index.ts` normalizes rejected values into `AppError`, logs through
`@/utils/log`, shows a localized command error toast, and rethrows. Callers
should catch only when they need local control flow, not to show another generic
error toast.

Rust command errors should be user-readable at the message level. Avoid action
prefixes like "save settings failed"; the frontend toast already supplies the
action label from `src/locales/*/commands.json`.

## Contract Checklist

When adding or renaming a command:

- Register it in `src-tauri/src/lib.rs`.
- Export it from `src-tauri/src/commands/mod.rs`.
- Add `TAURI_COMMAND` in `src/constants/commands.ts`.
- Add one wrapper in `src/commands/index.ts` with a command label key.
- Add `commands:labels.*` entries in both `src/locales/zh-CN/commands.json` and
  `src/locales/en-US/commands.json`.
- Update TypeScript types when Rust returns a new shape.

When adding an event:

- Add a Rust constant near the emitter.
- Add `TAURI_EVENT` in `src/constants/events.ts`.
- Define a typed payload at the consuming module.
- Decide whether hidden or dormant windows should process it immediately or
  defer work until `window://visibility`.

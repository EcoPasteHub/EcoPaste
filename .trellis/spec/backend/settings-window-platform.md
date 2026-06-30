# Settings, Windowing, and Platform

## Settings Model

Settings live in Rust under `src-tauri/src/settings/`.

- `model.rs` defines the serialized shape, defaults, enums, and helper methods.
- Every settings struct uses `#[serde(default, rename_all = "camelCase")]`, so
  newly added fields fall back to defaults.
- `store.rs` persists `<app_data_dir>/config/settings.json` with a temp file and
  atomic rename.
- `SettingsStore::update` accepts only object patches, deep-merges objects,
  replaces arrays wholesale, validates cross-field constraints, writes to disk,
  and updates the in-memory snapshot.

When adding a setting, update:

- Rust model and defaults.
- Validation if the field interacts with another field.
- TypeScript `src/types/settings.ts`.
- Preference schema in `src/pages/Preference/config/preferenceSchema.ts` if the
  setting is user-visible.
- zh-CN and en-US locale entries.
- Side effects in `commands/settings.rs` or
  `src/pages/Preference/services/preferenceSettings.ts` when OS state changes
  immediately.

### Window Open Selection Settings

#### 1. Scope / Trigger

Changing the default filter applied when the clipboard window opens is a
settings contract change. Rust owns the persisted shape; React only mirrors it
and applies the transient `clipboardViewState` side effect on `window://visibility`.

#### 2. Signatures

- Rust fields:
  - `Window.select_range_on_open: WindowOpenRangeSelection`
  - `Window.select_category_on_open: WindowOpenCategorySelection`
  - `Window.select_group_on_open: String`
- TypeScript mirror:
  - `Window.selectRangeOnOpen: "preserve" | ClipboardRange`
  - `Window.selectCategoryOnOpen: "preserve" | "all" | ClipboardCategory`
  - `Window.selectGroupOnOpen: "preserve" | "all" | \`group:${string}\``

#### 3. Contracts

- `preserve` means keep the user's last selected value for that dimension.
- `all` means clear that dimension to the built-in All state.
- `group:<id>` means select the custom clipboard group with that id.
- Range options are `preserve`, `all`, and `favorite`.
- Category options are `preserve`, `all`, `text`, `image`, and `files`.

#### 4. Validation & Error Matrix

- Invalid range/category literals fail serde deserialization.
- `select_group_on_open = "preserve"` -> valid.
- `select_group_on_open = "all"` -> valid.
- `select_group_on_open = "group:<non-empty id>"` -> valid.
- Any other group string, including `group:` -> `AppError::Other("open group selection is invalid")`.

#### 5. Good/Base/Bad Cases

- Good: selecting `favorite`, `image`, and `group:<uuid>` opens into favorites,
  image category, and that custom group.
- Base: all three fields default to `preserve`, so opening the window keeps the
  user's last filters.
- Bad: storing a raw group UUID without the `group:` prefix is rejected.

#### 6. Tests Required

- Missing-field defaults assert all three fields default to `preserve`.
- Settings validation rejects invalid `select_group_on_open` strings.
- Frontend type-check covers the mirrored selection unions and the dynamic group
  Select value shape.

#### 7. Wrong vs Correct

Wrong:

```json
{ "clipboard": { "window": { "selectGroupOnOpen": "550e8400-e29b" } } }
```

Correct:

```json
{ "clipboard": { "window": { "selectGroupOnOpen": "group:550e8400-e29b" } } }
```

## Settings Events and Mirrors

Rust is the source of truth. `commands::settings::update_settings` and
`reset_settings` emit `settings://updated` after persistence and side effects.
`src/stores/settings.ts` listens to that event and updates the Valtio mirror.

Do not mutate `settingsState` directly from UI components. Even the initiating
window waits for the broadcast so all windows share the same snapshot.

## Window Labels and Lifecycle

Window labels are constants in `window/mod.rs` and mirrored in
`src/constants/windows.ts`. Current labels are:

- `clipboard`
- `preference`
- `clipboard-preview`
- `onboarding`
- `update`
- `context-menu`
- `context-submenu`

When a known window label is displayed in diagnostics UI, map it through
`src/locales/*/preferences.json` instead of rendering the raw internal label.
Keep raw-label fallback only for unknown or unregistered labels so diagnostics
remain useful.

Use `window::show_window`, `hide_window`, and `toggle_window` instead of direct
webview calls. These functions restore/save geometry, apply clipboard-window
position settings, emit `window://visibility`, update lifecycle state, and
handle platform-specific show/hide.

`window/lifecycle/mod.rs` tracks phases such as `ready`, `visible`,
`hiddenWarm`, `dormant`, `destroyPending`, and `destroyed`. Destroyable windows
use descriptors and can rebuild after idle destruction. Frontend windows report
readiness through `notify_window_ready` once `settingsReady` has resolved.

### Scenario: Clipboard Window Label Contract

#### 1. Scope / Trigger

Changing the primary clipboard-history window label touches Tauri config, Rust
window constants, frontend constants, commands, events, lifecycle diagnostics,
and preview payloads. Treat it as a cross-layer contract change.

#### 2. Signatures

- Runtime label:
  - Rust `window::CLIPBOARD_WINDOW_LABEL = "clipboard"`
  - TypeScript `WINDOW_LABEL.CLIPBOARD = "clipboard"`
  - Tauri config window label `"clipboard"`
- Commands:
  - `set_clipboard_window_pinned(pinned: bool) -> ()`
  - `set_clipboard_window_auto_hide_suspended(suspended: bool) -> ()`
- Preview payload:
  - `ClipboardPreviewState.clipboardWindow: PreviewClipboardWindowRect | null`

#### 3. Contracts

- User-visible copy should say clipboard window, not main window.
- User-visible names for concrete window entities should include the window
  suffix, for example `剪贴板窗口` / `Clipboard Window` and
  `偏好设置窗口` / `Preferences Window`.
- Do not use raw `"clipboard"` in React components; compare with
  `WINDOW_LABEL.CLIPBOARD`.
- Keep platform terms such as Rust `main()`, `run_on_main_thread`, and Cocoa
  `can_become_main_window` unchanged.

#### 4. Validation & Error Matrix

- Stale `"main"` label in a consumer -> lifecycle or emit target is ignored.
- Stale command name -> frontend invoke fails with an unknown command.
- Stale preview field name -> preview layout loses clipboard-window geometry.

#### 5. Good/Base/Bad Cases

- Good: tray, shortcut, onboarding, context menu, preview, and copy-after-hide
  all target `CLIPBOARD_WINDOW_LABEL` / `WINDOW_LABEL.CLIPBOARD`.
- Base: secondary windows still use their existing labels and lifecycle rules.
- Bad: component code checks `event.payload.label === "main"` directly.

#### 6. Tests Required

- `rg` verifies no stale `MAIN_WINDOW_LABEL`, `WINDOW_LABEL.MAIN`, raw
  `"main"` label, or old command names remain outside platform terminology.
- Frontend: `pnpm lint` and `pnpm tsc`.
- Backend: `cargo fmt`, `cargo clippy -- -D warnings`, and `cargo test`.
- Manual desktop checks for clipboard show/hide/toggle, pin, auto-hide,
  preview, tray click, context menu dispatch, onboarding completion, and
  shortcuts.

#### 7. Wrong vs Correct

Wrong:

```tsx
if (event.payload.label !== "main") return;
```

Correct:

```tsx
if (event.payload.label !== WINDOW_LABEL.CLIPBOARD) return;
```

## Pending-Slot Pattern

When an event targets a destroyable preference window, use a Rust pending slot
if the window may not exist yet:

- Preference setting highlight: `window::open_preference_with_highlight` and
  `take_pending_preference_highlight`.
- Backup file receive: `backup::emit_received_backup` and
  `take_pending_backup`.

This avoids losing events while a WebView is being rebuilt and before React has
mounted listeners.

## macOS Window Rules

The clipboard window becomes an NSPanel in `window/macos.rs`. It can become key
without activating the app. Hidden constraints:

- The tauri-nspanel plugin must be registered before `to_panel`.
- Clipboard panel show is delayed briefly, then executed on the main thread.
- Visibility and lifecycle events for the clipboard panel are emitted after the panel
  actually shows.
- Pinned-window paste must resign key before simulated paste, then restore key
  after the target app consumes the keystroke.

Changing clipboard-window behavior on macOS requires manual testing of show/hide,
keyboard navigation, auto-hide, pinned paste, and Spaces behavior.

## Windows Window Rules

The clipboard window is non-focusable on Windows. `window/windows.rs` enables low-level
keyboard and mouse hooks while the clipboard window is visible:

- `keyboard/windows.rs` emits `keyboard://nav` for navigation keys and selected
  Ctrl shortcuts.
- `mouse/windows.rs` hides the clipboard window on outside click when auto-hide is
  allowed.
- Hiding the clipboard window disables hooks and hides context menus.

React code uses `useKeyboardEvent` so components do not need to know whether the
event came from the browser or Rust.

## Platform-Specific Tauri Capabilities

Platform-specific plugin permissions must live in their own capability file with
an explicit `platforms` guard. Keep `src-tauri/capabilities/default.json`
limited to permissions valid on both macOS and Windows; otherwise Windows
capability validation can reject macOS-only identifiers before runtime cfgs take
effect.

Wrong:

```json
{
  "identifier": "default",
  "permissions": ["core:default", "macos-permissions:default"],
  "windows": ["*"]
}
```

Correct:

```json
{
  "identifier": "macos-permissions",
  "permissions": ["macos-permissions:default"],
  "platforms": ["macOS"],
  "windows": ["*"]
}
```

## Native Integration Side Effects

Settings changes that affect OS state must be applied in Rust or through the
existing command layer:

- Global shortcuts: `shortcut::apply`, `suspend`, and `resume`.
- Tray menu and language: `tray::apply`.
- Autostart: `autostart::set_enabled`.
- Dock/taskbar icon: `window::show_taskbar_icon`.

Failures in side-effect refresh after a successful settings write are logged and
do not roll back the persisted settings unless the called command itself is the
direct user action, such as `set_autostart`.

## Scenario: Auto Updater Window

### 1. Scope / Trigger

Adding or changing automatic update behavior touches Tauri plugin setup, Rust
commands, stored update settings, Tauri window lifecycle, TypeScript command
mirrors, and localized UI strings. Treat it as a cross-layer contract change,
not a frontend-only preference action.

### 2. Signatures

- Window label:
  - Rust `window::UPDATE_WINDOW_LABEL = "update"`
  - TypeScript `WINDOW_LABEL.UPDATE = "update"`
- Commands:
  - `get_update_status() -> AppUpdateStatus`
  - `check_for_updates() -> AppUpdateStatus`
  - `download_update(version: String) -> UpdateMetadata`
  - `install_update(version: String) -> ()`
  - `skip_update_version(version: String) -> AppUpdateStatus`
  - `open_update_window() -> ()`
- Event:
  - `update://progress` with `{ downloaded, total, progress }`
- Settings fields:
  - `update.autoCheck: bool`
  - `update.includeBeta: bool`
  - `update.frequency: daily | weekly | monthly`
  - `update.lastCheckedAt: Option<String>`
  - `update.skippedVersion: Option<String>`

### 3. Contracts

- Rust owns update checks, signature verification, downloaded bytes, install,
  auto-check throttling, skipped-version persistence, and automatic update
  window display.
- `update::schedule_auto_check` must run for the whole app session: it performs
  the initial delayed background check, then computes the next due time from
  `update.lastCheckedAt + update.frequency` so the configured cadence is honored
  even when EcoPaste stays open for days. The scheduler may cap long sleeps to
  refresh settings, but network checks must still pass through Rust-side
  `should_auto_check` throttling.
- `update.skippedVersion` suppresses that exact version for both manual and
  automatic checks; a newer different version is shown normally.
- React renders `AppUpdateStatus` and sends user intent through command wrappers
  only. Do not call updater plugin APIs directly from React components.
- Stable endpoint default:
  `https://github.com/EcoPasteHub/EcoPaste/releases/latest/download/latest.json`.
- Optional environment overrides:
  - `ECOPASTE_UPDATE_ENDPOINT`: stable channel endpoint.
  - `ECOPASTE_UPDATE_BETA_ENDPOINT`: beta channel endpoint; checked before the
    stable endpoint when `includeBeta` is true.
  - `TAURI_UPDATER_PUBLIC_KEY` or `TAURI_SIGNING_PUBLIC_KEY`: public updater
    verification key injected into the plugin at runtime.
  - `TAURI_SIGNING_PRIVATE_KEY`: build-time private key used by Tauri bundling
    to sign artifacts; never read it from app runtime code.

### 4. Validation & Error Matrix

- Invalid endpoint URL -> command error `update endpoint is invalid: ...`.
- No configured endpoint or updater build failure -> command error from Rust
  updater setup.
- No pending update when downloading/installing -> `no update is ready`.
- Version mismatch between UI request and Rust pending update ->
  `the selected update is no longer current`.
- Install before successful download -> `update is not downloaded`.
- Signature mismatch -> propagated from `tauri-plugin-updater` download
  verification; frontend should show the command error and remain in error
  state.

### 5. Good/Base/Bad Cases

- Good: user opens About -> Check for Updates; Rust opens the `update` window,
  checks stable/beta endpoint according to settings, downloads through Rust,
  emits progress, verifies signature, then installs.
- Base: no update is available; Rust clears pending update state and the window
  renders the latest-version state.
- Bad: React stores downloaded bytes, performs signature checks, or decides
  auto-check frequency locally.

### 6. Tests Required

- Backend checks: `cargo clippy -- -D warnings` and `cargo test`.
- Auto-check scheduling changes must verify that the background task loops after
  the first check and still delegates frequency/auto-check gating to
  `should_auto_check`.
- Frontend checks: `pnpm tsc` and `pnpm lint`.
- Packaging/build check: `pnpm build` for route and locale integration.
- Manual update-server checks when release artifacts exist:
  - mock endpoint returns newer signed version -> update found and install path
    starts.
  - mock endpoint returns bad signature -> download refuses install.
  - `includeBeta=false` -> stable endpoint only.
  - `includeBeta=true` -> beta endpoint is attempted before stable fallback.

### 7. Wrong vs Correct

Wrong:

```tsx
import { check } from "@tauri-apps/plugin-updater";

await check();
```

Correct:

```tsx
import { checkForUpdates } from "@/commands";

const status = await checkForUpdates();
```

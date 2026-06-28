# State Management

## Ownership

Valtio stores frontend UI state and mirrors, not durable business data.

Current stores:

- `settingsState`: mirror of Rust `SettingsStore`, updated only from
  `settings://updated`.
- `clipboardViewState`: transient filters for the clipboard window.
- `clipboardStatsState`: small shared display value for the current filtered
  total.
- `sourceAppsState` and `windowLifecycleState`: UI mirrors of command/event
  results.

Clipboard history, groups, apps, backups, settings persistence, and storage
usage remain backend-owned.

## Settings Flow

`src/stores/settings.ts` documents the one-way flow:

```text
component
  -> updateSettings(patch)
  -> Rust SettingsStore persists
  -> Rust emits settings://updated
  -> every WebView listener Object.assigns settingsState
```

Do not optimistically mutate `settingsState` in components or services. The
caller can use the returned snapshot from `updateSettings` if it needs immediate
control flow, but rendering waits for the event mirror.

## Clipboard View State

`clipboardViewState` contains only query/filter state used by the clipboard
window: `category`, `keyword`, `groupId`, and `range`. Keep pagination inside
`useClipboardItems`; do not add `limit` or `offset` to the store.

Be careful with names. `List` maps store fields into `ClipboardItemQuery`, so a
new UI field with the same name as a backend query field can accidentally become
a filter. The store comment calls out `pinned` as a known footgun.

## Server Data

Use command hooks or local component state for command results:

- `useClipboardItems` owns the current virtualized page list.
- Preference panels read the current settings snapshot and commit patches.
- Backup/storage modals fetch their own inspection or usage state.

Promote data to a store only when multiple distant components need the same
small UI mirror and Rust remains the source of truth.

## Derived State

Prefer deriving display state from snapshots during render. Use `useEffect` only
when another store or an external system must be synchronized, such as updating
`clipboardStatsState.total` from `data.total`.

Do not cache derived copies of `ClipboardItem` arrays in stores. The list already
handles deferred reloads and mutation through the data hook.

# Code Reuse Thinking Guide

## Search First

Before adding a helper, constant, command wrapper, setting, action, or locale
key, search for the existing pattern:

```bash
rg "clipboard://updated" src src-tauri/src
rg "open_preference_with_highlight" src src-tauri/src
rg "itemActions" src src-tauri/src
```

Use `rg --files` to find nearby modules quickly. Prefer extending the existing
module that already owns the concept.

## Existing Single Sources

| Concept | Source |
| --- | --- |
| Tauri command names | Rust command function names, `src/constants/commands.ts`, `src/commands/index.ts` |
| Tauri event names | Rust constants near emitters, `src/constants/events.ts` |
| Window labels | `src-tauri/src/window/mod.rs`, `src/constants/windows.ts` |
| Clipboard item actions | `db::models::ClipboardAction`, `commands::clipboard::compute_available_actions`, `src/constants/itemActions.ts` |
| Capture kinds | `settings::CaptureKind`, `src/constants/captureKinds.ts` |
| Settings defaults | `src-tauri/src/settings/model.rs` |
| Settings persistence/validation | `src-tauri/src/settings/store.rs` |
| App data roots | `src-tauri/src/core/paths.rs` |
| Command errors/toasts | `src/commands/index.ts`, `src/locales/*/commands.json` |
| Conditional classes | `src/utils/cn.ts` |
| Platform/window checks | `src/utils/is.ts` |

## When Adding a Setting

Check all mirrors instead of adding local one-offs:

- Rust model/defaults in `settings/model.rs`.
- Rust validation in `settings/store.rs` if fields interact.
- Rust side effects in `commands/settings.rs`, `tray`, `shortcut`, `window`, or
  `autostart` when needed.
- TypeScript mirror in `src/types/settings.ts`.
- Preference schema/control in `src/pages/Preference/config/preferenceSchema.ts`.
- Commit behavior in `Preference/services/preferenceSettings.ts` for immediate
  OS side effects.
- zh-CN and en-US locale JSON.

## When Adding a Clipboard Field

Check the whole response chain:

- Database schema and `db::models::ClipboardItem`.
- Repository `SELECT` aliases and `INSERT`/`UPDATE` binds.
- Command enrichment in `commands/clipboard.rs`.
- TypeScript mirror in `src/types/clipboard.ts`.
- Card/preview components that should render the field.
- Tests constructing `ClipboardItem` literals.

## When to Abstract

Abstract when a pattern is shared by multiple real call sites and the abstraction
preserves ownership. Good examples are `useTauriListen`, `cn`, command `call<T>`,
and `db::test_support::memory_pool`.

Do not abstract across boundaries in a way that hides ownership. A React helper
that classifies clipboard content duplicates Rust. A Rust helper that returns
Ant Design-specific labels leaks UI concerns.

## Common Drift Patterns

- Adding a Rust command but forgetting `src/constants/commands.ts` or the wrapper.
- Adding an event string in Rust but listening to a raw string in React.
- Updating a settings enum in Rust but leaving TypeScript unions or locale
  option labels stale.
- Adding SQL fields but missing `LIST_SELECT_ITEM`, `SELECT_ITEM`, or test model
  literals.
- Adding a user-visible feature without updating the release changelog when it
  changes visible behavior relative to the old EcoPaste app before the next
  release.

# Type Safety

## Rust Contract Mirrors

Rust serde models use `camelCase` at the IPC boundary. TypeScript mirrors live
under `src/types/` and should name fields exactly as the frontend receives them.

Important mirrors:

- `src/types/clipboard.ts` mirrors `db::models::ClipboardItem`,
  `ClipboardAction`, query types, groups, apps, and page results.
- `src/types/settings.ts` mirrors `settings/model.rs`.
- Additional command-specific response types live near the command wrapper in
  `src/commands/index.ts` when they are not broadly shared.

When adding a Rust enum or field, update the TypeScript union/interface in the
same change. Do not widen to `string` when Rust has a closed enum.

## Command Wrapper Types

`src/commands/index.ts` should type every wrapper's return value and argument
shape. The generic `call<T>` wrapper is the only place that handles arbitrary
invoke errors.

Callers should not cast command results. If the wrapper type is insufficient,
fix the wrapper or shared type.

## Settings Patches

Preference controls build `SettingsPatch` values from schema paths in
`src/pages/Preference/services/preferenceSettings.ts`. Arrays are intentionally
replaced as a whole, matching Rust `deep_merge`.

For complex controls such as sortable checkbox trees, save both the selected
values and the complete order path. This preserves disabled item order for the
next edit session.

## Runtime Validation Boundary

Rust validates external or security-sensitive values:

- Image/icon file names in `commands::clipboard::validate_image_file_name`.
- CSS colors in `clipboard::sanitize_css_color`.
- Settings patch shape and duplicate global shortcuts in `SettingsStore`.
- Storage targets in `core::paths::validate_storage_target`.
- Backup passwords and container headers in `backup/mod.rs`.

Frontend controls can guide user input, but they should not be the only
protection for persisted data, filesystem paths, CSS injection, or OS actions.

## Constants and Literals

Use centralized constants for command names, events, windows, capture kinds,
item actions, and URLs. If a string crosses Rust and TypeScript, add or update a
constant on both sides.

Avoid writing raw event names or command names in components. `TAURI_COMMAND`
belongs to `src/commands/index.ts`; components import wrapper functions.

## Avoiding Type Erosion

- Do not add `any` for command payloads; use `unknown` plus narrowing or define
  an interface.
- Keep optional fields optional when Rust uses `skip_serializing_if`.
- Use discriminated unions such as `ClipboardKind` and `ClipboardSubKind` to
  drive rendering.
- Keep `ClipboardItemQuery` aligned with Rust defaults. Optional fields should
  be omitted with `void 0`, not sent as unrelated sentinel values.

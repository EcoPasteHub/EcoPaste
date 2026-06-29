# Fix Windows Tauri Capability Permissions

## Goal

Windows builds and editor validation must not load macOS-only Tauri permission
identifiers from the shared default capability. macOS should keep access to the
permission-check/request commands used by onboarding.

## What I Already Know

- The screenshot shows `src-tauri/capabilities/default.json` rejecting
  `macos-permissions:default` on Windows.
- `tauri_plugin_macos_permissions::init()` is registered only under
  `#[cfg(target_os = "macos")]` in `src-tauri/src/lib.rs`.
- Tauri capability files support a `platforms` field with values including
  `macOS` and `windows`.

## Requirements

- Remove `macos-permissions:default` from any all-platform capability.
- Keep macOS permission commands available to app windows on macOS.
- Avoid adding Linux support or changing runtime behavior unrelated to Tauri ACL
  configuration.
- Keep generated capability schema/cache aligned if the repo tracks it.

## Acceptance Criteria

- [x] Windows-targeted capability validation no longer references
  `macos-permissions:default`.
- [x] macOS-targeted capability configuration still grants
  `macos-permissions:default`.
- [x] Project checks run for the touched configuration.

## Out of Scope

- Onboarding UI changes.
- Rust plugin registration changes.
- New permission UX or platform feature work.

## Technical Notes

- Relevant files:
  - `src-tauri/capabilities/default.json`
  - `src-tauri/gen/schemas/capabilities.json`
  - `src-tauri/src/lib.rs`
- Relevant specs:
  - `.trellis/spec/backend/index.md`
  - `.trellis/spec/backend/settings-window-platform.md`
  - `.trellis/spec/guides/index.md`

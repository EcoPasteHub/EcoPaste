# Implementation Plan

## Checklist

- [x] Add Rust admin module
  - [x] `is_running_as_admin`
  - [x] scheduled-task query/path validation/create/delete/run
  - [x] UAC `ShellExecuteW("runas")` fallback
  - [x] restart argument builder with internal marker filtering
  - [x] early startup auto-elevation helper
- [x] Add `general.run_as_admin` to settings
  - [x] Rust model/defaults
  - [x] TypeScript mirror
  - [x] settings missing-field tests
- [x] Add command boundary
  - [x] Rust `commands/admin.rs`
  - [x] register in `commands/mod.rs` and `lib.rs`
  - [x] TypeScript constants and wrappers
  - [x] command labels in both locales
- [x] Wire startup
  - [x] call admin startup check before normal Tauri builder setup on Windows
  - [x] sync scheduled task when already elevated and configured
  - [x] exit unelevated process only when relaunch succeeds
- [x] Wire onboarding UI
  - [x] replace Windows `pendingIntegration` status with command-backed status
  - [x] authorization action persists intent and restarts as admin
  - [x] keep macOS permission flow unchanged
  - [x] update onboarding locale copy and statuses
- [x] Update release note
  - [x] add concise user-visible release note entry

## Validation

- [x] `cd src-tauri; cargo fmt`
- [x] `cd src-tauri; cargo test`
- [x] `cd src-tauri; cargo clippy -- -D warnings`
- [x] `pnpm lint`
- [x] `pnpm tsc`
- [ ] Manual Windows validation:
  - [ ] fresh non-elevated launch shows admin permission actionable
  - [ ] Grant Access shows UAC, launches elevated process, exits old process
  - [ ] elevated process reopens onboarding and shows granted
  - [ ] cancelling UAC leaves app open and shows an error
  - [ ] subsequent release launch with `runAsAdmin=true` auto-elevates before normal setup

Automated checks were rerun after the scheduled-task argument policy and onboarding switch confirmation flow were tightened. Manual UAC and Task Scheduler behavior still needs an interactive Windows desktop session.

## Risky Files

- `src-tauri/src/lib.rs`: startup order and command registration.
- `src-tauri/src/settings/model.rs`: settings contract.
- `src/pages/Onboarding/components/PermissionsStep.tsx`: cross-platform permission flow.
- `src/commands/index.ts`: command wrapper surface and toast behavior.

## Rollback

Remove the admin module, command registrations, settings field, frontend command wrappers, and onboarding Windows card changes. Since the app is unreleased, no persisted migration rollback is needed beyond removing `general.runAsAdmin` from settings defaults and mirrors.

# Onboarding Administrator Launch Flow

## Goal

Bring EcoPaste's Windows onboarding permission step onto a real "Run as Administrator" flow modeled after the implementation in `C:\Users\ayangweb\Downloads\QuickClipboard`, so first-run users can detect current elevation, request elevated restart, and preserve the existing onboarding and app startup behavior.

This is a cross-layer Windows system feature. Rust owns detection, launch/restart, startup decisions, and persisted settings; React only renders status and sends user intent through command wrappers.

## Background And Evidence

- QuickClipboard implements administrator detection in `src-tauri/src/services/system/elevate.rs` using `OpenProcessToken` plus `GetTokenInformation(TokenElevation)` on Windows, returning false on non-Windows.
- QuickClipboard persists a `run_as_admin` setting and checks it at startup in `src-tauri/src/lib.rs` before normal Tauri setup. In release builds, if `run_as_admin` is enabled and the process is not elevated, it tries to launch an elevated instance and exits the unelevated one.
- QuickClipboard's elevation flow first tries an existing scheduled task named `QuickClipboardAdmin` when the task exists and its target path matches the current executable. If that fails, it falls back to `ShellExecuteW` with the `runas` verb to trigger UAC.
- QuickClipboard exposes frontend commands for setting `run_as_admin`, checking current elevation, checking scheduled-task readiness, and restarting as admin. Its settings UI saves the setting and optionally calls restart.
- EcoPaste currently has no persisted `general.runAsAdmin` field, no admin command wrapper, and no admin module. The current Windows onboarding permission card in `src/pages/Onboarding/components/PermissionsStep.tsx` uses `pendingIntegration`.
- EcoPaste's onboarding window opens automatically when `settings.onboarding.completed` is false in `src-tauri/src/lib.rs`, and `show_default_foreground_window` reopens onboarding while setup is incomplete.
- EcoPaste centralizes command names in `src/constants/commands.ts`, command wrappers in `src/commands/index.ts`, and Rust commands through `src-tauri/src/commands/mod.rs` plus `tauri::generate_handler!`.
- Relevant project constraints from `AGENTS.md` and `.trellis/spec/`: support macOS plus Windows only; platform behavior must be Rust-first; user-visible new capability requires a changelog note; command names and TypeScript mirrors must stay centralized.

## Requirements

- R1. On Windows, EcoPaste must expose a real administrator permission state to onboarding:
  - current process elevated -> granted
  - current process not elevated -> denied/actionable
  - non-Windows -> not required or existing macOS permission behavior
- R2. Onboarding must let a Windows user request administrator launch from the permissions step, not from a placeholder copy path.
- R3. Requesting administrator launch must persist the user's intent so future release launches can auto-elevate using the same setting.
- R4. If the current process is not elevated and the user requests administrator launch, EcoPaste must launch an elevated instance and exit the unelevated one only after the new launch request succeeds.
- R5. The elevated restart flow must preserve relevant launch arguments where needed, including file-open backup arguments and auto-launch context where feasible.
- R6. Startup auto-elevation must run early enough to avoid initializing normal windows, tray, database, or clipboard watcher in the unelevated process.
- R7. Windows implementation must follow QuickClipboard's shape where appropriate:
  - detect elevation through the Windows access token
  - prefer a scheduled task when configured and valid
  - fall back to UAC `ShellExecuteW("runas", ...)`
  - hide helper command windows
- R8. macOS onboarding permission behavior must remain unchanged.
- R9. Existing onboarding steps must keep working: welcome, permissions, shortcuts, ignored apps, legacy import, and finish behavior.
- R10. Settings, command constants, command wrappers, i18n labels, and TypeScript mirrors must be updated together.
- R11. Add a release note entry because this is a user-visible capability relative to the old Rust-first app state.

## Out Of Scope

- Linux support or Linux elevation behavior.
- Broad redesign of onboarding layout, preference schema, or shortcut behavior.
- Changing installer elevation requirements.
- Adding unrelated Windows system capabilities beyond administrator launch.
- Backward compatibility shims for older settings schemas while the app remains `0.6.0-beta.3`.

## Acceptance Criteria

- [ ] On Windows, the onboarding permissions step shows administrator launch as granted when EcoPaste is already elevated.
- [ ] On Windows, the onboarding permissions step shows administrator launch as actionable when EcoPaste is not elevated.
- [ ] Clicking the Windows onboarding authorization action persists `general.runAsAdmin = true`, requests elevated restart, and exits the old process only when the elevated launch request succeeds.
- [ ] On restart after approval, onboarding opens normally and the administrator permission card reports granted.
- [ ] When UAC is cancelled or elevation launch fails, EcoPaste stays open and shows the command-layer error without marking the permission granted.
- [ ] Future release launches with `general.runAsAdmin = true` auto-elevate before normal setup if the process is unelevated.
- [ ] macOS Accessibility and Full Disk Access checks still work as they do today.
- [ ] Existing onboarding completion still hides onboarding and opens the clipboard window.
- [ ] Release notes include a concise user-visible change note.
- [ ] Verification covers `cargo fmt`, focused Rust checks, `pnpm lint`, and `pnpm tsc`; manual Windows validation is listed if it cannot be run in this environment.

## Decisions

- D1. Implement the full QuickClipboard-style flow in this task: scheduled-task reuse/maintenance first, then UAC `runas` fallback. The task should not ship an onboarding-only UAC shortcut that leaves the planned restart path incomplete.

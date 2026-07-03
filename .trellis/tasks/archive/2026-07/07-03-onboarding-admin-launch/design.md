# Design

## Architecture

Add a Windows-focused Rust module owned by native system behavior:

```text
src-tauri/src/admin.rs
  -> detect current elevation
  -> create/delete/query highest-privilege scheduled task
  -> launch elevated instance through task or UAC
  -> preserve restart arguments
```

`settings::General` gains `run_as_admin: bool`, mirrored as `general.runAsAdmin` in TypeScript. This setting records the user's persistent intent, not the current process state.

Commands stay thin:

```text
commands::admin
  get_run_as_admin_status() -> AdminLaunchStatus
  set_run_as_admin(enabled: bool) -> Settings
  restart_as_admin() -> ()
```

The frontend only consumes command wrappers from `@/commands`. `PermissionsStep` renders the Windows card from `AdminLaunchStatus` and calls `setRunAsAdmin(true)` followed by `restartAsAdmin()` when the user authorizes.

## Data Flow

### Onboarding Authorization

```text
User clicks Grant Access
  -> setRunAsAdmin(true)
  -> Rust updates SettingsStore.general.run_as_admin
  -> Rust syncs admin scheduled task when possible
  -> settings://updated
  -> restartAsAdmin()
  -> Rust launches elevated process through scheduled task or UAC
  -> current process exits only if launch succeeds
  -> elevated process starts
  -> startup sees already elevated and continues setup
  -> onboarding window opens because onboarding.completed is still false
```

If launch fails or UAC is cancelled, `restart_as_admin` returns an error and the current process remains open.

### Startup Auto Elevation

Startup must check persisted settings before normal Tauri setup initializes windows, tray, database, clipboard watcher, or hooks:

```text
run()
  -> early admin startup check on Windows
     -> read settings.json using the same data-root logic as SettingsStore
     -> if runAsAdmin=false: continue
     -> if already elevated: sync scheduled task best-effort, continue
     -> if not elevated: launch elevated instance and exit current process on success
  -> build Tauri app
```

Because current settings path resolution depends on Tauri path APIs and storage manifest logic, add a lightweight early settings loader that accepts the app-local-data base path and mirrors `core::paths` enough to find `<app_local_data>/<env>/storage.json` then `<data_root>/config/settings.json`.

### Argument Passing

Elevated relaunch should pass current process arguments except internal anti-loop markers. Add an internal marker such as `--ecopaste-admin-restarted` to prevent accidental repeated relaunch attempts from the same user-triggered restart path.

The scheduled task action should include the current executable and the internal restart marker only. Because `schtasks /Run` cannot supply per-run dynamic arguments, use the task only when the current launch has no external arguments. File-open backup paths, auto-launch markers, and other dynamic arguments must fall back to UAC `runas`, which can preserve the full restart argument list. The task path validity check must validate the current executable path to avoid reusing a stale task after upgrades or moves.

## Contracts

- `Settings.general.runAsAdmin` defaults to `false`.
- `AdminLaunchStatus` returns:
  - `configured: bool`
  - `runningAsAdmin: bool`
  - `taskReady: bool`
- Commands:
  - `get_run_as_admin_status`
  - `set_run_as_admin`
  - `restart_as_admin`
- Command constants and labels must be mirrored in:
  - `src/constants/commands.ts`
  - `src/commands/index.ts`
  - `src/locales/zh-CN/commands.json`
  - `src/locales/en-US/commands.json`

## Platform Behavior

Windows:

- Use Windows token APIs for elevation detection.
- Use `schtasks` with `CREATE_NO_WINDOW` for task query/create/delete/run.
- Create task with `/RL HIGHEST`, `/SC ONCE`, `/ST 00:00`, `/F`.
- Prefer valid scheduled task launch; fallback to `ShellExecuteW` with `runas`.
- Hide helper command windows.

macOS:

- Admin commands should return harmless non-Windows status or a clear error only if the command is inapplicable.
- Existing Accessibility and Full Disk Access onboarding checks remain unchanged.

## Compatibility

The app is still `0.6.0-beta.3`; missing `runAsAdmin` falls back through serde defaults. No compatibility layer or migration file is required.

## Risks

- Scheduled task command-line quoting must preserve paths with spaces.
- Startup early settings path resolution must match the runtime settings path, including custom storage.
- UAC cancellation must not mark the permission granted in UI; the persisted intent can remain enabled, but the current process state remains denied until elevated.
- Manual Windows validation is required because UAC and scheduled tasks cannot be fully proven through type checks.

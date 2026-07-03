# Contributing to EcoPaste

English | [简体中文](./CONTRIBUTING.zh-CN.md)

Thank you for helping improve EcoPaste. This guide covers the project scope,
development setup, architecture boundaries, quality checks, and contribution
expectations for code and documentation changes.

## Project Status

EcoPaste has entered its stable release line. Future changes should evolve the
current app directly and include migrations or upgrade handling for released
user data when storage, settings, or database contracts change.

## Platform Scope

EcoPaste supports macOS and Windows only.

Linux is not supported, and new code, dependencies, builds, and documentation
should stay focused on macOS and Windows.

## Before You Change Code

Read the [AGENTS.md](./AGENTS.md) first. It is the source of truth for architecture,
platform scope, coding conventions, and quality expectations in this repository.

Respect the current working tree. Do not overwrite or roll back changes you did
not make.

Use single-line Conventional Commits for commit messages, such as `feat:`,
`fix:`, `refactor:`, or `docs:`.

## Architecture

EcoPaste uses a Rust-first Tauri architecture:

- `src-tauri/src/clipboard/` owns clipboard capture, content detection,
  writeback, source apps, resource storage, and loop suppression.
- `src-tauri/src/db/` owns SQLite repositories, models, migrations, and FTS
  search.
- `src-tauri/src/settings/`, `window/`, `shortcut/`, `tray/`, `menu/`,
  `autostart/`, and `backup/` own native behavior and persistent app state.
- `src/` contains the React UI, Ant Design components, UnoCSS styling, Valtio
  UI/settings mirrors, i18n resources, and typed Tauri command wrappers.

The frontend calls Rust through Tauri commands and receives refresh signals
through namespaced events such as `clipboard://updated`, `settings://updated`,
and `window://visibility`.

## Tech Stack

| Area          | Stack                                          |
| ------------- | ---------------------------------------------- |
| Desktop shell | Tauri v2                                       |
| Frontend      | React 19, Ant Design 6, UnoCSS `presetWind4`   |
| State         | Valtio for UI state and settings mirrors       |
| Backend       | Rust, sqlx, SQLite                             |
| Build         | Vite, pnpm                                     |
| Quality       | Biome, TypeScript, rustfmt, clippy, cargo test |

## Getting Started

### Prerequisites

- macOS or Windows.
- Node.js 20 or newer.
- pnpm 10 or newer.
- Rust toolchain from `rust-toolchain.toml` (`1.96.0`, with `rustfmt` and
  `clippy`).
- Native dependencies required by Tauri v2. See the
  [Tauri prerequisites](https://tauri.app/start/prerequisites/) for your
  operating system.
- Trellis workflow docs: [Trellis documentation](https://docs.trytrellis.app).

### Install

```bash
pnpm install
```

### Run in Development

```bash
pnpm tauri dev
```

### Build

```bash
pnpm tauri build
```

## Quality Checks

Frontend:

```bash
pnpm lint
pnpm tsc
```

Rust:

```bash
cd src-tauri
cargo fmt
cargo clippy -- -D warnings
cargo test
```

Format frontend files:

```bash
pnpm format
```

## Repository Layout

```text
src-tauri/
  src/
    commands/    # Tauri command entry points
    clipboard/   # clipboard read/write, capture, detection, storage
    db/          # SQLite repositories, models, migrations
    settings/    # settings model and persistence
    window/      # window state, positioning, lifecycle
    shortcut/    # global shortcuts
    tray/        # tray menu
    menu/        # item context menus
    backup/      # backup import/export
    i18n/        # Rust-side user-visible text
  migrations/
src/
  commands/      # typed Tauri invoke wrappers
  components/    # shared React components
  constants/     # mirrored cross-layer constants
  hooks/         # shared hooks
  locales/       # zh-CN and en-US translations
  pages/         # Clipboard, Preference, Preview, ContextMenu
  stores/        # Valtio UI state and settings mirrors
  types/         # TypeScript contract mirrors
```

## Contribution Checklist

- Keep business logic, native capabilities, database access, storage, settings
  persistence, and platform integration in Rust unless the existing architecture
  clearly says otherwise.
- Keep React focused on rendering, interaction, UI state, frontend i18n, and
  previews.
- Mirror cross-layer constants in both Rust and `src/constants/` when command
  names, event names, channels, or storage keys are reused across layers.
- Add migrations for released schema changes. Do not edit released migrations in
  place.
- Update both `zh-CN` and `en-US` locale resources for user-facing frontend
  text.
- Use Rust-side `i18n/` for short user-visible native strings such as tray
  labels, native menus, and command-returned toast text.
- Run focused checks for the files you changed, and broaden validation when the
  change touches shared behavior or cross-layer contracts.

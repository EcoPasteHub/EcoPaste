<div align="center">
  <img src="./public/logo.png" alt="EcoPaste" width="96" height="96" />

  # EcoPaste

  **A local-first clipboard manager for macOS and Windows.**

  English | [简体中文](./README.zh-CN.md)

  <br />

  <img alt="Tauri v2" src="https://img.shields.io/badge/Tauri-v2-24c8db?style=flat-square" />
  <img alt="Rust first" src="https://img.shields.io/badge/Rust-first-b7410e?style=flat-square" />
  <img alt="React 19" src="https://img.shields.io/badge/React-19-61dafb?style=flat-square" />
  <img alt="macOS" src="https://img.shields.io/badge/macOS-supported-000000?style=flat-square&logo=apple&logoColor=white" />
  <img alt="Windows" src="https://img.shields.io/badge/Windows-supported-0078d4?style=flat-square&logo=windows&logoColor=white" />
</div>

## About

EcoPaste is an open-source desktop clipboard manager built with a Rust-first Tauri architecture: durable behavior lives in Rust, while the React frontend focuses on rendering and interaction.

The app is designed to be fast, lightweight, and maintainable, with local storage, SQLite search, native shortcuts, tray integration, backup support, and a focused cross-platform surface for macOS and Windows.

## Project Status

EcoPaste has entered its stable release line. Future changes should evolve the current app directly and include migrations or upgrade handling for released user data when storage, settings, or database contracts change.

## Platform Scope

EcoPaste supports macOS and Windows only.

Linux is not supported, and new code, dependencies, builds, and documentation should stay focused on macOS and Windows.

## Features

- Capture clipboard history for plain text, HTML, RTF, images, files, and folders.
- Search clipboard content and notes with SQLite FTS5.
- Filter history by source application and content type.
- Protect sensitive content by skipping high-confidence secrets such as private keys, service tokens, AWS keys, and JWTs.
- Preview text, images, and files in a dedicated preview window.
- Paste, copy, copy as plain text, reveal files, open links, add notes, pin, favorite, delete, and drag items out to other apps.
- Organize history with favorites, pinned items, notes, custom groups, and configurable item actions.
- Tune capture order, size limits, retention, display density, list sorting, and window behavior.
- Export and import `.ecopastebak` backups, including encrypted backup containers.
- Keep clipboard data, resources, and settings local to your machine.

## Architecture

EcoPaste uses a Rust-first Tauri architecture:

- `src-tauri/src/clipboard/` owns clipboard capture, content detection, writeback, source apps, resource storage, and loop suppression.
- `src-tauri/src/db/` owns SQLite repositories, models, migrations, and FTS search.
- `src-tauri/src/settings/`, `window/`, `shortcut/`, `tray/`, `menu/`, `autostart/`, and `backup/` own native behavior and persistent app state.
- `src/` contains the React UI, Ant Design components, UnoCSS styling, Valtio UI/settings mirrors, i18n resources, and typed Tauri command wrappers.

The frontend calls Rust through Tauri commands and receives refresh signals through namespaced events such as `clipboard://updated`, `settings://updated`, and `window://visibility`.

## Tech Stack

| Area | Stack |
| --- | --- |
| Desktop shell | Tauri v2 |
| Frontend | React 19, Ant Design 6, UnoCSS `presetWind4` |
| State | Valtio for UI state and settings mirrors |
| Backend | Rust, sqlx, SQLite |
| Build | Vite, pnpm |
| Quality | Biome, TypeScript, rustfmt, clippy, cargo test |

## Getting Started

### Prerequisites

- macOS or Windows.
- Node.js 20 or newer.
- pnpm 10 or newer.
- Rust toolchain from `rust-toolchain.toml` (`1.96.0`, with `rustfmt` and `clippy`).
- Native dependencies required by Tauri v2. See the [Tauri prerequisites](https://tauri.app/start/prerequisites/) for your operating system.

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

## Contributing

Read [AGENTS.md](./AGENTS.md) before changing code. It is the source of truth for architecture, platform scope, coding conventions, and quality expectations.

## License

EcoPaste is licensed under the [Apache License 2.0](./LICENSE).

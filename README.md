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

EcoPaste is an open-source desktop clipboard manager. This repository is the Rust-first refactor of EcoPaste: durable behavior lives in Rust, while the React frontend focuses on rendering and interaction.

The rewrite is designed for a faster, lighter, and more maintainable app with local storage, SQLite search, native shortcuts, tray integration, backup support, and a focused cross-platform surface for macOS and Windows.

## Project Status

This repository is currently a beta refactor (`0.6.0-beta.3`). It is not a stable release channel yet.

Before trying this version, back up important data from any older EcoPaste installation. The refactor changes the runtime architecture, settings model, storage layout, and database schema, so legacy data compatibility is not guaranteed during the beta period.

## Platform Scope

The Rust-first refactor supports macOS and Windows only.

Linux support from the legacy EcoPaste app has been dropped in this refactor, and there are no current plans to support Linux again. Please use the legacy release line if you need Linux support.

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

| Area          | Stack                                          |
| ------------- | ---------------------------------------------- |
| Desktop shell | Tauri v2                                       |
| Frontend      | React 19, Ant Design 6, UnoCSS `presetWind4`   |
| State         | Valtio for UI state and settings mirrors       |
| Backend       | Rust, sqlx, SQLite                             |
| Build         | Vite, pnpm                                     |
| Quality       | Biome, TypeScript, rustfmt, clippy, cargo test |

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

Read [CONTRIBUTING](./CONTRIBUTING.md) before changing code. It describes the ordinary development flow, the required Trellis workflow for AI-assisted development, and the first-time AI contributor setup.

For user-visible feature changes in the next release, update [RELEASE-NEXT.md](./RELEASE-NEXT.md). Keep documentation aligned with the current beta status and supported platforms.

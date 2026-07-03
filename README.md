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

## Contributing

Development setup, architecture notes, quality checks, and contribution
expectations live in the [contribution guide](./CONTRIBUTING.md).

## License

EcoPaste is licensed under the [Apache License 2.0](./LICENSE).

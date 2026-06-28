# EcoPaste Specification Index

These specs capture how EcoPaste is built today. Read this index together with
`AGENTS.md`; when they overlap, `AGENTS.md` is the project-level authority and
these files add source-backed detail.

## Project Shape

EcoPaste is a Tauri v2 desktop app for macOS and Windows. Product behavior is
Rust-first: clipboard capture and writeback, database access, settings
persistence, window positioning, shortcuts, tray, autostart, backup, and storage
live in `src-tauri/src/`. The React app in `src/` renders UI, mirrors settings,
handles ordinary interactions, and calls Rust through typed command wrappers.

## Spec Directories

| Directory | Scope |
| --- | --- |
| [backend](./backend/index.md) | Rust/Tauri modules, SQLite, clipboard pipeline, settings, storage, platform behavior |
| [frontend](./frontend/index.md) | React, Ant Design, UnoCSS, Valtio mirrors, command wrappers, i18n, UI quality |
| [guides](./guides/index.md) | Cross-layer and code-reuse checklists for changes that span modules |

## Before Editing

- Identify which layer owns the behavior. If it touches system clipboard,
  persistence, content detection, OS integration, or stored settings, start in
  Rust.
- Search for existing constants and contracts before adding strings. Command
  names are mirrored by `src-tauri/src/lib.rs` and `src/constants/commands.ts`;
  event names are mirrored by Rust constants and `src/constants/events.ts`.
- Keep frontend business data as command results, not as a local database copy.
  `src/stores/clipboardView.ts` and `src/stores/settings.ts` are examples of UI
  state and settings mirrors only.
- Use macOS and Windows `#[cfg]` branches only. There is no Linux support target
  in this rewrite.

## Verification Baseline

- Frontend: `pnpm lint` and `pnpm tsc` for TypeScript and Biome checks.
- Rust: `cd src-tauri && cargo fmt && cargo clippy -- -D warnings && cargo test`.
- UI changes still need a manual run through the affected window. This project
  has no frontend test runner configured yet.

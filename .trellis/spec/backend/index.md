# Backend Development Guidelines

Rust owns EcoPaste's durable and system-facing behavior. Frontend code should
not duplicate backend decisions about clipboard kinds, search, stored settings,
window placement, or OS-specific side effects.

## Guides

| Guide | Read When |
| --- | --- |
| [Architecture](./architecture.md) | Adding Rust modules, changing startup, moving ownership between layers |
| [Commands and Events](./commands-and-events.md) | Adding or changing Tauri commands, emits, command wrappers, or errors |
| [Clipboard Pipeline](./clipboard-pipeline.md) | Clipboard capture, content typing, writeback, source apps, preview/list payloads |
| [Database and Storage](./database-and-storage.md) | SQLite schema, repositories, FTS search, backups, data directory changes |
| [Settings, Windowing, and Platform](./settings-window-platform.md) | Settings model, shortcuts, tray, autostart, window lifecycle, macOS/Windows code |

## Layer Map

| Module | Owns |
| --- | --- |
| `src-tauri/src/commands/` | Tauri command boundary: validate input, read `State`, call lower layers, enrich response models |
| `src-tauri/src/clipboard/` | OS clipboard read/write, content ingestion, source app detection, resource stores, loop suppression |
| `src-tauri/src/db/` | sqlx repositories, SQLite models, migrations, FTS, test database helpers |
| `src-tauri/src/settings/` | Stored settings model, defaults, validation, atomic JSON persistence |
| `src-tauri/src/window/` | Window labels, geometry, lifecycle, preview positioning, platform show/hide |
| `src-tauri/src/shortcut/`, `keyboard/`, `mouse/`, `keystroke/` | OS shortcuts, Windows navigation hooks, paste simulation |
| `src-tauri/src/tray/`, `menu/`, `autostart/` | Native tray/menu/autostart integration |
| `src-tauri/src/backup/`, `commands/storage.rs`, `core/paths.rs` | Backup containers, hot data directory switching, persistent path roots |

## Default Rule

When adding a feature that needs persisted data, content classification, or OS
interaction, design the Rust contract first, then expose a compact command or
event for React to render.

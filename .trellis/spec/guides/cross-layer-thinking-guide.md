# Cross-Layer Thinking Guide

## Rust-First Boundary

For EcoPaste, most cross-layer features should start with a Rust contract. The
frontend renders and sends intent; Rust owns system behavior, storage, content
classification, cleanup, and OS integration.

Ask this first: "Which layer is the source of truth?" If the answer is anything
durable or native, write the Rust model/command/event first and let TypeScript
mirror it.

## Map the Flow

Use this shape before changing a cross-layer feature:

```text
External input
  -> Rust validation/conversion
  -> SQLite/settings/resources/native side effect
  -> command response or event payload
  -> TypeScript mirror
  -> React render/state update
```

Name every boundary and the exact data shape at that boundary.

## Important Local Flows

### Clipboard Item List

```text
OS clipboard
  -> ClipboardPayload
  -> ClipboardItem database row
  -> list_clipboard_items enriched page
  -> ClipboardItem TypeScript interface
  -> Virtuoso cards
```

Watch for: trimmed text content in list queries, `summary` rendering, FTS
keyword behavior, `hasMore` from Rust, file entries prepared by Rust, and
sensitive redaction.

### Settings

```text
Preference control
  -> SettingsPatch
  -> SettingsStore::update
  -> side effects
  -> settings://updated
  -> settingsState mirror
```

Watch for: arrays replacing wholesale, duplicate shortcut validation, side
effects for tray/shortcuts/autostart/taskbar, and both locale files.

### Window Lifecycle

```text
show/hide command or native close
  -> window module
  -> platform show/hide
  -> window://visibility and window://lifecycle
  -> frontend mirrors/deferred work
```

Watch for: macOS delayed NSPanel show, Windows non-focusable keyboard hooks,
destroyable preference/preview windows, pending-slot delivery, and keepalive or
dirty owners before destroy.

### Storage and Backup

```text
user chooses storage/backup action
  -> Rust validates path/container/password
  -> pauses watcher or opens backup payload
  -> replaces pool/settings/resources if needed
  -> emits settings and clipboard refresh events
  -> preference UI updates
```

Watch for: `DatabaseState::close_and_replace`, `core::paths` as the only data
root resolver, pending backup receive when preference is destroyed, and restart
requirements after imported settings.

## Contract Checklist

Before implementation:

- Which Rust type is the source of truth?
- Which TypeScript type mirrors it?
- Are command and event names centralized?
- Does the data need to be stored in SQLite, settings JSON, resources, or only
  local UI state?
- Does the feature behave differently on macOS and Windows?
- Does a hidden/dormant window need deferred handling?

After implementation:

- Search for the old field/name/value with `rg`.
- Run focused Rust tests if the owner is backend logic.
- Run `pnpm lint` and `pnpm tsc` for frontend changes.
- Manually exercise UI paths that depend on native windows, clipboard, drag,
  shortcuts, tray, autostart, or file dialogs.

## Anti-Patterns

- Frontend infers persisted clipboard subtypes or available item actions.
- Rust emits directly to a destroyable window without a pull fallback.
- UI state stores duplicate command result collections.
- SQL schema changes skip TypeScript type mirrors.
- A settings change updates UI schema but not Rust defaults or validation.
- Platform behavior is implemented for one OS and silently falls through on the
  other.

# Clipboard Pipeline

## Capture Flow

The main capture flow is:

```text
OS clipboard change
  -> clipboard::watcher
  -> clipboard::read::ClipboardReader
  -> clipboard::payload::ClipboardPayload
  -> clipboard::ingest::build_item_with_settings
  -> db::items::upsert_item
  -> clipboard://updated
  -> React list fetches enriched item/page
```

Manual `read_clipboard` reuses the same `build_item_with_settings` and
`persist_and_notify` path, so manual and automatic capture share dedupe,
source-app upsert, copy sound, and event semantics.

Reference files:
- `src-tauri/src/clipboard/watcher.rs`
- `src-tauri/src/clipboard/read.rs`
- `src-tauri/src/clipboard/payload.rs`
- `src-tauri/src/clipboard/ingest.rs`
- `src-tauri/src/db/items.rs`
- `src-tauri/src/commands/clipboard.rs`

## Payload and Storage Semantics

`ClipboardPayload` is a typed read result, not a database model. The read layer
only classifies one copy as text, image, or files based on the user-configured
capture order.

`build_item_with_settings` decides persisted fields:

- Text `content` is the selected source representation: HTML source, RTF source,
  or plain text.
- Text `search_text` is the OS plain text representation for FTS and plain paste.
- Text `summary` is a trimmed, capped list preview.
- Plain text subtypes (`url`, `email`, `color`, `path`) are detected in Rust by
  `clipboard/detect.rs`.
- Images are stored as PNG resources; the database `content` is the image file
  name.
- Files store newline-separated paths in `content` and compact file kind markers
  in `file_types`.

Do not parse HTML/RTF, infer color safety, or split file entries in React when a
command can return a prepared field. `ClipboardItem` already includes
`fileEntries`, `filesPreviewKind`, `colorPreview`, `availableActions`,
`displayCreatedAt`, `imageThumbnailPath`, and source app icon paths after command
enrichment.

## Dedupe and Reuse

`db::items::content_hash` hashes `"<kind>:<content>"` with blake3. `upsert_item`
checks this hash:

- New content inserts a row.
- Existing content increments `use_count`, refreshes `updated_at`, and returns
  `deduplicated = true`.

Metadata edits such as notes, favorites, pins, and groups must not refresh
`updated_at`; that timestamp means "content was reused." This is called out in
`db::items::update_item_group` and the project rules.

## Loop Suppression

Writing history back to the system clipboard would normally trigger the watcher.
`clipboard::write` calls `WritebackGuard::suppress` before writing, and the
watcher calls `should_skip` after rebuilding the item hash. The guard stores
content hashes with a short TTL, not a global boolean, so a real unrelated copy
is not swallowed.

If writeback behavior changes, keep the hash computed by the write path aligned
with the hash the watcher will compute after reading the OS clipboard.

Reference files:
- `src-tauri/src/clipboard/write.rs`
- `src-tauri/src/clipboard/guard.rs`

## Sensitive Content

Secret detection lives in `clipboard/secrets.rs` and is applied during ingest.
Settings can skip saving secrets or save them with `is_sensitive = true`.
List and preview commands apply redaction based on the current settings before
returning payloads.

React should display `isSensitive` and render the provided text. It should not
try to run its own secret detector.

## Source Applications and Icons

The watcher captures the frontmost app before reading the clipboard because
foreground state can change by the time async persistence runs. Source apps are
materialized through `AppIconStore` and `AppsRegistry`, then upserted into
`clipboard_apps`. App upsert failure logs a warning and clears `source_app_id`
instead of dropping the clipboard item.

Source app IDs are platform-specific:

- macOS: bundle id.
- Windows: executable path.

The command layer resolves icon file names into absolute paths for React.

## Tests

Clipboard logic has focused Rust tests for pure parsing, ingestion, storage,
loop suppression, and repository behavior. Real system clipboard round trips are
marked `#[ignore = "touches the real system clipboard; ..."]` in
`clipboard/read.rs`, `clipboard/write.rs`, and `clipboard/watcher.rs`. Run those
ignored tests only in a desktop session when changing native clipboard behavior.

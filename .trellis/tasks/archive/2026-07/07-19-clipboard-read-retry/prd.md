# Retry transient clipboard read failures

## Goal

Prevent EcoPaste from dropping PixPin image captures when another Windows clipboard listener,
notably Xiaomi PC Manager's cross-device clipboard, briefly holds the clipboard open.

## Requirements

- Keep Xiaomi PC Manager cross-device clipboard enabled and functional.
- Retry only transient clipboard read errors in EcoPaste's existing watcher path.
- Preserve the current capture ordering, source-app detection, filtering, writeback guard,
  deduplication, persistence, and update-event behavior.
- Keep the retry window short enough that clipboard capture remains responsive.
- Produce a separately runnable Windows test build without replacing the user's installed copy.
- Publish the focused source change to `kumu-ze/EcoPaste` and open a draft PR against
  `EcoPasteHub/EcoPaste` after local validation.

## Acceptance Criteria

- [ ] A first-read failure followed by a successful read is captured instead of dropped.
- [ ] Exhausted retries log one final warning and return without persisting invalid data.
- [ ] `Ok(None)` is not retried.
- [ ] Focused Rust tests cover success, retry success, empty content, and exhausted retries.
- [ ] Formatting, clippy, Rust tests, frontend checks, and a Windows build pass.
- [ ] With Xiaomi cross-device clipboard enabled, repeated PixPin screenshots appear in EcoPaste.
- [ ] The original installed EcoPaste remains available for rollback.

## Notes

- The observed implementation currently logs `clipboard watcher: read failed` and immediately
  returns from `on_clipboard_change`, so the same clipboard update is never read again.

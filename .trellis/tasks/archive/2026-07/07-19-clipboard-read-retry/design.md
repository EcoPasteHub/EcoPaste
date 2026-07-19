# Design

## Scope

Change only the synchronous OS-read step in `src-tauri/src/clipboard/watcher.rs`.
No settings, database, frontend, or event contracts change.

## Retry Policy

Wrap `ClipboardReader::read_with_capture` in a small synchronous retry helper. The watcher
already runs on its own blocking thread, so short sleeps do not block Tauri's async runtime or
the UI. Retry after transient errors using bounded backoff delays of 15ms, 35ms, and 75ms
(four reads, 125ms maximum added latency). Return immediately on `Ok(Some(_))` or `Ok(None)`.

The helper remains generic over the read closure so deterministic unit tests can use zero-delay
policies without touching the real system clipboard.

## Observability

Do not warn for intermediate failures that recover. If all reads fail, preserve the existing
warning at the call site and include the final error. This avoids noisy logs during ordinary
clipboard contention while retaining failure evidence.

## Rollback

The test build is emitted to the repository build output and copied to a separate artifact
folder. The installed `D:\tool\pc\EcoPaste\EcoPaste.exe` is not overwritten during validation.

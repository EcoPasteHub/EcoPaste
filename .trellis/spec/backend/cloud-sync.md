# Cloud Sync (`src-tauri/src/cloud/` + EcoPasteServer)

Client side of the self-hosted cloud sync. The server lives in a separate repo
(`ack528/EcoPasteServer`: Go service in `server/`, Tauri admin app in `desktop/`).

## Client module map

| File | Owns |
| --- | --- |
| `pin.rs` | rustls verifier that trusts exactly one SHA-256 certificate fingerprint (or records it in probe mode) |
| `client.rs` | REST client (`login`, `me`, `history`, `post_item`, `put_blob`/`get_blob`, `logout`), `CloudError`, address normalization |
| `mod.rs` | `CloudState`: creds (`state/cloud.json`), background `run` loop (WebSocket, backoff, epoch-based restart), apply / upload, file fetch relay |
| `commands/cloud.rs` | `cloud_get_status/probe/login/logout/pull_recent`, error -> i18n mapping |

## Protocol (server API, TLS 1.3 only)

- `POST /api/login` -> `{token, deviceId}`; `POST /api/logout` revokes the calling device.
- `POST /api/items {hash, kind: text|image|files, size, blob?, payload}`; image items need the blob uploaded first
  (`PUT /api/blobs/{sha256}`, hash verified by the server). Payloads: text `{plain,html,rtf}`, image
  `{width,height,blob}` (the blob hash must also be inside the payload: clients only receive `payload`),
  files `{offer, entries:[{relPath,isDir,size}]}` (manifest only).
- `GET /api/ws?since=<seq>`: server pushes `item` messages (backfill first, then realtime) and a `hello`.
  File content is never stored: requester sends `fetch {req,to,offer,index,offset,len}`, the server routes it
  to the origin device which answers `fdata {req,to,data(base64)|err}`.
- `seq` is per account; an item with an existing `hash` replaces the old one and gets a new `seq`.

## Trust and security

- The server uses a self-signed certificate. First connect: `cloud_probe` returns the fingerprint, the user compares
  it with the admin app's Overview page and confirms; afterwards only that fingerprint is accepted (a mismatch maps to
  `CloudError::PinMismatch`). Never add a "skip verification" path.
- Only HTTPS in transit (user decision): the server sees plaintext. `is_sensitive` items are never uploaded; apps
  excluded by the clipboard watcher never reach `broadcast_item`.
- Token, fingerprint and server URL live in `state/cloud.json`, NOT in `settings.json` (settings mirror to the
  frontend). `settings.cloud` only has `enabled` and `pullRecent`.
- `reqwest` uses `rustls-no-provider`: install the ring provider before building any client
  (`pin::client_config` does it; same trick as tauri-plugin-updater).

## Coexistence with LAN sync

- Same hooks in `clipboard::watcher` (`sync::broadcast_item` + `cloud::broadcast_item`), same `sync::apply` write path
  and `WritebackGuard`, so received items never re-ingest or re-broadcast.
- A content hash arriving through both channels is applied once: `sync::apply::seen_recently` (20 s TTL, only for
  live clipboard writes). History pulls (`to_clipboard = false`) skip this check and never touch the clipboard.
- File offers are shared: `SyncState::prepare_files` caches `content_hash -> (offer, entries)` so LAN and cloud
  advertise the same offer; `read_offer_range` serves both. Cloud peers appear to `remote_files` as `cloud:<deviceId>`
  and `sync::manager::fetch_file_range` delegates those to `cloud::fetch_file_range`.

## Testing

- `cloud/e2e_tests.rs` spawns the real `ecopaste-server.exe` (env `ECOPASTE_SERVER_EXE`, default
  `D:\Projects\claude\EcoPasteServer\desktop\src-tauri\ecopaste-server.exe`; skipped when missing) on random ports
  and drives pin mismatch, bad credentials, realtime text/image, history pull and logout.
- Run the LAN (`sync::e2e_tests`) and cloud (`cloud::e2e_tests`) ignored suites in **separate** invocations; they each
  build a Tauri app and share process-wide test sinks.
- Server tests: `. .\env.ps1; cd server; go test ./...` in the EcoPasteServer repo.

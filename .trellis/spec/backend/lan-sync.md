# LAN Clipboard Sync (`src-tauri/src/sync/`)

Contract notes for the cross-device sync module. Read before changing pairing,
the wire protocol, or what gets sent to peers.

## Module map

| File | Owns |
| --- | --- |
| `identity.rs` | Local `device_id`, paired-device table (`<app_data>/state/sync-devices.json`, atomic write, long-term key in hex) |
| `pairing.rs` | 6-digit code lifecycle + SPAKE2 handshake (`run_handshake`) |
| `session.rs` | Mutual HMAC challenge/response for already-paired devices, derives session keys |
| `protocol.rs` | `Message` enum, length-prefixed frames, ChaCha20-Poly1305 `SecureReader/Writer` |
| `discovery.rs` | mDNS `_ecopaste-sync._tcp.local.` register/browse (TXT: `id`, `name`, `pair`, `v`) |
| `manager.rs` | `SyncState`: listener, supervisor, connections, broadcast, pairing entry points |
| `apply.rs` | Receive side for text/image: build item -> write system clipboard -> `persist_and_notify` |
| `files.rs` / `remote_files/` | Send-side manifest + ranged reads; receive-side virtual files (Windows OLE) |

## Security invariants

- The 6-digit code is only a SPAKE2 password; it never goes on the wire and is never
  used directly as a key. One code allows at most `MAX_ATTEMPTS` (3) handshakes, lives
  `CODE_TTL` (5 min), and is cleared on success. `begin_attempt` reserves an attempt
  *before* the handshake so concurrent guesses cannot exceed the cap.
- Long-term key = HKDF of the SPAKE2 secret salted with the sorted device ids. Session
  keys are derived per connection from fresh nonces; each direction has its own key and
  a counter nonce (replayed/reordered frames fail authentication).
- Never serialize `PairedDevice.key` to the frontend: status DTOs (`PeerStatus`) omit it.
- Frames are capped at `MAX_FRAME` (16 MiB); bulk data is chunked at `CHUNK_SIZE`.

## What is (not) sent

- Skipped: `is_sensitive` items, anything the clipboard watcher already dropped
  (excluded apps, disabled capture kinds), kinds disabled in `settings.sync`.
- Received items go through `WritebackGuard` before the clipboard write, so the local
  watcher ignores them: no re-ingest and no re-broadcast (star topology, no relay).
- Files are never pre-transferred. The sender keeps up to three offers that have
  yet to be read. The first range request protects its offer from eviction by
  newer copies, and each further request refreshes its 30-minute idle expiry.
  The receiver places a virtual-file `IDataObject` and pulls ranges with
  `FetchFile` only when an app reads the stream.

## Discovery fallback (manual IP)

- Many campus / office Wi-Fi networks drop multicast between clients, so mDNS never sees
  the peer. The listener prefers fixed port `DEFAULT_PORT` (52526; random if taken), the
  pairing-code dialog shows local IPv4s (`SyncStatus.addresses`, excluding loopback,
  link-local and the 198.18.0.0/15 proxy-TUN range), and `sync_pair_with_code` accepts an
  optional `host` (`ip` or `ip:port`) to dial directly.
- Every successful pairing/session records the peer's `ip:port` in `PairedDevice.addr`
  (`DeviceMeta.port` / `Intro::Session.port` carry the listening port). `supervise_loop`
  falls back to that address when mDNS has not discovered the peer.
- Protocol version is 2 since these fields were added; v1 peers are ignored/rejected.

## Gotchas

- Windows clipboard open can be refused while another app holds it: receive-side writes
  retry with backoff (`write_item_with_retry`). Do not hold `ClipboardContext` across `.await`.
- OLE clipboard objects must live on an STA thread that pumps messages. In
  `remote_files/windows.rs`, each presented offer keeps its STA alive as long as
  OLE or a paste target holds the data object or a derived stream. A new clipboard
  item can coexist with old paste streams, each with its own cursor and prefetch
  window. The stream `Read` calls `block_on` and must run outside a Tokio worker.
- macOS virtual files (`NSFilePromiseProvider`) are **not implemented** (TODO in
  `remote_files/mod.rs`); text and image sync work on both platforms.
- `windows` crate needs the `implement` feature for `#[implement(IDataObject/IStream)]`.
- Only the peer with the smaller `device_id` dials; `supervise_loop` retries every 5 s.

## Testing

- Unit tests live next to each module (crypto, pairing, framing, offers).
- `e2e_tests.rs` runs two `SyncState`s over real mDNS + TCP inside one test process. The
  system clipboard write and OLE placement are replaced by `apply::TEST_SINK` /
  `remote_files::CAPTURED`, so no desktop clipboard permission is needed. Run:
  `cargo test --lib sync:: -- --include-ignored --test-threads=1`.
- On the GNU toolchain the test exe needs a comctl32 v6 manifest (embed one, see
  `target/test-manifest`) or it dies with `STATUS_ENTRYPOINT_NOT_FOUND`; this is an
  environment issue, not a code issue.

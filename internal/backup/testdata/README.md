# Legacy backup fixtures

`legacy-plain.ecopastebak` and `legacy-encrypted.ecopastebak` are generated
from the Rust implementation in `legacy/backend/src/backup/mod.rs`. The
encrypted fixture uses the password `legacy-password`.

The payload contains a minimal published-schema database with three items
(two text, one image), one group, one source app, one image resource and a
camelCase `settings.json`. The fixtures pin the legacy ZIP entry layout,
manifest fields, container magic, Argon2id parameters and
XChaCha20-Poly1305 envelope so the Go implementation can be checked against
bytes rather than only its own round trip.

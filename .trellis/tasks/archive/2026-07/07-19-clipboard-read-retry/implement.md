# Implementation Plan

- [x] Add the bounded retry helper and policy constants to the clipboard watcher.
- [x] Route watcher reads through the helper without changing downstream behavior.
- [x] Add focused unit tests for immediate success, recovered error, empty result, and exhaustion.
- [x] Install the pinned Rust 1.96 toolchain and frontend dependencies in the isolated checkout.
- [x] Run rustfmt, clippy, Rust tests, frontend lint/type checks, and the Windows Tauri build.
- [x] Copy the runnable test artifact to a clearly named sibling workspace folder.
- [ ] Validate repeated PixPin image capture while Xiaomi cross-device clipboard remains enabled.
- [ ] Review the full diff, commit only task files and source changes, push the branch, and open
  a draft PR from `kumu-ze:agent/clipboard-read-retry` to `EcoPasteHub:master`.

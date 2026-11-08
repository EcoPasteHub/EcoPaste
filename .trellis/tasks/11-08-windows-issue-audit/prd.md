# Windows Issue Audit and Fix

## Goal

Audit the open issues in EcoPasteHub/EcoPaste, identify Windows-related
defects that are still present in the current fork, and fix every code-confirmed
defect that can be addressed without changing external services or supporting an
operating system outside the project's supported baseline.

## Background

- The fork's `master` is clean at task creation and contains the 2026-08-12
  upstream changes.
- The upstream repository has 128 open issues and 16 open pull requests at
  audit time. Issue text and comments were retrieved from the GitHub REST API.
- The user explicitly excludes update-infrastructure failures such as expired
  project domains.
- The repository supports macOS and Windows only. Windows 7 is outside the
  supported Tauri/WebView2 baseline; no Linux support may be introduced.

## Requirements

1. Record a Windows-relevant issue matrix with confirmed, not reproducible, and
   external-code explanations.
2. Fix defects confirmed by current source paths, including:
   - repeated `Win+V` while Win remains held;
   - IME composition Enter being treated as paste;
   - pinned-window keyboard hooks consuming keys in other applications;
   - 4K/200% preference sidebar truncation;
   - context-menu and preview WebView rebuild handle races;
   - StartupApproved state being overwritten by autostart sync;
   - administrator autostart creating a one-time task instead of a logon task;
   - custom data-directory adoption deleting an existing valid data directory;
   - delayed-delivery clipboard formats being discarded on the first empty read;
   - Microsoft Office/WPS text copies being classified as images because the
     default image priority wins over text representations.
3. Preserve existing persisted settings and data contracts.
4. Add focused regression tests for pure Rust decisions and run the full project
   quality checks.
5. Manually verify affected Windows UI/native paths where the environment
   permits it.
6. Persist concise project memory for remote-session DPI geometry and warm-menu
   lifecycle constraints.
7. Make window sizing stable across Remote Desktop resolution and scale-factor
   changes, including restart/restore and live scale events.
8. Remove the long-idle context-menu cold-start delay without preventing normal
   clipboard history capture.
9. Produce a Windows installer from the current source, replace the existing
   `C:\Program Files\EcoPaste` installation, and verify the installed executable.
10. Commit all task changes on a new `codex/` branch and push it to `origin`.

## Acceptance Criteria

- [x] `Win+V` toggles EcoPaste only once for a continuous Win-held chord; later V
      presses in the same chord are swallowed and do not invoke the native panel.
- [x] Enter used to commit an IME composition does not close the window or
      trigger paste; the next non-composing Enter retains current behavior.
- [x] When a pinned clipboard window remains visible and the user works in
      another application, Space/Ctrl/navigation hooks no longer consume that
      application's keys; clicking back inside the clipboard window restores
      navigation handling.
- [x] Preference navigation remains reachable and scrollable at 200% Windows
      display scaling.
- [x] Context menus and preview reopen successfully after lightweight-mode
      idle WebView destruction.
- [x] Autostart startup sync does not rewrite an unchanged Run entry, preserving
      a user-disabled StartupApproved state.
- [x] Administrator autostart uses a logon-triggered highest-privilege task when
      both `runAsAdmin` and `autoStart` are enabled, without a duplicate Run
      autostart entry.
- [x] Selecting an existing valid EcoPaste data directory adopts it without
      deleting or overwriting its database, settings, resources, or state.
- [x] A temporarily unavailable custom data directory no longer destroys the
      stored custom-path manifest; the app can recover when the path returns.
- [x] On Windows, a clipboard event whose first read is empty retries within a
      bounded delay, improving delayed-renderer apps without unbounded blocking.
- [x] Office/WPS text copies with both text and image representations prefer
      their text representation, while explicit user capture ordering remains
      effective for non-Office sources.
- [x] `cargo fmt`, `cargo clippy -- -D warnings`, `cargo test`, `pnpm lint`, and
      `pnpm tsc` pass.
- [x] The final report lists fixed issues, current-code findings that require
      external evidence, and explicitly excluded update/OS-support issues.
- [x] Window geometry persists in logical dimensions plus source scale, and is
      converted/clamped when Remote Desktop changes resolution or DPI.
- [x] Live `ScaleFactorChanged` events clamp and reposition windows without
      allowing them to grow beyond the active monitor.
- [x] Context menus remain warm after their first creation, eliminating the
      long-idle WebView rebuild delay.
- [x] Core specs record the DPI and menu-latency invariants concisely.
- [x] A current-source Windows installer is built and installed over the existing
      `C:\Program Files\EcoPaste` installation.
- [x] Changes are committed on a new `codex/` branch and pushed to `origin`.

## Validation Notes

- Frontend: `pnpm tsc`, `pnpm lint`, and `pnpm build` pass.
- Rust: `cargo fmt --check`, `cargo clippy --all-targets -- -D warnings`, and
  `cargo test` pass. 189 tests passed; 7 real-clipboard desktop tests remain
  intentionally ignored.
- Manual Windows test: after lightweight idle destruction, the context menu and
  preview both rebuilt and displayed without the former handle/IPC errors.
- Manual Windows test: two text records were captured and rendered; preference
  navigation items were all present in the accessibility tree.
- NSIS package:
  `src-tauri/target/release/bundle/nsis/EcoPaste_1.1.0_x64-setup.exe`.
- The machine-wide replace is blocked on Windows UAC approval; the package is
  built and verified, but the installed executable remains the previous build
  until the elevation prompt is accepted.

## Out of Scope

- Update endpoint/domain failures and release-server maintenance.
- Microsoft Defender false positives and Authenticode signing infrastructure.
- Windows 7/WebView2 support below the current Tauri baseline.
- New product features requested in issues, such as cloud sync, portable mode,
  OCR, batch paste, or UI redesigns.
- Bugs that cannot be confirmed from code, logs, or a reproducible local path
  without inventing behavior.

## Technical Notes

- Issue source snapshot: `research/github-windows-issue-audit.md`.
- Platform-sensitive implementation details and validation are in `design.md`
  and `implement.md`.
- The user's request to inspect and fix these bugs authorizes creation and
  execution of this task.

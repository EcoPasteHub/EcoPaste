# Write README Documentation

## Goal

Replace the default Tauri template README with project-specific documentation for the Rust-first EcoPaste refactor, and add a Simplified Chinese README alongside the English default README.

## What I already know

* The current `README.md` is still the default Tauri + React + TypeScript template.
* `README.zh-CN.md` does not exist yet.
* The project is EcoPaste, a Rust-first Tauri v2 clipboard manager refactor.
* The supported platforms for this refactor are macOS and Windows only.
* The refactor has dropped legacy Linux support and there are no current plans to support Linux again.
* `package.json` version is `0.6.0-beta.3`, and the project is still in beta / not released as a stable version.
* The old EcoPaste README is available as read-only reference under `/Users/ayang/Documents/PersonalProject/2024/EcoPaste_bak`, but it includes outdated Linux and release-link content that must not be copied blindly.
* Current local assets include `public/logo.png` and `public/sponsor-qr.png`.

## Requirements

* Write `README.md` in English.
* Write `README.zh-CN.md` in Simplified Chinese.
* Document the current Rust-first refactor accurately:
  * Tauri v2 desktop app.
  * React 19 + Ant Design 6 + UnoCSS frontend.
  * Rust + SQLite/sqlx backend.
  * Local-first clipboard history and settings.
  * macOS and Windows support only.
  * Linux support removed from the refactor and not currently planned.
* Include practical developer setup:
  * prerequisites,
  * install command,
  * dev command,
  * build command,
  * quality commands.
* Include a concise beta / migration notice explaining that the refactor is not compatible with old data and users should back up old data before trying it.
* Include language links between the two README files.
* Avoid claiming Linux support, stable release availability, or installer/download links that are not present in this repo.
* If Linux is mentioned, mention it only as an explicit unsupported platform notice.

## Acceptance Criteria

* [ ] `README.md` no longer contains the default Tauri template text.
* [ ] `README.zh-CN.md` exists and mirrors the English README's structure and meaning.
* [ ] Both READMEs mention only macOS and Windows as supported platforms.
* [ ] Both READMEs explicitly state that Linux support was dropped in the refactor and is not currently planned.
* [ ] Both READMEs include accurate development and quality commands from `package.json` and project instructions.
* [ ] Markdown links between the English and Chinese READMEs work locally.
* [ ] No outdated old-version Linux download, Homebrew, community, or release badge claims are introduced.
* [ ] Any Linux mention is clearly framed as unsupported in the refactor.

## Definition of Done

* Documentation is updated in both languages.
* Markdown is readable without relying on missing local screenshots.
* A lightweight verification confirms the expected files exist and do not contain Linux support claims outside the unsupported-platform notice.

## Technical Approach

Draft both README files directly from current repo facts: `package.json`, `src-tauri/tauri.conf.json`, `CHANGELOG.md`, `AGENTS.md`, and the old README as read-only tone/reference material. Keep the public-facing documentation conservative because the refactor is still beta.

## Decision (ADR-lite)

**Context**: The old README advertises the released legacy app and includes Linux/download/community details that are no longer true for this refactor.

**Decision**: Make `README.md` the English default and `README.zh-CN.md` the Simplified Chinese translation. Present the repo as a beta Rust-first refactor for developers/testers, not as a stable release landing page.

**Consequences**: The docs are accurate for this repository today and avoid over-promising. Future release/download/community details can be added once the refactor has a real release channel.

## Out of Scope

* Adding screenshots or generating new visual assets.
* Adding installer download links, Homebrew instructions, or release badges.
* Changing application code, package metadata, build scripts, or changelog entries.
* Restoring Linux support or documenting Linux support.

## Technical Notes

* Existing files inspected: `README.md`, `package.json`, `src-tauri/tauri.conf.json`, `CHANGELOG.md`.
* Old read-only references inspected: `/Users/ayang/Documents/PersonalProject/2024/EcoPaste_bak/README.md`, `/Users/ayang/Documents/PersonalProject/2024/EcoPaste_bak/README.en-US.md`.
* Existing project logo is available at `public/logo.png`.

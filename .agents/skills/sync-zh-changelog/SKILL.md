---
name: sync-zh-changelog
description: "Synchronize an EcoPaste release section from CHANGELOG.md into CHANGELOG.zh-CN.md with project terminology and structural validation. Use when the user explicitly invokes $sync-zh-changelog or asks to generate, translate, update, or verify the Chinese changelog for a release."
---

# Sync Chinese Changelog

Synchronize one EcoPaste release from `CHANGELOG.md` to
`CHANGELOG.zh-CN.md` without changing release metadata or unrelated files.

## Trellis routing

- Treat an explicit `$sync-zh-changelog` invocation as the user's standing
  choice to skip Trellis task creation for this narrow documentation change.
  Do not ask again, and do not create, activate, or archive a Trellis task.
- When this skill is selected implicitly from natural-language intent, follow
  the normal Trellis task-creation routing.
- Still respect `AGENTS.md`, the dirty worktree, and applicable release specs.

## Workflow

1. Read both changelog files before editing. Use the version named by the user;
   otherwise select the first release section in `CHANGELOG.md`.
2. If that version already exists in `CHANGELOG.zh-CN.md`, update it in place.
   Otherwise insert it immediately after the `# 更新日志` title. Never create
   a duplicate section.
3. Translate release prose and headings into concise Simplified Chinese.
   Search `src/locales/zh-CN/`, `src/locales/en-US/`, and existing Chinese
   changelog entries with `rg` when product terminology is unclear.
4. Preserve the source release header's version, compare URL, date, section
   order, entry order, issue numbers, commit hashes, and all link targets.
   Translate only human-readable prose. Keep platform and product names such as
   EcoPaste, macOS, Windows, Token, AWS Key, and JWT consistent with the UI.
5. Edit only `CHANGELOG.zh-CN.md`. Do not modify `CHANGELOG.md`, version files,
   release configuration, or unrelated dirty changes. Do not commit or push.
6. Validate the result from the repository root:

   ```bash
   python3 .agents/skills/sync-zh-changelog/scripts/check_sync.py
   git diff --check -- CHANGELOG.zh-CN.md
   ```

   Pass `--version <version>` to validate a user-selected non-latest release.
7. Fix every validation error, rerun both checks, then report the synchronized
   version and validation result.

## Translation headings

Use these established mappings when the source headings occur:

- `✨ Features` → `✨ 新功能`
- `🐛 Bug Fixes` → `🐛 问题修复`
- `⚡️ Performance` → `⚡️ 性能优化`
- `⏪ Reverts` → `⏪ 回退`
- `⚠️ Upgrade Notice` → `⚠️ 升级说明`

For an unfamiliar heading, translate it faithfully while preserving any emoji
and its position.

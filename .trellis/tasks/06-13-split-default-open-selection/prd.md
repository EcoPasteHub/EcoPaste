# Split Default Open Selection Settings

## Goal

Replace the current single "select all on open" preference with three independent Select preferences so users can choose the exact clipboard filter target applied when the main window opens: range, category, and custom group.

## Requirements

* Replace the single `selectAllGroupOnOpen` preference with three independent Select settings:
  * select range on open: keep last selection / all / favorites
  * select category on open: keep last selection / all / text / image / files
  * select group on open: keep last selection / all / one custom group
* When the main clipboard window becomes visible, apply only dimensions whose Select value is not `preserve`.
* Keep the existing default behavior equivalent to the current default: all three new Select values default to `preserve`.
* Surface all three settings in Preferences > Interface > Main Window with zh-CN and en-US labels, descriptions, and option labels.
* Update TypeScript setting mirrors, preference schema, setting icons, Rust settings model, and settings default/missing-field tests.

## Acceptance Criteria

* [x] Preferences shows three separate Select settings for open-time range, category, and group.
* [x] Choosing a non-preserve value for only one Select changes only that dimension when the main window opens.
* [x] Existing "return to top on open" behavior still works independently.
* [x] Rust settings deserialize with the new fields and default missing fields to `preserve`.
* [x] Frontend type-check/lint and relevant Rust checks pass.

## Definition of Done

* Tests added/updated where appropriate.
* Lint / typecheck green for touched frontend code.
* Rust format/checks green for touched backend code.
* User-visible text exists in both zh-CN and en-US.
* No compatibility layer for `selectAllGroupOnOpen`, because the project is still pre-release per AGENTS.md.

## Technical Approach

Rename the old single Rust/TypeScript setting into three selection fields under `clipboard.window`. Range and category use closed Rust/TypeScript unions. Group uses `preserve`, `all`, or `group:<id>` so the Select can target runtime custom groups. Update the visibility handler in the clipboard list to apply each non-preserve selection independently.

## Decision (ADR-lite)

**Context**: The previous setting title said "select all", but its behavior reset all three clipboard filter dimensions together.

**Decision**: Model range, category, and custom group as separate persisted selection values rather than booleans. Range and category are closed enums; custom group selection uses a string encoding for dynamic group ids.

**Consequences**: The settings UI gets two additional rows and a dynamic group Select control. Because the app has not shipped this Rust-first version yet, the removed field does not need a migration or backward compatibility shim.

## Out of Scope

* Changing the clipboard filter UI layout itself.
* Adding per-window-profile defaults.
* Migrating the old `selectAllGroupOnOpen` JSON key.

## Technical Notes

* Previous Rust field: `src-tauri/src/settings/model.rs` had `Window::select_all_group_on_open`.
* Previous frontend mirror: `src/types/settings.ts` had `Window.selectAllGroupOnOpen`.
* Previous behavior: `src/pages/Clipboard/components/List.tsx` reset `range`, `category`, and `groupId` together when `selectAllGroupOnOpen` was true.
* Previous preferences entry: `src/pages/Preference/config/preferenceSchema.ts` rendered one switch at `clipboard.window.selectAllGroupOnOpen`.
* Previous i18n keys: `src/locales/zh-CN/preferences.json` and `src/locales/en-US/preferences.json` defined `selectAllGroupOnOpen`.
* Corrected user clarification: the three settings should be Select controls, not switches.

## Verification

* `pnpm exec biome check src/constants/windowOpenSelection.ts src/types/settings.ts src/pages/Clipboard/components/List.tsx src/pages/Preference/config/preferenceSchema.ts src/pages/Preference/types/preferences.ts src/pages/Preference/components/PreferenceSettingRow.tsx src/pages/Preference/components/settingControls/ClipboardGroupSelectControl.tsx src/pages/Preference/components/settingControls/settingVisual.ts src/locales/zh-CN/preferences.json src/locales/en-US/preferences.json`
* `pnpm tsc`
* `pnpm build`
* `cd src-tauri && cargo fmt --check`
* `cd src-tauri && cargo clippy -- -D warnings`
* `cd src-tauri && cargo test`

## Spec Update Review

Updated `.trellis/spec/backend/settings-window-platform.md` with the window-open selection settings contract. The important durable detail is that custom group selection uses `group:<id>` rather than a raw group id, while `preserve` and `all` remain reserved built-in values.

# Internationalize Window Lifecycle Name

## Goal

Fix the Preferences diagnostics window lifecycle table so the onboarding window uses the current UI language instead of showing the raw internal `onboarding` label.

## What I Already Know

- The screenshot shows `onboarding` rendered in the Window Lifecycle modal while other window names are localized.
- `src/pages/Preference/components/settingControls/ActionControl.tsx` maps known lifecycle snapshot labels through `WINDOW_LIFECYCLE_WINDOW_LABEL_KEYS`.
- Unknown labels intentionally fall back to the raw label for diagnostics.
- `WINDOW_LABEL.ONBOARDING` already exists in `src/constants/windows.ts` and mirrors Rust `ONBOARDING_WINDOW_LABEL`.

## Assumptions

- Onboarding is a known application window and should be localized in this diagnostics table.
- Unknown future labels should continue to fall back to the raw label for troubleshooting.

## Requirements

- Add onboarding to the known lifecycle window label mapping.
- Add matching `windows.onboarding` entries in both `zh-CN` and `en-US` preferences locale files.
- Keep the existing fallback behavior for unknown labels.

## Acceptance Criteria

- [ ] The lifecycle snapshot row for label `onboarding` renders as Chinese in `zh-CN`.
- [ ] The lifecycle snapshot row for label `onboarding` renders as English in `en-US`.
- [ ] No raw `onboarding` text is shown for the known onboarding window in the diagnostics modal.
- [ ] Frontend lint/type checks pass for the touched files.

## Definition of Done

- Implementation is scoped to the diagnostics table label formatting and locale strings.
- Both locale files are updated together.
- Quality checks are run and results are recorded in the final response.

## Out of Scope

- Changing Rust lifecycle snapshot payloads.
- Changing Tauri window labels or routes.
- Localizing unknown/unregistered diagnostic labels.

## Technical Notes

- Relevant component: `src/pages/Preference/components/settingControls/ActionControl.tsx`.
- Relevant locales: `src/locales/zh-CN/preferences.json`, `src/locales/en-US/preferences.json`.
- Relevant specs: frontend component, quality, type-safety, and code reuse guides.

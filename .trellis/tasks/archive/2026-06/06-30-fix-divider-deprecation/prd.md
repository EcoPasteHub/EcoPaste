# Fix antd Divider deprecation warning

## Goal

Remove the runtime warning shown by Ant Design when `Divider` uses the deprecated `type` prop.

## Requirements

* Replace deprecated `Divider` `type` usage with the current Ant Design API.
* Keep the existing vertical divider appearance and surrounding layout unchanged.
* Limit the change to the warning source.

## Acceptance Criteria

* [ ] No source code uses `Divider type="vertical"`.
* [ ] Preference switch controls still render with a vertical divider.
* [ ] Frontend lint/type checks pass for the touched code.

## Definition of Done

* Relevant frontend specs reviewed.
* Code updated with the minimal API migration.
* Quality check run or any inability to run it reported.

## Technical Approach

The warning points to Ant Design's `Divider` API change. The affected usage is in `src/pages/Preference/components/settingControls/SwitchControl.tsx`, where a vertical divider separates switch labels from controls. Change `type="vertical"` to `orientation="vertical"`.

## Out of Scope

* UI redesign.
* Broad Ant Design API audit beyond the warning source.
* Backend or settings behavior changes.

## Technical Notes

* Located usage with `rg "<Divider|Divider\\b" src -n`.
* Relevant spec index: `.trellis/spec/frontend/index.md`.

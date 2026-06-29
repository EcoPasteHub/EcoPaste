# Enrich Onboarding Final Welcome Page

## Goal

Make the final onboarding step feel complete and consistent with the first welcome step. The Done page should invite users to start using EcoPaste, with the same calm hero-and-three-cards rhythm as the Welcome page.

## Requirements

- Replace the sparse Ant Design `Result` final step with a richer custom layout.
- Keep the final step inside the existing onboarding shell, progress indicator, language switcher, and bottom action buttons.
- Preserve the dark onboarding visual direction and Ant Design token-based colors.
- Add concise completion content:
  - Brand/logo and "ready" title area.
  - Three cards matching the Welcome page layout and density.
- Keep strings localized in both `zh-CN` and `en-US`.
- Avoid changing onboarding persistence, step order, permissions, shortcut recording, import behavior, or finish command semantics.

## Acceptance Criteria

- [ ] The last onboarding page no longer relies on Ant Design `Result` as the whole content.
- [ ] The page uses the same `OnboardingStepLayout` + three-card pattern as the Welcome step.
- [ ] The layout fits comfortably within the existing fixed onboarding window.
- [ ] The final "Start using" action remains the existing footer action.
- [ ] All new user-visible text exists in both onboarding locale files.
- [ ] `pnpm lint` and `pnpm tsc` pass.
- [ ] The onboarding final page is manually verified in the app or local browser.

## Definition of Done

- Frontend implementation follows component guidelines (`FC<Props>`, props destructured inside the function body, named handlers where events exist).
- Styling uses UnoCSS utilities and Ant Design token color classes.
- No new frontend business state or Rust commands are introduced.
- CHANGELOG is updated if this is considered a user-visible new capability; otherwise this remains a visual polish change only.

## Technical Approach

Implement the Done step in `src/pages/Onboarding/components/DoneStep.tsx` by mirroring `WelcomeStep`'s `OnboardingStepLayout` and `OnboardingCard` composition. Update `src/locales/zh-CN/onboarding.json` and `src/locales/en-US/onboarding.json` for the new copy.

## Decision (ADR-lite)

**Context**: The final page needs visual richness, but onboarding should remain a compact setup wizard rather than a marketing page.

**Decision**: Reuse the Welcome page composition for Done: hero logo/title/description plus three cards. Avoid custom completion dashboards or separate preview panels because they make the fixed-size onboarding window feel crowded.

**Consequences**: This improves perceived completeness without adding new onboarding state or changing behavior, while keeping the final step calmer and easier to scan.

## Out of Scope

- Changing onboarding step order or navigation behavior.
- Tracking exact permission/import/shortcut completion status for the final summary.
- Adding a new illustration/image generation pipeline.
- Reworking the whole onboarding shell or earlier step pages.

## Technical Notes

- Existing final step: `src/pages/Onboarding/components/DoneStep.tsx`.
- Existing shell: `src/pages/Onboarding/components/OnboardingShell.tsx`.
- Existing onboarding PRD: `.trellis/tasks/06-12-onboarding/prd.md`.
- Relevant frontend specs:
  - `.trellis/spec/frontend/index.md`
  - `.trellis/spec/frontend/component-guidelines.md`
  - `.trellis/spec/frontend/quality-guidelines.md`

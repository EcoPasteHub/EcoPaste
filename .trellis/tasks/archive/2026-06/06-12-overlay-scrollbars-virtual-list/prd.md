# OverlayScrollbars Virtual List Integration

## Goal

Introduce OverlayScrollbars as the custom scrollbar foundation and use it first in the main clipboard history virtual list. The first step should keep existing list behavior intact while making the scroll container consistent across macOS and Windows.

## Requirements

- Add the official OverlayScrollbars React integration needed by this app.
- Create a reusable virtual-list scroller component for `react-virtuoso`.
- Wire the main clipboard history `Virtuoso` to the custom scroller.
- Preserve current list behavior: virtual range loading, top detection, sticky pinned items, keyboard navigation, scroll-to-top on open, deferred reload, hover preview close on scroll, and empty/loading states.
- Scrollbars should stay hidden by default, appear during mouse movement over the list, hide after leaving, and hide automatically after the pointer stops moving.
- Avoid visual customization for this phase; use OverlayScrollbars default styles.
- Keep ordinary non-virtual scroll containers out of scope until this first path is tested.

## Acceptance Criteria

- [ ] Clipboard history list renders through `react-virtuoso` with an OverlayScrollbars-backed scroller.
- [ ] Scrolling still triggers `rangeChanged` and pagination range loading.
- [ ] Programmatic `scrollToIndex` and `scrollIntoView` still work for open/reset and keyboard navigation.
- [ ] Pinned top items remain sticky and visually stable.
- [ ] Scrollbars auto-hide when the pointer leaves or stops moving.
- [ ] TypeScript and Biome checks pass.

## Definition of Done

- Frontend lint and typecheck are green.
- Main list path is manually runnable for smoke testing.
- No unrelated page scroll containers are changed in this task.
- No scrollbar theme/styling work is added beyond required package CSS.

## Technical Approach

Use `overlayscrollbars-react` with the same `scrollerRef` integration pattern as the official `react-virtuoso` example. A virtual-list wrapper owns the OverlayScrollbars root element, receives Virtuoso's scroll viewport through `scrollerRef`, and initializes OverlayScrollbars with that viewport so Virtuoso keeps receiving all native scroll calls and events. The clipboard `List` should only render through this wrapper and otherwise keep business logic unchanged.

## Decision (ADR-lite)

**Context**: The project needs a gradual scrollbar replacement path, with virtual lists first because the clipboard history is the highest-impact scroll surface.

**Decision**: Start with a virtual-list-specific component rather than forcing one abstraction for all scroll scenarios.

**Consequences**: This keeps the first integration small and lowers regression risk. A separate ordinary scroll container can be added later after the virtual-list path is stable.

## Out of Scope

- Replacing preference, preview, dropdown, popover, or other ordinary scroll containers.
- Custom scrollbar colors, width, radius, hover behavior, or platform-specific theme tuning.
- Rust-side behavior changes.

## Technical Notes

- Main target: `src/pages/Clipboard/components/List.tsx`.
- Existing list uses `react-virtuoso` with `rangeChanged`, `atTopStateChange`, `topItemCount`, a custom `TopItemList`, `scrollToIndex`, and `scrollIntoView`.
- Existing scroll-sensitive behavior lives in refs inside the list component and should remain untouched.
- User-provided reference: OverlayScrollbars + `react-virtuoso` official example at <https://stackblitz.com/edit/vitejs-vite-e53qob?file=src%2FVirtualized.jsx>.

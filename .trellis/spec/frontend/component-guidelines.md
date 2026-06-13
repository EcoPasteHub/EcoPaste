# Component Guidelines

## Component Shape

Components use `FC<Props>` and destructure `props` inside the function body.
This is the local style in `ClipboardCard`, `ClipboardGroupIcon`, and `App`.

```tsx
const ClipboardCard: FC<ClipboardCardProps> = (props) => {
  const { item, isSelected, onPointerEnter } = props;
  // ...
};
```

JSX event callbacks should be named functions in the component body. Use action
names for single-purpose handlers (`handleDragStart`, `handleContextMenu`) and
`handleXxx` for generic events.

Avoid parameter destructuring and implicit-return arrow callbacks in new
components.

## Rendering Backend-Owned Models

Render the fields returned by commands instead of recomputing backend logic:

- `ClipboardCard` dispatches by `item.kind` and displays `item.availableActions`.
- `FilesCard` consumes `fileEntries` and `filesPreviewKind`.
- Color previews use `colorPreview`, which Rust has sanitized.
- `displayCreatedAt` is formatted in Rust.
- Source app icons and image thumbnails are absolute paths prepared by Rust.

If a component needs a field that is expensive, security-sensitive, or derived
from database/OS state, add it to the Rust response model rather than deriving
it in React.

## Styling

Use Ant Design v6 components for controls and UnoCSS utilities for layout.
Colors should come from Ant Design tokens exposed through
`src/unocss/presetAntdColors.ts`, such as `text-ant-secondary`,
`bg-ant-container`, and `border-ant-border`.

Use `cn` from `@/utils/cn` for conditional classes:

```tsx
className={cn("rounded-2 border border-ant-border-secondary", {
  "border-ant-primary bg-ant-blue-1": isSelected,
})}
```

Do not concatenate class strings manually or add arbitrary pixel utilities when
Wind4 numeric spacing works.

## Virtualized Scrollbars

Use `src/components/VirtuosoScroller` when wiring OverlayScrollbars into
`react-virtuoso`. The wrapper owns the OverlayScrollbars root and passes its
`scrollerRef` render-prop result directly to `Virtuoso`, so Virtuoso keeps its
native scroll viewport for range calculation, programmatic scroll, and keyboard
navigation.

The default virtual-list scrollbar behavior should use OverlayScrollbars'
`scrollbars.autoHide: "move"` option: hidden at rest, visible while the pointer
moves over the list, and hidden again when the pointer leaves or stops moving.

Keep ordinary non-virtual scroll containers separate from this component. Add a
plain scroll-area component only when a real non-virtual call site is being
converted, instead of stretching `VirtuosoScroller` across incompatible scroll
contracts.

## HTML / SVG Safety

Clipboard HTML preview is rendered as plain text. Do not add a generic shared
HTML renderer unless a real feature needs rich HTML DOM again.

Any user-provided HTML or SVG rendered into the DOM must be sanitized with
DOMPurify at the rendering boundary. Keep the sanitizer narrow to the feature
and forbid risky tags or attributes there; for example, `ClipboardGroupIcon`
sanitizes custom SVG icons before turning them into a mask image.

Do not use `dangerouslySetInnerHTML` directly in page components. If a future
feature needs it, add a small owning component that wraps DOMPurify and document
the allowed tags / forbidden attributes beside that component.

## Native Menus and Drag

`ClipboardCard` uses Rust-backed native context menus through
`popupClipboardItemMenu` instead of in-web menus. Keep this pattern for clipboard
item actions because the native route avoids known Tauri/muda lifetime issues
and keeps action availability backend-owned.

Drag-out starts through `startDragClipboardItem`; the frontend prevents the
browser drag default and lets Rust own platform drag behavior.

## Accessibility

Use semantic roles and stable focus behavior where the UI is custom:

- Clipboard cards use `role="option"` and `aria-selected`.
- Decorative icons set `aria-hidden="true"`.
- `AssetImage` receives meaningful `alt` text when source app names are known.

For Ant Design controls, prefer built-in components and their `disabled`,
`checked`, `open`, and `onClick` props rather than recreating control semantics.

## Common Mistakes

- Deriving business rules in React that Rust already returns, such as reveal
  actions or sensitive redaction.
- Adding a shared component before two real call sites need it.
- Styling Ant Design internals with global `.ant-*` selectors when semantic
  `classNames` or `styles` slots are available.
- Rendering locale text in only one language.

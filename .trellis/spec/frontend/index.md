# Frontend Development Guidelines

The frontend renders EcoPaste UI and ordinary interactions. It calls Rust for
business data, persistence, OS behavior, and cross-window lifecycle state.

## Guides

| Guide | Scope |
| --- | --- |
| [Directory Structure](./directory-structure.md) | Where frontend modules live and how features are organized |
| [Component Guidelines](./component-guidelines.md) | React component shape, styling, accessibility, HTML safety |
| [Hook Guidelines](./hook-guidelines.md) | Data hooks, event subscriptions, async effects, platform hooks |
| [State Management](./state-management.md) | Valtio UI state, settings mirrors, server data ownership |
| [Quality Guidelines](./quality-guidelines.md) | Linting, i18n, UI checks, review checklist |
| [Type Safety](./type-safety.md) | Type mirrors, command contracts, constants, runtime validation boundaries |

## Layer Rule

React should render command results and send user intent back to Rust. Do not
add frontend copies of database state, content classification, FTS/search
semantics, window positioning, shortcut registration, tray behavior, autostart,
or storage persistence.

## Primary References

- App shell and theme: `src/App.tsx`, `src/hooks/useAppTheme.ts`
- Command wrapper boundary: `src/commands/index.ts`
- Settings mirror: `src/stores/settings.ts`
- Clipboard UI state: `src/stores/clipboardView.ts`
- Virtualized list: `src/pages/Clipboard/components/List.tsx`
- Preference schema: `src/pages/Preference/config/preferenceSchema.ts`
- HTML rendering: `src/components/SafeHtml/index.tsx`

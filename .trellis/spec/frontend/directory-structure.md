# Directory Structure

## Overview

Frontend code is organized by boundary first, then by page. Shared components
and hooks live directly under `src/`; page-specific UI stays under the owning
page directory. Backend-owned contracts are mirrored in `src/types/` and
`src/constants/`.

## Layout

```text
src/
  commands/      # single Tauri invoke wrapper boundary
  components/    # shared UI primitives and reusable widgets
  constants/     # Rust-mirrored command, event, window, action constants
  hooks/         # reusable React hooks for Tauri events, theme, keyboard, data
  i18n/          # react-i18next setup
  locales/       # zh-CN and en-US JSON namespaces
  pages/         # route-level features: Clipboard, Preference, Preview, ContextMenu
  router/        # React Router definitions
  stores/        # Valtio UI state and settings/window mirrors
  styles/        # global SCSS
  types/         # TypeScript mirrors for Rust contracts and UI schemas
  unocss/        # local UnoCSS presets
  utils/         # small shared helpers: cn, is, log, shortcut
```

## Feature Ownership

Use the page directory when a component is only meaningful inside that workflow:

- Clipboard list pieces live under `src/pages/Clipboard/components/` and
  `src/pages/Clipboard/components/cards/`.
- Clipboard preview measurement, layout, cache, and hooks live under
  `src/pages/Preview/`.
- Preference schema, controls, services, and search helpers live under
  `src/pages/Preference/`.

Use `src/components/` only for shared primitives such as `SafeHtml`,
`ShortcutRecorder`, `AssetImage`, `Tooltip`, and `KeyHint`.

## Command and Contract Files

`src/commands/index.ts` is the only file that imports Tauri `invoke`. New Rust
commands get one wrapper there. Business modules import wrappers from
`@/commands`.

Cross-layer string literals are centralized:

- Commands: `src/constants/commands.ts`
- Events: `src/constants/events.ts`
- Window labels: `src/constants/windows.ts`
- Item actions and capture kinds: `src/constants/itemActions.ts`,
  `src/constants/captureKinds.ts`

When adding fields returned from Rust, update the nearest TypeScript mirror in
`src/types/`. `src/types/clipboard.ts` documents which fields are backend
enriched and should be rendered directly.

## Naming and Imports

Use the `@/` alias for source imports. Keep file names aligned with existing
patterns: page components are PascalCase `.tsx`, hooks start with `use`, and
utility modules are short lower-case names.

Avoid adding barrel files inside page subdirectories unless there is already an
established local pattern. Most current components import concrete files.

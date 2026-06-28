# Quality Guidelines

## Commands

Use these checks for frontend work:

```bash
pnpm lint
pnpm tsc
```

`pnpm lint` runs Biome. `pnpm tsc` checks TypeScript without emitting. There is
no frontend test runner configured in this repository, so UI changes also need
a manual app run through the affected path.

For full cross-layer changes, also run Rust checks:

```bash
cd src-tauri
cargo fmt
cargo clippy -- -D warnings
cargo test
```

## Biome Rules That Matter

`biome.json` enforces sorted imports/classes/properties, no unused imports or
variables, no `console.*`, self-closing JSX, and strict unused template literal
cleanup. `noDangerouslySetInnerHtml` is disabled for narrow, sanitizer-owned
escape hatches; page components should still avoid direct raw HTML injection.

Use `@/utils/log` instead of console logging. Keep imports organized by Biome
rather than hand-sorting in a different style.

## Required Frontend Patterns

- Call Tauri only through wrappers in `src/commands/index.ts`.
- Use `async`/`await` and `try`/`catch`; avoid `.then()` chains in new code.
- Use `void 0` for intentionally absent optional fields when matching local
  style, as in `Clipboard/List.tsx`.
- Keep i18n in both `src/locales/zh-CN/` and `src/locales/en-US/`.
- Use `getCurrentWebviewWindow()` for the current window.
- Use `cn` for conditional classes.
- Sanitize user-provided HTML/SVG with DOMPurify at the rendering boundary.

## UI Verification

For UI changes, manually verify the main path and at least one boundary case:

- Clipboard list: empty state, search, scroll pagination, pinned/favorite item,
  hidden-window update behavior if events changed.
- Preview: text, HTML/RTF, image, files, missing files, sensitive redaction.
- Preferences: setting commit, reset, search, both language files, relevant side
  effects such as shortcut/tray/autostart.
- Window changes: main, preference, and preview windows on the affected platform.

If a change affects macOS NSPanel timing or Windows non-focusable keyboard
navigation, manual desktop validation is required. Type checks cannot cover
those paths.

## Review Checklist

- Does the layer ownership match the Rust-first boundary?
- Are command/event names centralized and mirrored?
- Are new user-visible strings present in both locales?
- Does the UI render backend-prepared fields instead of recomputing business
  rules?
- Does a hidden or dormant window need deferred work instead of immediate IPC?
- Are Ant Design token colors used instead of hard-coded colors?
- Did the change avoid global `.ant-*` overrides unless no semantic slot exists?

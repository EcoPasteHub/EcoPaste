# Hook Guidelines

## Async Initialization and Cleanup

Use `useMount` and `useUnmount` from ahooks for async setup and teardown. Keep
`useEffect` for synchronous DOM/state reactions or for async work wrapped inside
the effect with a stale flag.

Local examples:

- `useTauriListen` subscribes on mount, stores the unlisten function in a ref,
  and unsubscribes on unmount.
- `useAppTheme` initializes Tauri theme listeners on mount and cleans them up on
  unmount, then uses `useEffect` to toggle `light`/`dark` classes.
- `App` uses `useMount` to notify Rust that the WebView is ready after
  `settingsReady` resolves.

## Event Handlers and Refs

Long-lived subscriptions should call the latest handler through a ref. Do not
resubscribe on every render just to avoid stale closures.

`useTauriListen` keeps `handlerRef.current = handler`. `useKeyboardEvent` uses
`useLatest` so Windows `keyboard://nav` events call the newest handler while the
subscription remains stable.

In complex components such as `Clipboard/List.tsx`, refs store scroll position,
pending reload state, current reload function, and preview close callbacks so
event handlers can make decisions without re-subscribing.

## Data Fetching

Use backend commands as the data source. `useClipboardItems` wraps
`useInfiniteScroll`, calls `listClipboardItems`, passes Rust-owned pagination
fields, and trusts Rust `total`/`hasMore`.

Do not store command result collections in Valtio as a second database. Keep the
results local to hooks/components unless another UI surface needs a small
derived mirror, such as `clipboardStatsState.total` for the Footer.

## Tauri Events

Use `useTauriListen` for component-level event subscriptions. Type the payload
near the consumer:

- `ClipboardUpdatedPayload` in `Clipboard/List.tsx`.
- Keyboard payloads in `useKeyboardEvent`.
- Window visibility payloads in clipboard preview hooks.

When an event can arrive while the main window is hidden, decide whether to
defer work. The clipboard list defers reload while hidden or scrolled away from
the top to avoid hidden-window IPC churn and scroll jumps.

## Platform Hooks

Use `@/utils/is` for platform/window checks. `useKeyboardEvent` is the standard
keyboard abstraction:

- macOS and focusable windows use browser keyboard events.
- Windows main window receives Rust `keyboard://nav` events because the window is
  non-focusable.

Components should not inspect Tauri window labels directly unless they are
building a reusable platform hook.

## Effects Checklist

Before adding a hook or effect:

- Is the side effect synchronous? Use `useEffect`.
- Does it subscribe, allocate, or return a cleanup handle? Use `useMount` plus
  `useUnmount`, or a tightly scoped `useEffect` cleanup.
- Does the callback need current state inside a long-lived listener? Use a ref.
- Does it fetch business data? Prefer a command wrapper and keep results local.

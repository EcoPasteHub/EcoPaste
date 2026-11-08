# GitHub Windows Issue Audit

Source snapshot: GitHub REST API, 2026-11-08, `EcoPasteHub/EcoPaste`.

## Open Scope

- 128 open issues after excluding 16 pull requests returned by the issues
  endpoint.
- 78 issues have no labels, so platform and severity must be inferred from the
  issue text and current code rather than labels alone.

## Confirmed in Current Code and Targeted

| Issue | Windows symptom | Current-code evidence | Fix |
| --- | --- | --- | --- |
| #1466 | Holding Win and pressing V twice leaks the native clipboard | `shortcut/win_v.rs` clears `V_CONSUMED` on every V keyup, so later presses in the same Win chord can toggle again | Latch the chord until Win is released |
| #1460 | IME Enter commits selection and also closes/pastes | `useKeyboardEvent` forwards editable Enter and composition Enter through the same path | Ignore composition Enter |
| #1465 | Preference navigation clipped at 4K/200% | Sidebar nav is `flex-1` without scrolling | Add scrollable nav viewport |
| #1457 | Context menu fails after idle WebView destruction | `show_for_clipboard_item` queries `scale_factor` immediately after rebuild | Await WebView usability |
| #1449 | Windows startup manager disabled state is undone | Enabling autostart always calls `AutoLaunch::enable`, even when Run already matches | Exact Run-value no-op |
| #1437 | Admin autostart triggers UAC/uses a one-time task | `admin.rs` creates `/SC ONCE /ST 00:00` | Use `/SC ONLOGON` and coordinate duplicate Run entry |
| #1436, #1431, #1402 | WPS/PowerPoint text copied as image | Default priority is files > image > HTML/RTF/text; Office exposes an image representation too | Source-aware text priority for Office/WPS |
| #1444 | Choosing a recovered old data directory wipes it | `copy_storage_data` deletes every target content directory before copying | Adopt a valid existing data directory |
| #1400 | Preview reports `underlying handle is not available` | Preview is also destroyable and uses native window calls immediately after rebuild | Await WebView usability |
| #1364, #1432 | Some copies intermittently absent | Watcher treats first `Ok(None)` as final despite delayed renderers | Bounded Windows empty-read retry |
| #1335 | Pinned window makes Space stop working in other apps | Navigation hook remains enabled while a pinned window stays visible | Disable hook after outside click; reenable on inside click |

## Current Code Already Contains Mitigations

| Issue | Observation |
| --- | --- |
| #894 | Shortcut recorder/validation now filters unsupported Win shortcuts; comments confirm Win key was intentionally excluded |
| #1398 | Editable focus temporarily disables Windows navigation hooks, so focused search/IME input receives native keys |
| #1246 | Clipboard reads are already retried on transient errors; no direct code path mutates Snipping Tool output files |
| #1402 part 1 | Hidden/scroll-away clipboard list updates are intentionally deferred; this is refresh policy, not confirmed data loss |

## Not Confirmable From Code Alone

| Issue | Reason |
| --- | --- |
| #1434 | No reproduction detail; current global navigation should only receive keys while the clipboard window is the intended target after the pinned-window fix |
| #1468 | Startup latency depends on machine, WebView2, database size, and app initialization; no actionable failure path |
| #1380 | Defender false positive/signing is external distribution policy |
| #1386 | App enumeration covers running/excluded apps through the registry; missing icons depend on specific app packaging, not enough examples |
| #1396 | React error #185 needs the failing route/state and console stack; current early-window race fixes may remove some causes but do not prove this bug fixed |
| #1123 | WeChat-specific image color corruption is likely native clipboard DIB interoperability; no local WeChat environment to validate a safe format change |
| #1208 | Issue platform is macOS, not Windows |
| #1277, #1098, #1199 | macOS-specific |

## Explicitly Excluded

- #1464, #1459, #1463: updater/domain availability.
- #1455: Windows 7 is below the current Tauri/WebView2 support baseline and would
  require bundling an obsolete runtime.
- Feature requests and product redesigns are outside this bug-fix task.

## Verification Record

- Windows test instance used an isolated application identifier and temporary
  data root.
- Captured two text entries and verified list rendering.
- Right-clicked an entry, hid the menu, waited for `DestroyWhenIdle`, and
  successfully rebuilt/displayed the menu on the next right-click.
- Opened Space preview, hid it, waited for `DestroyWhenIdle`, and successfully
  rebuilt/displayed the preview on the next Space press.
- The isolated test data was removed and the previously running installed app
  was restarted after validation.

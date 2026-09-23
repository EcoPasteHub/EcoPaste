# Image paste behavior

Selecting an image category filters the list. With a selected item and list/filter focus, Enter invokes paste, not copy. A first Enter inside the search input still hands keyboard focus back to the list; normal text inputs retain native editing.

Mouse behavior remains configurable in Preferences: choose `singleClickPaste` for a single left click to paste. The copy action icon continues to copy explicitly. Existing user settings are not globally changed by this fix.

On macOS, the clipboard panel remembers the external application active when it opens. Paste returns focus to a live external destination, waits for the main-thread panel handoff and actual foreground state, then posts the paste keystroke. Missing/terminated targets or failed activation produce an actionable error instead of silently posting into EcoPaste. Accessibility permission is still required.

Images written to the clipboard may be re-encoded by the OS or clipboard library. The temporary writeback guard compares dimensions and decoded RGBA pixels before the watcher stores another PNG. Stored history hashes and original images are unchanged; pre-existing duplicates are retained.

Validation: PNG re-encoding regression RED/GREEN; different-image and expiration checks; native clipboard image round trip with saved/restored clipboard formats; seven target/session policy regressions; eight synthetic browser interaction cases; full Rust and frontend checks. A native paste receipt test additionally requires the installed application's accessibility authorization and an external receiver that accepts images.

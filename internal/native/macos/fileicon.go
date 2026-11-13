//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

// FileIconPNG returns a PNG representation of a path's Finder icon.
// AppKit supplies a generic icon for missing paths, so callers should only
// request an icon for a path that still exists.
func FileIconPNG(path string) []byte {
	if path == "" {
		return nil
	}
	var out []byte
	onMain(func() {
		workspace := objc.ID(class("NSWorkspace")).Send(sel("sharedWorkspace"))
		image := objc.Send[objc.ID](workspace, sel("iconForFile:"), nsString(path))
		if image == 0 {
			return
		}
		tiff := objc.Send[objc.ID](image, sel("TIFFRepresentation"))
		if tiff == 0 {
			return
		}
		out = tiffToPNG(dataBytes(tiff))
	})
	return out
}

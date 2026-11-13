//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

// AppIconPNG returns a PNG representation of a running application's icon.
// The caller is responsible for caching it; AppKit may need to load icon
// resources on first use.
func AppIconPNG(bundleID string) []byte {
	if bundleID == "" {
		return nil
	}
	var out []byte
	onMain(func() {
		running := objc.ID(class("NSRunningApplication")).
			Send(sel("runningApplicationsWithBundleIdentifier:"), nsString(bundleID))
		if running == 0 || objc.Send[uint](running, sel("count")) == 0 {
			return
		}
		app := running.Send(sel("objectAtIndex:"), uint(0))
		if app == 0 {
			return
		}
		image := objc.Send[objc.ID](app, sel("icon"))
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

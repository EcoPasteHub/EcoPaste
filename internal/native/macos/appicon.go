//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

// iconPixels matches the cached app and file icon edge. Rendering the
// image into a small bitmap avoids encoding every representation of the
// source icon.
const iconPixels = 128

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
		out = imagePNG(objc.Send[objc.ID](app, sel("icon")), iconPixels)
	})
	return out
}

// AppIconForPath returns a PNG representation of an app bundle's icon,
// which also covers apps that are not currently running.
func AppIconForPath(path string) []byte {
	if path == "" {
		return nil
	}
	var out []byte
	onMain(func() {
		image := objc.Send[objc.ID](objc.ID(class("NSWorkspace")).Send(sel("sharedWorkspace")),
			sel("iconForFile:"), nsString(path))
		out = imagePNG(image, iconPixels)
	})
	return out
}

func imagePNG(image objc.ID, edge int) []byte {
	if image == 0 || edge <= 0 {
		return nil
	}
	rep := objc.ID(class("NSBitmapImageRep")).Send(sel("alloc")).Send(
		sel("initWithBitmapDataPlanes:pixelsWide:pixelsHigh:bitsPerSample:samplesPerPixel:hasAlpha:isPlanar:colorSpaceName:bytesPerRow:bitsPerPixel:"),
		objc.ID(0), int64(edge), int64(edge), int64(8), int64(4), true, false,
		nsString("NSDeviceRGBColorSpace"), int64(0), int64(0))
	if rep == 0 {
		return nil
	}
	defer rep.Send(sel("release"))
	size := CGSize{Width: float64(edge), Height: float64(edge)}
	rep.Send(sel("setSize:"), size)
	context := objc.ID(class("NSGraphicsContext")).Send(sel("graphicsContextWithBitmapImageRep:"), rep)
	if context == 0 {
		return nil
	}
	objc.ID(class("NSGraphicsContext")).Send(sel("saveGraphicsState"))
	objc.ID(class("NSGraphicsContext")).Send(sel("setCurrentContext:"), context)
	image.Send(sel("drawInRect:fromRect:operation:fraction:"),
		CGRect{Size: size}, CGRect{}, int64(2), 1.0)
	objc.ID(class("NSGraphicsContext")).Send(sel("restoreGraphicsState"))
	properties := objc.ID(class("NSDictionary")).Send(sel("dictionary"))
	return dataBytes(rep.Send(sel("representationUsingType:properties:"), 4, properties))
}

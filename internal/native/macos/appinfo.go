//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

// AppVersion is the bundle's short version, empty where the app runs
// without an Info.plist.
func AppVersion() string {
	bundle := objc.ID(class("NSBundle")).Send(sel("mainBundle"))
	info := bundle.Send(sel("infoDictionary"))
	if info == 0 {
		return ""
	}
	version := info.Send(sel("objectForKey:"), nsString("CFBundleShortVersionString"))
	return goString(version)
}

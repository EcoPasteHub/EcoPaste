//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

// SourceURL is EcoPaste's public source repository.
const SourceURL = "https://github.com/EcoPasteHub/EcoPaste"

// IssuesURL is EcoPaste's public issue tracker.
const IssuesURL = SourceURL + "/issues"

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

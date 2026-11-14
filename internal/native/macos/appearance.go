//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

// SetAppearance applies "auto", "light" or "dark" to every app window.
func SetAppearance(theme string) error {
	onMain(func() {
		var appearance objc.ID
		switch theme {
		case "light":
			appearance = namedAppearance("NSAppearanceNameAqua")
		case "dark":
			appearance = namedAppearance("NSAppearanceNameDarkAqua")
		}
		app := objc.ID(class("NSApplication")).Send(sel("sharedApplication"))
		app.Send(sel("setAppearance:"), appearance)
	})
	return nil
}

func namedAppearance(name string) objc.ID {
	return objc.ID(class("NSAppearance")).Send(sel("appearanceNamed:"), nsString(name))
}

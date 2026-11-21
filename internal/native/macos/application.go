//go:build darwin

package macos

import (
	"log/slog"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

const (
	activationPolicyRegular   = 0
	activationPolicyAccessory = 1
)

// SetDockIconVisible switches between a regular app with a Dock icon and
// a menu-bar accessory app without one.
func SetDockIconVisible(visible bool) error {
	ok := false
	onMain(func() {
		policy := int64(activationPolicyAccessory)
		if visible {
			policy = activationPolicyRegular
		}
		app := objc.ID(class("NSApplication")).Send(sel("sharedApplication"))
		if objc.Send[int64](app, sel("activationPolicy")) == policy {
			ok = true
			return
		}
		ok = objc.Send[bool](app, sel("setActivationPolicy:"), policy)
	})
	if !ok {
		return i18n.Error(i18n.ErrorDockIcon)
	}
	return nil
}

// syncSettingsDockIcon keeps the Dock icon tied to the settings window: it
// shows while any settings window is open and drops once the last closes.
func syncSettingsDockIcon(visible bool) {
	if err := SetDockIconVisible(visible); err != nil {
		slog.Error("switch dock icon", "module", "native.macos.application", "error", err)
	}
}

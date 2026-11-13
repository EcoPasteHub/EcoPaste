//go:build darwin

package macos

import (
	"fmt"

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
		return fmt.Errorf("%s", i18n.T(i18n.ErrorDockIcon))
	}
	return nil
}

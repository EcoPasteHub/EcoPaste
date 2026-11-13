//go:build darwin

package macos

import (
	"fmt"

	"github.com/ebitengine/purego/objc"
)

const (
	activationPolicyRegular   = 0
	activationPolicyAccessory = 1
)

// SetDockIconVisible switches between a regular app with a Dock icon and
// a menu-bar accessory app without one.
func SetDockIconVisible(visible bool) error {
	changed := false
	onMain(func() {
		policy := activationPolicyAccessory
		if visible {
			policy = activationPolicyRegular
		}
		app := objc.ID(class("NSApplication")).Send(sel("sharedApplication"))
		changed = objc.Send[bool](app, sel("setActivationPolicy:"), int64(policy))
	})
	if !changed {
		return fmt.Errorf("无法切换 Dock 图标")
	}
	return nil
}

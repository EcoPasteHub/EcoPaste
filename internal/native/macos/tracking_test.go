//go:build darwin

package macos

import (
	"testing"

	"github.com/ebitengine/purego/objc"
)

func TestControlsKeepOneHoverTrackingArea(t *testing.T) {
	onMain(func() {
		registerPanelOnce.Do(registerPanelTargets)
		registerTextInputOnce.Do(registerTextInputTargets)
		registerTextAreaOnce.Do(registerTextAreaTargets)
		registerSettingsPopupClass()
		registerSettingsInfoOnce.Do(registerSettingsInfoTargets)
		registerNumberInputOnce.Do(registerNumberInputTargets)
		registerShortcutRecorderOnce.Do(registerShortcutRecorderTargets)
		classes := []objc.Class{
			iconButtonClass, navCapsuleClass, settingsInfoButtonClass,
			numberInputClass, numberStepButtonClass, textInputClass,
			textAreaClass, settingsPopupClass, shortcutRecorderClass,
		}
		for _, controlClass := range classes {
			view := objc.ID(controlClass).Send(sel("new"))
			for range 100 {
				view.Send(sel("updateTrackingAreas"))
			}
			areas := view.Send(sel("trackingAreas"))
			count := objc.Send[uint](areas, sel("count"))
			owned := 0
			for index := uint(0); index < count; index++ {
				area := areas.Send(sel("objectAtIndex:"), index)
				if objc.Send[objc.ID](area, sel("owner")) == view &&
					objc.Send[uint64](area, sel("options")) == hoverTrackingOptions {
					owned++
				}
			}
			if owned != 1 {
				t.Errorf("class %v retained %d hover areas after 100 updates", controlClass, owned)
			}
			view.Send(sel("release"))
		}
	})
}

func TestHoverTrackingPreservesOtherTrackingAreas(t *testing.T) {
	onMain(func() {
		view := objc.ID(class("NSView")).Send(sel("new"))
		defer view.Send(sel("release"))
		other := objc.ID(class("NSTrackingArea")).Send(sel("alloc")).
			Send(sel("initWithRect:options:owner:userInfo:"), CGRect{}, uint64(2|128|512), view, objc.ID(0))
		view.Send(sel("addTrackingArea:"), other)
		other.Send(sel("release"))
		for range 100 {
			ensureHoverTracking(view)
		}
		if count := objc.Send[uint](view.Send(sel("trackingAreas")), sel("count")); count != 2 {
			t.Errorf("tracking area count = %d, want one hover and one existing area", count)
		}
	})
}

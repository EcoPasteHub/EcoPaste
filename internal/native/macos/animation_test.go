//go:build darwin

package macos

import (
	"testing"

	"github.com/ebitengine/purego/objc"
)

func TestAnimateInKeepsViewVisibleAndAddsFade(t *testing.T) {
	view := objc.ID(class("NSView")).Send(sel("new"))
	defer view.Send(sel("release"))

	animateIn(view, 0.2)
	if alpha := objc.Send[float64](view, sel("alphaValue")); alpha != 1 {
		t.Fatalf("alpha = %v, want 1", alpha)
	}
	if reduceMotionEnabled() {
		return
	}
	layer := view.Send(sel("layer"))
	if layer == 0 {
		t.Fatal("animated view has no layer")
	}
	if animation := layer.Send(sel("animationForKey:"), nsString("fadeIn")); animation == 0 {
		t.Fatal("fade animation was not added")
	}
}

//go:build darwin

package macos

import (
	"testing"

	"github.com/ebitengine/purego/objc"
)

func TestTransitionEasesTowardTarget(t *testing.T) {
	tr := transition{target: 1}
	for i := 0; i < 240 && tr.value != tr.target; i++ {
		if tr.advance() {
			break
		}
	}
	if tr.value != 1 {
		t.Fatalf("value = %v, want 1", tr.value)
	}
}

func TestTransitionEasesBackToIdle(t *testing.T) {
	tr := transition{value: 1}
	for i := 0; i < 240 && tr.value != tr.target; i++ {
		if tr.advance() {
			break
		}
	}
	if tr.value != 0 {
		t.Fatalf("value = %v, want 0", tr.value)
	}
}

func TestTransitionSnapJumpsToTarget(t *testing.T) {
	tr := transition{value: 0.4, target: 1}
	tr.snap(0)
	if tr.value != 0 || tr.target != 0 {
		t.Fatalf("snap left value=%v target=%v, want both 0", tr.value, tr.target)
	}
}

// TestTransitionRetainsViewWhileAnimating guards the crash where the shared
// frame driver repainted an NSTableCellView the list had already released. The
// driver must hold its own reference for as long as the transition is queued.
func TestTransitionRetainsViewWhileAnimating(t *testing.T) {
	view := objc.ID(class("NSView")).Send(sel("new"))
	tr := newViewTransition(view)
	before := int(objc.Send[uint64](view, sel("retainCount")))

	requestTransitionFrame(tr)
	if got := int(objc.Send[uint64](view, sel("retainCount"))); got != before+1 {
		releaseTransitionFrame(tr)
		view.Send(sel("release"))
		t.Fatalf("retain count while animating = %d, want %d", got, before+1)
	}

	// The owner drops its reference mid-animation; the driver keeps the view
	// alive until it stops.
	view.Send(sel("release"))
	releaseTransitionFrame(tr)
	// The releaseTransitionFrame call released the final reference, so the
	// view is gone now and must not be touched.
}

// TestAdvanceTransitionAfterOwnerRelease reproduces the row-hover crash: the
// list drops the cell while its hover fade is still queued, then the frame
// driver ticks. The retained view must stay valid for every frame.
func TestAdvanceTransitionAfterOwnerRelease(t *testing.T) {
	view := objc.ID(class("NSView")).Send(sel("new"))
	redraws := 0
	tr := &transition{view: view, redraw: func() {
		redraws++
		view.Send(sel("setNeedsDisplay:"), true)
	}}
	tr.setTarget(1)
	// The table releases the cell while the fade is running.
	view.Send(sel("release"))

	for i := 0; i < 240 && len(transitionFrames) > 0; i++ {
		advanceTransitionFrames()
	}
	if redraws == 0 {
		t.Fatal("transition never repainted")
	}
	if len(transitionFrames) != 0 {
		t.Fatalf("transition did not settle: %d frames left", len(transitionFrames))
	}
	// The driver released the final reference, so the view is gone now.
}

// TestClearRowHoverStatesReleasesViews covers the row-hover path: reloading
// the history list drops the hover transitions before the cells go away.
func TestClearRowHoverStatesReleasesViews(t *testing.T) {
	view := objc.ID(class("NSView")).Send(sel("new"))
	before := int(objc.Send[uint64](view, sel("retainCount")))

	setHistoryRowHovered(view, true)
	if got := int(objc.Send[uint64](view, sel("retainCount"))); got != before+1 {
		clearRowHoverStates()
		view.Send(sel("release"))
		t.Fatalf("retain count while hovered = %d, want %d", got, before+1)
	}

	clearRowHoverStates()
	if _, ok := rowHoverStates[view]; ok {
		view.Send(sel("release"))
		t.Fatal("clearRowHoverStates kept the row transition")
	}
	if got := int(objc.Send[uint64](view, sel("retainCount"))); got != before {
		view.Send(sel("release"))
		t.Fatalf("retain count after clear = %d, want %d", got, before)
	}
	view.Send(sel("release"))
}

func TestFadeColorScalesAlpha(t *testing.T) {
	white := objc.ID(classNSColor).Send(sel("whiteColor"))
	if alpha := objc.Send[float64](fadeColor(white, 1), sel("alphaComponent")); alpha != 1 {
		t.Fatalf("opaque alpha = %v, want 1", alpha)
	}
	if alpha := objc.Send[float64](fadeColor(white, 0), sel("alphaComponent")); alpha != 0 {
		t.Fatalf("clear alpha = %v, want 0", alpha)
	}
	if alpha := objc.Send[float64](fadeColor(white, 0.5), sel("alphaComponent")); alpha < 0.4 || alpha > 0.6 {
		t.Fatalf("half alpha = %v, want ~0.5", alpha)
	}
}

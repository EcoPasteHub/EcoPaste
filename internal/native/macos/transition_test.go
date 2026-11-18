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

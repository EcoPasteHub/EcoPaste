//go:build darwin

package macos

import (
	"math"
	"sync"

	"github.com/ebitengine/purego/objc"
)

// Every hover, focus and active transition in the app shares one easing
// curve and one frame timer. Sharing the driver keeps unrelated controls
// moving in lockstep and stops each view from scheduling its own timer.
const (
	transitionFrameInterval = 1.0 / 60.0
	// transitionEasing is the fraction of the remaining distance covered per
	// frame. It lands a 0 -> 1 change in roughly 120ms: light and responsive,
	// without the delay a slower curve would add to hover feedback.
	transitionEasing = 0.28
	// transitionSnap is the remaining distance at which the value jumps to
	// the target, avoiding an endless tail of sub-pixel changes.
	transitionSnap = 0.02
	// transitionSettle is the distance used to decide whether a value still
	// needs frames.
	transitionSettle = 0.001
)

// transition eases a scalar value toward a target. Values are usually 0
// (idle) to 1 (hovered/focused), but any range works. redraw is invoked
// once per frame while the value is moving, and should repaint the owning
// view.
//
// view is the AppKit view redraw paints. The shared driver retains it while
// the transition animates: a view can be owned by something that drops it
// mid-animation (an NSTableView recycles and releases its cells as rows
// reload), and a timed redraw into a freed view crashes the process. Retain
// holds the object alive until the transition settles or stops, then
// releases it.
type transition struct {
	value  float64
	target float64
	redraw func()
	view   objc.ID
	held   bool
}

// newViewTransition builds the common transition whose only repaint is a
// redraw of view. It keeps the view alive while animating.
func newViewTransition(view objc.ID) *transition {
	return &transition{view: view, redraw: func() { view.Send(sel("setNeedsDisplay:"), true) }}
}

// setTarget starts (or continues) the animation toward target. Setting the
// current target again is a no-op so repeated mouse events stay cheap.
func (t *transition) setTarget(target float64) {
	if t.target == target {
		return
	}
	t.target = target
	requestTransitionFrame(t)
}

// snap jumps straight to target and cancels any running animation. Use it
// when a control must not animate, such as a popup collapsing as its native
// menu opens.
func (t *transition) snap(target float64) {
	t.target = target
	t.value = target
	// Repaint before releasing: if the driver was the only thing keeping the
	// view alive, releasing first would free it before this final redraw.
	t.redrawView()
	releaseTransitionFrame(t)
}

// stop cancels the animation without changing the value. Call it when the
// owning view is torn down so the global driver drops the reference.
func (t *transition) stop() {
	releaseTransitionFrame(t)
}

func (t *transition) redrawView() {
	if t.redraw != nil {
		t.redraw()
	}
}

// advance moves one frame toward the target and reports whether the value
// has settled.
func (t *transition) advance() bool {
	delta := t.target - t.value
	if math.Abs(delta) < transitionSnap {
		t.value = t.target
	} else {
		t.value += delta * transitionEasing
	}
	return math.Abs(t.value-t.target) <= transitionSettle
}

var (
	registerTransitionTickerOnce sync.Once
	transitionTickerTarget       objc.ID
	transitionTimer              objc.ID
	transitionFrames             = map[*transition]struct{}{}
)

// requestTransitionFrame adds t to the shared driver and starts the frame
// timer if it is not already running.
func requestTransitionFrame(t *transition) {
	registerTransitionTickerOnce.Do(registerTransitionTicker)
	t.retainView()
	transitionFrames[t] = struct{}{}
	if transitionTimer != 0 {
		return
	}
	transitionTimer = objc.ID(class("NSTimer")).Send(
		sel("scheduledTimerWithTimeInterval:target:selector:userInfo:repeats:"),
		transitionFrameInterval, transitionTickerTarget, sel("transitionTick:"), objc.ID(0), true)
}

func releaseTransitionFrame(t *transition) {
	delete(transitionFrames, t)
	t.releaseView()
	if len(transitionFrames) == 0 {
		stopTransitionTimer()
	}
}

// retainView keeps the repainted view alive while the driver animates it. It is
// idempotent so repeated setTarget calls while a frame is already pending do
// not over-retain.
func (t *transition) retainView() {
	if t.view == 0 || t.held {
		return
	}
	t.view.Send(sel("retain"))
	t.held = true
}

func (t *transition) releaseView() {
	if t.view == 0 || !t.held {
		return
	}
	t.held = false
	t.view.Send(sel("release"))
}

func stopTransitionTimer() {
	if transitionTimer == 0 {
		return
	}
	transitionTimer.Send(sel("invalidate"))
	transitionTimer = 0
}

// advanceTransitionFrames steps every active transition once, repaints its
// view and retires the ones that have settled.
func advanceTransitionFrames() {
	for t := range transitionFrames {
		settled := t.advance()
		t.redrawView()
		if settled {
			releaseTransitionFrame(t)
		}
	}
	if len(transitionFrames) == 0 {
		stopTransitionTimer()
	}
}

func registerTransitionTicker() {
	tickerClass, err := objc.RegisterClass("EcoPasteTransitionTicker", class("NSObject"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("transitionTick:"), Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) {
				advanceTransitionFrames()
			}},
		})
	if err != nil {
		panic(err)
	}
	transitionTickerTarget = objc.ID(tickerClass).Send(sel("new"))
}

// blendColors interpolates from one NSColor to another. It is the shared way
// to fade a tint between semantic colors.
func blendColors(from, to objc.ID, fraction float64) objc.ID {
	if fraction <= 0 {
		return from
	}
	if fraction >= 1 {
		return to
	}
	return from.Send(sel("blendedColorWithFraction:ofColor:"), fraction, to)
}

// fadeColor scales a color's own alpha by fraction, so a fill can fade in and
// out without hard-coding its base opacity.
func fadeColor(color objc.ID, fraction float64) objc.ID {
	if fraction <= 0 {
		return color.Send(sel("colorWithAlphaComponent:"), 0.0)
	}
	if fraction >= 1 {
		return color
	}
	clear := color.Send(sel("colorWithAlphaComponent:"), 0.0)
	return blendColors(clear, color, fraction)
}

package app

import (
	"testing"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"
)

// TestHotKeyActionRunsOnGraphGoroutine covers the Carbon callback path: the
// action must not run inline on the callback's goroutine, where NSGo window
// state belongs to another goroutine.
func TestHotKeyActionRunsOnGraphGoroutine(t *testing.T) {
	var scope *core.Scope
	app := nsgotest.StartApp(t, func(cx *core.Scope) { scope = cx })
	keys := &hotKeySet{app: scope}

	ran := false
	keys.onGraph(func() { ran = true })()
	if ran {
		t.Fatal("hotkey action ran on the callback goroutine instead of the graph's")
	}
	app.Settle()
	if !ran {
		t.Fatal("hotkey action never reached the graph goroutine")
	}
}

func TestHotKeyActionWithoutScopeRunsInline(t *testing.T) {
	ran := false
	(&hotKeySet{}).onGraph(func() { ran = true })()
	if !ran {
		t.Fatal("hotkey action did not run without an app scope")
	}
}

package ui

import (
	"github.com/nsgo-dev/nsgo/core"
)

// ClipboardPanel builds the history panel. hide closes the panel after
// an action that the user configured to close it.
func ClipboardPanel(model HistoryModel, hide func()) func(*core.Scope) core.Node {
	return func(cx *core.Scope) core.Node {
		panel := newHistoryPanel(cx, model, hide)
		panel.start()
		return panel.node()
	}
}

//go:build darwin

package macos

import (
	"testing"

	"github.com/ebitengine/purego/objc"
)

func TestSettingsCloseReleasesOwnedControls(t *testing.T) {
	onMain(func() {
		pool := objc.ID(class("NSAutoreleasePool")).Send(sel("new"))
		view := newSettings(settingsEmitterFunc(func(string, any) {}))
		controls := map[string]objc.ID{"shortcut": view.shortcutRecorders[0].control}
		for _, field := range view.fields {
			controls[field.kind] = field.control
		}
		for _, control := range controls {
			control.Send(sel("retain"))
		}
		view.Close()
		pool.Send(sel("drain"))
		for name, control := range controls {
			if count := objc.Send[uint](control, sel("retainCount")); count != 1 {
				t.Errorf("%s retains %d references after closing settings, want only the test reference", name, count)
			}
			control.Send(sel("release"))
		}
	})
}

func TestHistoryRowReleasesLayoutWhenCellDies(t *testing.T) {
	onMain(func() {
		if err := registerHistoryRowViews(); err != nil {
			t.Error(err)
			return
		}
		for range 100 {
			cell := objc.ID(historyRowClass).Send(sel("new"))
			storeRowLayout(cell, &rowLayout{titleText: "retained row content"})
			cell.Send(sel("release"))
			if rowLayoutFor(cell) != nil {
				t.Error("released cell retained its Go layout")
				rowLayouts.Delete(cell)
				return
			}
		}
	})
}

func TestPanelCloseReleasesOwnedChildren(t *testing.T) {
	onMain(func() {
		pool := objc.ID(class("NSAutoreleasePool")).Send(sel("new"))
		panel := NewPanel(nil).(*Panel)
		children := map[string]objc.ID{
			"table":          panel.table,
			"preview scroll": panel.previewScroll,
			"preview image":  panel.previewImage,
		}
		for _, child := range children {
			child.Send(sel("retain"))
		}
		panel.Close()
		pool.Send(sel("drain"))
		for name, child := range children {
			if count := objc.Send[uint](child, sel("retainCount")); count != 1 {
				t.Errorf("%s retains %d references after closing panel, want only the test reference", name, count)
			}
			child.Send(sel("release"))
		}
	})
}

func TestNativeDragEndCancelsFallback(t *testing.T) {
	onMain(func() {
		panel := NewPanel(nil).(*Panel)
		defer panel.Close()
		before := objc.Send[uint](panel.table, sel("retainCount"))
		panel.dragging = true
		historyTarget.Send(sel("performSelector:withObject:afterDelay:"),
			sel("checkDragFinished:"), panel.table, 3600.0)
		if count := objc.Send[uint](panel.table, sel("retainCount")); count <= before {
			t.Error("drag fallback did not retain its table argument")
			return
		}
		historyTargetDragEnded(historyTarget, 0, panel.table, 0, CGPoint{}, 0)
		if panel.dragging {
			t.Error("native drag end left the panel dragging")
		}
		if count := objc.Send[uint](panel.table, sel("retainCount")); count != before {
			t.Errorf("native drag end left a fallback retaining the table: %d, want %d", count, before)
		}
	})
}

func TestPanelCancelsQueuedClickActions(t *testing.T) {
	for _, action := range []string{"disable single click", "hide"} {
		t.Run(action, func(t *testing.T) {
			onMain(func() {
				panel := NewPanel(nil).(*Panel)
				defer panel.Close()
				before := objc.Send[uint](panel.table, sel("retainCount"))
				panel.behavior.SingleClick = true
				panel.pendingClick = "pending"
				panel.table.Send(sel("performSelector:withObject:afterDelay:"),
					sel("activatePendingClick:"), objc.ID(0), 3600.0)
				if count := objc.Send[uint](panel.table, sel("retainCount")); count <= before {
					t.Error("click action did not retain its table target")
					return
				}
				if action == "hide" {
					panel.Command("cancelInteractions", nil)
				} else {
					setPanelBehavior(panel, PanelBehavior{})
				}
				if panel.pendingClick != "" {
					t.Error("canceled click retained its action")
				}
				if count := objc.Send[uint](panel.table, sel("retainCount")); count != before {
					t.Errorf("canceled click left a task retaining the table: %d, want %d", count, before)
				}
			})
		})
	}
}

//go:build darwin

package macos

import (
	"testing"

	"github.com/ebitengine/purego/objc"
)

func TestSettingsSourceAppRowShowsModeAndEmitsChange(t *testing.T) {
	onMain(func() {
		var changes []SettingsChange
		view := newSettings(settingsEmitterFunc(func(_ string, data any) {
			if change, ok := data.(SettingsChange); ok {
				changes = append(changes, change)
			}
		}))
		defer view.Close()

		view.SetProp("values", SettingsValues{Apps: []SettingsApp{
			{ID: "com.example.a", Name: "Alpha", Mode: settingsAppModeBlockWake},
		}})
		if len(view.apps) != 1 || view.apps[0].Mode != settingsAppModeBlockWake {
			t.Fatalf("apps = %+v", view.apps)
		}
		if len(view.appsRowViews) != 1 {
			t.Fatalf("rows = %d", len(view.appsRowViews))
		}
		if got := objc.Send[CGRect](view.appsHost, sel("frame")).Size.Height; got != settingsAppRowHeight {
			t.Errorf("apps host height = %v", got)
		}
		if view.pageHeights["clipboard"] < settingsAppRowHeight {
			t.Errorf("clipboard page height = %v", view.pageHeights["clipboard"])
		}
		popup := view.appsRowViews[0].popup
		if popup == 0 {
			t.Error("mode popup missing")
			return
		}
		if got := int(objc.Send[int64](popup, sel("numberOfItems"))); got != len(settingsAppModeOptions()) {
			t.Errorf("popup items = %d", got)
			return
		}
		options := settingsAppModeOptions()
		index := int(objc.Send[int64](popup, sel("indexOfSelectedItem")))
		if index < 0 || options[index].Value != settingsAppModeBlockWake {
			t.Errorf("selected mode = %d", index)
			return
		}
		name := view.appsRowViews[0].view.Send(sel("viewWithTag:"), int64(settingsAppNameTag))
		if name == 0 || goString(name.Send(sel("stringValue"))) != "Alpha" {
			t.Error("app name not rendered")
			return
		}

		popup.Send(sel("selectItemAtIndex:"), int64(3))
		settingsAppsTarget.Send(sel("appModeChanged:"), popup)
		if len(changes) != 1 {
			t.Errorf("changes = %+v", changes)
			return
		}
		change := changes[0]
		selection, ok := change.Value.(SettingsAppSelection)
		if !ok || change.Path != "sourceApps.mode" {
			t.Errorf("change = %+v", change)
			return
		}
		if selection.ID != "com.example.a" || selection.Name != "Alpha" || selection.Mode != settingsAppModeBlockCaptureAndWake {
			t.Errorf("selection = %+v", selection)
		}
	})
}

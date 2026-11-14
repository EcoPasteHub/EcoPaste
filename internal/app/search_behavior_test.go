package app

import (
	"path/filepath"
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

func TestSearchBehaviorReadsSettings(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	model := &historyModel{store: store}
	behavior := model.SearchBehavior()
	if behavior.DefaultFocus || !behavior.ClearOnHide {
		t.Fatalf("default behavior = %+v", behavior)
	}
	if _, err := store.Update(map[string]any{
		"clipboard": map[string]any{
			"search": map[string]any{"defaultFocus": true, "clearOnHide": false},
		},
	}); err != nil {
		t.Fatal(err)
	}
	behavior = model.SearchBehavior()
	if !behavior.DefaultFocus || behavior.ClearOnHide {
		t.Fatalf("updated behavior = %+v", behavior)
	}
}

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
	if !behavior.ClearOnHide {
		t.Fatalf("default behavior = %+v", behavior)
	}
	if _, err := store.Update(map[string]any{
		"clipboard": map[string]any{
			"search": map[string]any{"clearOnHide": false},
		},
	}); err != nil {
		t.Fatal(err)
	}
	behavior = model.SearchBehavior()
	if behavior.ClearOnHide {
		t.Fatalf("updated behavior = %+v", behavior)
	}
}

func TestSearchBehaviorFallsBackToDefaults(t *testing.T) {
	behavior := (&historyModel{}).SearchBehavior()
	expected := settings.Default().Clipboard.Search
	if behavior.ClearOnHide != expected.ClearOnHide {
		t.Fatalf("fallback behavior = %+v", behavior)
	}
}

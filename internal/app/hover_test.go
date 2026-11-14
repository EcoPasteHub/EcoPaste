package app

import (
	"path/filepath"
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

func TestQuickActionsKeepConfiguredOrder(t *testing.T) {
	root := t.TempDir()
	store, err := settings.Open(filepath.Join(root, settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(map[string]any{
		"clipboard": map[string]any{"content": map[string]any{
			"itemActions": []any{"delete", "copy", "note", "star"},
			"itemActionOrder": []any{
				"paste", "copy", "note", "star", "pinItem", "delete",
			},
		}},
	}); err != nil {
		t.Fatal(err)
	}
	actions := (&historyModel{store: store}).QuickActions()
	want := []string{"copy", "note", "star", "delete"}
	if len(actions) != len(want) {
		t.Fatalf("actions = %v", actions)
	}
	for index, action := range want {
		if actions[index] != action {
			t.Errorf("actions = %v", actions)
			break
		}
	}
}

func TestQuickActionsFallBackToDefaultOrder(t *testing.T) {
	root := t.TempDir()
	store, err := settings.Open(filepath.Join(root, settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(map[string]any{
		"clipboard": map[string]any{"content": map[string]any{
			"itemActions":     []any{"star", "copy"},
			"itemActionOrder": []any{},
		}},
	}); err != nil {
		t.Fatal(err)
	}
	actions := (&historyModel{store: store}).QuickActions()
	if len(actions) != 2 || actions[0] != "copy" || actions[1] != "star" {
		t.Errorf("actions = %v", actions)
	}
}

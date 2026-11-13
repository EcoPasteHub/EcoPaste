package app

import (
	"path/filepath"
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/settings"
	"github.com/EcoPasteHub/EcoPaste/internal/ui"
)

func openBehaviorWith(t *testing.T, window map[string]any) ui.HistoryOpenBehavior {
	t.Helper()
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	if window != nil {
		if _, err := store.Update(map[string]any{
			"clipboard": map[string]any{"window": window},
		}); err != nil {
			t.Fatal(err)
		}
	}
	return (&historyModel{store: store}).OpenBehavior()
}

func TestOpenBehaviorDefaults(t *testing.T) {
	behavior := openBehaviorWith(t, nil)
	if !behavior.ScrollTop || behavior.Filter != "preserve" || behavior.GroupID != "preserve" {
		t.Errorf("behavior = %+v", behavior)
	}
}

func TestOpenBehaviorCategoryWins(t *testing.T) {
	behavior := openBehaviorWith(t, map[string]any{"selectCategoryOnOpen": "image"})
	if behavior.Filter != "image" {
		t.Errorf("filter = %q", behavior.Filter)
	}
}

func TestOpenBehaviorFavoriteRange(t *testing.T) {
	for _, category := range []string{"preserve", "all"} {
		behavior := openBehaviorWith(t, map[string]any{
			"selectCategoryOnOpen": category,
			"selectRangeOnOpen":    "favorite",
		})
		if behavior.Filter != "favorite" {
			t.Errorf("category %q filter = %q", category, behavior.Filter)
		}
	}
}

func TestOpenBehaviorGroup(t *testing.T) {
	behavior := openBehaviorWith(t, map[string]any{"selectGroupOnOpen": "group:work"})
	if behavior.GroupID != "work" {
		t.Errorf("group = %q", behavior.GroupID)
	}
	behavior = openBehaviorWith(t, map[string]any{"selectGroupOnOpen": "all"})
	if behavior.GroupID != "" {
		t.Errorf("group = %q", behavior.GroupID)
	}
}

func TestOpenBehaviorScrollTop(t *testing.T) {
	behavior := openBehaviorWith(t, map[string]any{"scrollToTopOnOpen": false})
	if behavior.ScrollTop {
		t.Error("scroll top stayed enabled")
	}
}

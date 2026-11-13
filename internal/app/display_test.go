package app

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
	"github.com/EcoPasteHub/EcoPaste/internal/storage"
)

func TestDisplayClampsStoredDensity(t *testing.T) {
	root := t.TempDir()
	store, err := settings.Open(filepath.Join(root, settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(map[string]any{
		"clipboard": map[string]any{"display": map[string]any{
			"textMaxLines": 0, "imageMaxHeight": 400, "fileMaxCount": 9,
		}},
	}); err != nil {
		t.Fatal(err)
	}
	display := (&historyModel{store: store}).Display()
	if display.TextMaxLines != 1 || display.ImageMaxHeight != 100 || display.FileMaxCount != 5 {
		t.Errorf("display = %+v", display)
	}
}

func TestHistoryItemHonorsFileMaxCount(t *testing.T) {
	root := t.TempDir()
	db, err := database.Open(filepath.Join(root, "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	store, err := settings.Open(filepath.Join(root, settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(map[string]any{
		"clipboard": map[string]any{"display": map[string]any{"fileMaxCount": 2}},
	}); err != nil {
		t.Fatal(err)
	}
	model := &historyModel{
		store: store,
		runtime: &clipboardRuntime{
			db:     db,
			images: storage.NewImageStore(filepath.Join(root, "resources")),
		},
	}
	item := database.Item{
		Kind:     database.KindFiles,
		Content:  "/tmp/a.txt\n/tmp/b.txt\n/tmp/c.txt\n/tmp/d.txt",
		Platform: "macos",
	}
	row := model.historyItem(context.Background(), item)
	if row.Title != "a.txt、b.txt 等 4 个" {
		t.Errorf("title = %q", row.Title)
	}
}

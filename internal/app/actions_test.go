package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
	"github.com/EcoPasteHub/EcoPaste/internal/storage"
)

func actionModel(t *testing.T) (*historyModel, *database.DB) {
	t.Helper()
	root := t.TempDir()
	db, err := database.Open(filepath.Join(root, "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	model := &historyModel{runtime: &clipboardRuntime{
		db:     db,
		images: storage.NewImageStore(filepath.Join(root, "resources")),
	}}
	return model, db
}

func TestOpenDispatchesBySubKind(t *testing.T) {
	model, db := actionModel(t)
	ctx := context.Background()
	var opened []string
	openExternal = func(raw string) error {
		opened = append(opened, raw)
		return nil
	}
	openFile = func(path string) error {
		opened = append(opened, path)
		return nil
	}
	t.Cleanup(func() {
		openExternal = macos.OpenURL
		openFile = macos.OpenPath
	})

	tests := []struct {
		text   string
		sub    database.SubKind
		expect string
	}{
		{"https://example.com", database.SubURL, "https://example.com"},
		{"  www.example.com  ", database.SubURL, "https://www.example.com"},
		{"user@example.com", database.SubEmail, "mailto:user@example.com"},
		{"  /tmp/file.txt  ", database.SubPath, "/tmp/file.txt"},
	}
	for _, tt := range tests {
		sub := tt.sub
		item := &database.Item{Kind: database.KindText, Content: tt.text, SubKind: &sub, Platform: "macos"}
		id, _, err := db.Upsert(ctx, item)
		if err != nil {
			t.Fatal(err)
		}
		if err := model.Open(ctx, id); err != nil {
			t.Fatal(err)
		}
		if got := opened[len(opened)-1]; got != tt.expect {
			t.Errorf("opened %q, want %q", got, tt.expect)
		}
	}
}

func TestRevealFilesAndPath(t *testing.T) {
	model, db := actionModel(t)
	ctx := context.Background()
	var revealed [][]string
	revealFiles = func(paths []string) error {
		revealed = append(revealed, paths)
		return nil
	}
	t.Cleanup(func() { revealFiles = macos.RevealPaths })

	files := &database.Item{Kind: database.KindFiles, Content: "/tmp/a\n/tmp/b", Platform: "macos"}
	id, _, err := db.Upsert(ctx, files)
	if err != nil {
		t.Fatal(err)
	}
	if err := model.Reveal(ctx, id); err != nil {
		t.Fatal(err)
	}
	sub := database.SubPath
	pathItem := &database.Item{Kind: database.KindText, Content: "/tmp/c", SubKind: &sub, Platform: "macos"}
	id, _, err = db.Upsert(ctx, pathItem)
	if err != nil {
		t.Fatal(err)
	}
	if err := model.Reveal(ctx, id); err != nil {
		t.Fatal(err)
	}
	if len(revealed) != 2 || len(revealed[0]) != 2 || revealed[1][0] != "/tmp/c" {
		t.Errorf("revealed = %v", revealed)
	}
}

func TestSaveImageWritesChosenFile(t *testing.T) {
	model, db := actionModel(t)
	ctx := context.Background()
	stored, err := model.runtime.images.StorePNG(testPNG(t))
	if err != nil {
		t.Fatal(err)
	}
	item := &database.Item{Kind: database.KindImage, Content: stored.FileName, Platform: "macos"}
	id, _, err := db.Upsert(ctx, item)
	if err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(t.TempDir(), "saved.png")
	chooseSave = func(_, _ string) (string, bool) { return dest, true }
	t.Cleanup(func() { chooseSave = macos.SavePanel })
	if err := model.SaveImage(ctx, id); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	want, err := model.runtime.images.Read(stored.FileName)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Error("saved image differs")
	}
}

func TestEditNoteNormalizesAndAutoFavorites(t *testing.T) {
	model, db := actionModel(t)
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(map[string]any{
		"clipboard": map[string]any{"content": map[string]any{"autoFavorite": true}},
	}); err != nil {
		t.Fatal(err)
	}
	model.store = store

	ctx := context.Background()
	item := &database.Item{Kind: database.KindText, Content: "hello", Platform: "macos"}
	id, _, err := db.Upsert(ctx, item)
	if err != nil {
		t.Fatal(err)
	}
	before, err := db.FindByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	editNote = func(string) (string, bool) { return "  a useful note  ", true }
	t.Cleanup(func() { editNote = macos.EditNote })

	saved, err := model.EditNote(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if !saved {
		t.Error("note was not saved")
	}
	after, err := db.FindByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if after.Note == nil || *after.Note != "a useful note" {
		t.Errorf("note = %v", after.Note)
	}
	if !after.IsFavorite {
		t.Error("note did not auto-favorite")
	}
	if !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Errorf("updated_at moved: %v -> %v", before.UpdatedAt, after.UpdatedAt)
	}
}

func TestEditNoteCancelAndClear(t *testing.T) {
	model, db := actionModel(t)
	ctx := context.Background()
	note := "existing"
	item := &database.Item{Kind: database.KindText, Content: "hello", Note: &note, Platform: "macos"}
	id, _, err := db.Upsert(ctx, item)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { editNote = macos.EditNote })

	editNote = func(string) (string, bool) { return "", false }
	saved, err := model.EditNote(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if saved {
		t.Error("cancel reported a save")
	}
	found, err := db.FindByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if found.Note == nil || *found.Note != note {
		t.Errorf("cancel changed note: %v", found.Note)
	}

	editNote = func(string) (string, bool) { return "   ", true }
	saved, err = model.EditNote(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if !saved {
		t.Error("clear was not saved")
	}
	found, err = db.FindByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if found.Note != nil {
		t.Errorf("note = %v, want nil", found.Note)
	}
}

func TestEditNoteLimitsLength(t *testing.T) {
	model, db := actionModel(t)
	ctx := context.Background()
	item := &database.Item{Kind: database.KindText, Content: "hello", Platform: "macos"}
	id, _, err := db.Upsert(ctx, item)
	if err != nil {
		t.Fatal(err)
	}
	editNote = func(string) (string, bool) { return strings.Repeat("备", noteMaxChars+10), true }
	t.Cleanup(func() { editNote = macos.EditNote })
	if _, err := model.EditNote(ctx, id); err != nil {
		t.Fatal(err)
	}
	found, err := db.FindByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if found.Note == nil || len([]rune(*found.Note)) != noteMaxChars {
		t.Errorf("note length = %d", len([]rune(*found.Note)))
	}
}

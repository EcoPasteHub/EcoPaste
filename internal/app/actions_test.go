package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
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

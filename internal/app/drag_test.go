package app

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/clipboard"
	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/storage"
)

func TestDragOffersRichText(t *testing.T) {
	root := t.TempDir()
	db, err := database.Open(filepath.Join(root, "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	model := &historyModel{runtime: &clipboardRuntime{
		db:      db,
		images:  storage.NewImageStore(filepath.Join(root, "resources")),
		service: &clipboard.Service{DB: db},
	}}
	plain := "rich text"
	sub := database.SubHTML
	id, _, err := db.Upsert(context.Background(), &database.Item{
		Kind:       database.KindText,
		SubKind:    &sub,
		Content:    "<b>rich text</b>",
		SearchText: &plain,
		Platform:   "macos",
	})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := model.Drag(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if payload.Text != plain || payload.HTML != "<b>rich text</b>" || payload.RTF != "" {
		t.Errorf("drag payload = %+v", payload)
	}
}

func TestDragOffersFiles(t *testing.T) {
	root := t.TempDir()
	db, err := database.Open(filepath.Join(root, "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	model := &historyModel{runtime: &clipboardRuntime{
		db:      db,
		images:  storage.NewImageStore(filepath.Join(root, "resources")),
		service: &clipboard.Service{DB: db},
	}}
	id, _, err := db.Upsert(context.Background(), &database.Item{
		Kind:     database.KindFiles,
		Content:  "/tmp/a.txt\n/tmp/b.txt",
		Platform: "macos",
	})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := model.Drag(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if len(payload.Files) != 2 || payload.Files[0] != "/tmp/a.txt" || payload.Text != "" {
		t.Errorf("drag payload = %+v", payload)
	}
}

func TestDragOffersImageBytes(t *testing.T) {
	root := t.TempDir()
	db, err := database.Open(filepath.Join(root, "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	images := storage.NewImageStore(filepath.Join(root, "resources"))
	png := testPNG(t)
	stored, err := images.StorePNG(png)
	if err != nil {
		t.Fatal(err)
	}
	model := &historyModel{runtime: &clipboardRuntime{
		db:      db,
		images:  images,
		service: &clipboard.Service{DB: db, Images: images},
	}}
	id, _, err := db.Upsert(context.Background(), &database.Item{
		Kind:     database.KindImage,
		Content:  stored.FileName,
		Platform: "macos",
	})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := model.Drag(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(payload.ImageData, png) {
		t.Errorf("image bytes differ: got %d, want %d", len(payload.ImageData), len(png))
	}
}

func TestDragRejectsMissingItem(t *testing.T) {
	root := t.TempDir()
	db, err := database.Open(filepath.Join(root, "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	model := &historyModel{runtime: &clipboardRuntime{
		db:      db,
		images:  storage.NewImageStore(filepath.Join(root, "resources")),
		service: &clipboard.Service{DB: db},
	}}
	if _, err := model.Drag(context.Background(), "missing"); err == nil {
		t.Fatal("missing item produced a drag payload")
	}
}

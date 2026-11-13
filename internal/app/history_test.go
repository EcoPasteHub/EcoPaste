package app

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/keystroke"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
	"github.com/EcoPasteHub/EcoPaste/internal/storage"
)

func TestAutoPasteModes(t *testing.T) {
	cases := map[settings.AutoPaste]bool{
		settings.AutoPasteDisabled:         false,
		settings.AutoPasteSingleClickCopy:  false,
		settings.AutoPasteDoubleClickCopy:  false,
		settings.AutoPasteSingleClickPaste: true,
		settings.AutoPasteDoubleClickPaste: true,
	}
	for mode, want := range cases {
		if got := autoPaste(mode); got != want {
			t.Errorf("autoPaste(%q) = %v, want %v", mode, got, want)
		}
	}
}

func TestNeedsAccessibility(t *testing.T) {
	model := &historyModel{}
	if !model.NeedsAccessibility(keystroke.ErrPermission) {
		t.Error("permission error was not recognized")
	}
	if model.NeedsAccessibility(nil) {
		t.Error("nil error asked for permission")
	}
}

func TestDeleteRemovesImageFile(t *testing.T) {
	root := t.TempDir()
	db, err := database.Open(filepath.Join(root, "db", "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	images := storage.NewImageStore(filepath.Join(root, "resources"))
	stored, err := images.StorePNG(testPNG(t))
	if err != nil {
		t.Fatal(err)
	}
	item := &database.Item{
		ID:       database.NewID(),
		Kind:     database.KindImage,
		Content:  stored.FileName,
		Platform: "macos",
	}
	id, _, err := db.Upsert(context.Background(), item)
	if err != nil {
		t.Fatal(err)
	}

	model := &historyModel{runtime: &clipboardRuntime{db: db, images: images}}
	if err := model.Delete(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(images.OriginPath(stored.FileName)); !os.IsNotExist(err) {
		t.Errorf("image still exists: %v", err)
	}
	found, err := db.FindByID(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if found != nil {
		t.Errorf("item %s still exists", id)
	}
}

func TestDeleteUnknownItemIsNoop(t *testing.T) {
	root := t.TempDir()
	db, err := database.Open(filepath.Join(root, "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	model := &historyModel{runtime: &clipboardRuntime{db: db, images: storage.NewImageStore(root)}}
	if err := model.Delete(context.Background(), "missing"); err != nil {
		t.Fatalf("delete missing item: %v", err)
	}
}

func testPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

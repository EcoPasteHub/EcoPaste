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
	"github.com/EcoPasteHub/EcoPaste/internal/storage"
)

func testPNGBytes(t *testing.T, width, height int, shade uint8) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.Set(x, y, color.RGBA{R: shade, G: shade, B: shade, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestFormatBytes(t *testing.T) {
	cases := map[int64]string{
		512:                    "512 B",
		2048:                   "2.0 KB",
		1536 * 1024:            "1.5 MB",
		3 * 1024 * 1024 * 1024: "3.0 GB",
	}
	for size, want := range cases {
		if got := formatBytes(size); got != want {
			t.Errorf("formatBytes(%d) = %q, want %q", size, got, want)
		}
	}
}

func TestCleanCacheKeepsReferencedFiles(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	images := storage.NewImageStore(dir)
	icons := storage.NewAppIconStore(dir)

	kept, err := images.StorePNG(testPNGBytes(t, 32, 32, 10))
	if err != nil {
		t.Fatal(err)
	}
	orphan, err := images.StorePNG(testPNGBytes(t, 24, 24, 20))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := db.Upsert(ctx, &database.Item{Kind: database.KindImage, Content: kept.FileName}); err != nil {
		t.Fatal(err)
	}
	keptIcon, err := icons.StorePNG(testPNGBytes(t, 16, 16, 30))
	if err != nil {
		t.Fatal(err)
	}
	orphanIcon, err := icons.StorePNG(testPNGBytes(t, 16, 16, 40))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.EnsureApp(ctx, "com.example.app", "Example", &keptIcon); err != nil {
		t.Fatal(err)
	}

	runtime := &clipboardRuntime{db: db, images: images, appIcons: icons}
	files, freed, err := runtime.cleanCache(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if files != 3 || freed <= 0 {
		t.Fatalf("removed %d files, %d bytes", files, freed)
	}
	if _, err := os.Stat(images.OriginPath(kept.FileName)); err != nil {
		t.Errorf("referenced image removed: %v", err)
	}
	if _, err := os.Stat(images.OriginPath(orphan.FileName)); !os.IsNotExist(err) {
		t.Errorf("orphan image kept: %v", err)
	}
	if !icons.Exists(keptIcon) {
		t.Error("referenced icon removed")
	}
	if icons.Exists(orphanIcon) {
		t.Error("orphan icon kept")
	}
}

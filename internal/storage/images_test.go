package storage

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func testPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	img.Set(0, 0, color.RGBA{B: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestImageStoreIsIdempotent(t *testing.T) {
	store := NewImageStore(t.TempDir())
	data := testPNG(t, 3, 2)
	first, err := store.StorePNG(data)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.StorePNG(data)
	if err != nil {
		t.Fatal(err)
	}
	if first.FileName != second.FileName || first.Width != 3 || first.Height != 2 {
		t.Fatalf("stored = %+v / %+v", first, second)
	}
	stored, err := store.Read(first.FileName)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stored, data) {
		t.Error("stored bytes differ")
	}
	if err := store.Remove(first.FileName); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(store.root, "origin")); err != nil {
		t.Fatalf("origin dir missing: %v", err)
	}
}

func TestPNGDimensionsRejectsOtherData(t *testing.T) {
	if w, h := PNGDimensions([]byte("not a png")); w != 0 || h != 0 {
		t.Errorf("dimensions = %d×%d", w, h)
	}
}

func TestImageStoreCreatesBoundedThumbnail(t *testing.T) {
	store := NewImageStore(t.TempDir())
	stored, err := store.StorePNG(testPNG(t, 900, 300))
	if err != nil {
		t.Fatal(err)
	}
	thumbPath := store.ThumbPath(stored.FileName)
	data, err := os.ReadFile(thumbPath)
	if err != nil {
		t.Fatal(err)
	}
	thumb, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if got := thumb.Bounds().Dx(); got != ThumbnailMaxEdge {
		t.Errorf("thumbnail width = %d, want %d", got, ThumbnailMaxEdge)
	}
	if got := thumb.Bounds().Dy(); got != 100 {
		t.Errorf("thumbnail height = %d, want 100", got)
	}
	if err := store.Remove(stored.FileName); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(thumbPath); !os.IsNotExist(err) {
		t.Errorf("thumbnail still exists: %v", err)
	}
}

func TestSmallImageThumbnailIsOriginal(t *testing.T) {
	store := NewImageStore(t.TempDir())
	data := testPNG(t, 12, 8)
	stored, err := store.StorePNG(data)
	if err != nil {
		t.Fatal(err)
	}
	thumb, err := os.ReadFile(store.ThumbPath(stored.FileName))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(thumb, data) {
		t.Error("small thumbnail was re-encoded")
	}
}

func TestEnsureThumbnailBackfillsMissingFile(t *testing.T) {
	store := NewImageStore(t.TempDir())
	stored, err := store.StorePNG(testPNG(t, 600, 200))
	if err != nil {
		t.Fatal(err)
	}
	thumbPath := store.ThumbPath(stored.FileName)
	if err := os.Remove(thumbPath); err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureThumbnail(stored.FileName); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(thumbPath); err != nil {
		t.Fatalf("thumbnail was not recreated: %v", err)
	}
}

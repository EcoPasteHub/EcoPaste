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

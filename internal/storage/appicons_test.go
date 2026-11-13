package storage

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

func TestAppIconStoreIsIdempotent(t *testing.T) {
	store := NewAppIconStore(t.TempDir())
	data := []byte("\x89PNG\r\n\x1a\nicon")
	first, err := store.StorePNG(data)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.StorePNG(data)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Errorf("file names differ: %q != %q", first, second)
	}
	got, err := os.ReadFile(store.Path(first))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Error("stored icon differs")
	}
	if !store.Exists(first) {
		t.Error("stored icon should exist")
	}
}

func TestAppIconStoreRejectsTraversal(t *testing.T) {
	store := NewAppIconStore(t.TempDir())
	if path := store.Path("../outside.png"); path != "" {
		t.Errorf("path = %q", path)
	}
	if store.Exists("../outside.png") {
		t.Error("traversal path should not exist")
	}
}

func TestDownscalePNG(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 32, 16))
	source.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, source); err != nil {
		t.Fatal(err)
	}
	got, err := DownscalePNG(buf.Bytes(), 8)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := png.Decode(bytes.NewReader(got))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Bounds().Dx() != 8 || decoded.Bounds().Dy() != 4 {
		t.Errorf("bounds = %v", decoded.Bounds())
	}
}

func TestAppIconStoreRemovesUnreferenced(t *testing.T) {
	store := NewAppIconStore(t.TempDir())
	keep, err := store.StorePNG([]byte("one"))
	if err != nil {
		t.Fatal(err)
	}
	remove, err := store.StorePNG([]byte("two"))
	if err != nil {
		t.Fatal(err)
	}
	removed, err := store.RemoveUnreferenced(map[string]struct{}{keep: {}})
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Errorf("removed = %d, want 1", removed)
	}
	if !store.Exists(keep) {
		t.Error("referenced icon was removed")
	}
	if store.Exists(remove) {
		t.Error("unreferenced icon still exists")
	}
}

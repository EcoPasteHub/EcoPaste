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
	removed, bytes, err := store.RemoveUnreferenced(map[string]struct{}{keep: {}})
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Errorf("removed = %d, want 1", removed)
	}
	if bytes != int64(len("two")) {
		t.Errorf("removed bytes = %d", bytes)
	}
	if !store.Exists(keep) {
		t.Error("referenced icon was removed")
	}
	if store.Exists(remove) {
		t.Error("unreferenced icon still exists")
	}
}

func TestFileIconStoreUsesPublishedDirectory(t *testing.T) {
	resources := t.TempDir()
	store := NewFileIconStore(resources)
	name, err := store.StorePNG([]byte("file icon"))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(resources, "file-icons", name)
	if got := store.Path(name); got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
	if !store.Exists(name) {
		t.Error("stored file icon is missing")
	}
}

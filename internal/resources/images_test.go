package resources

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreAndThumbnailUseBlake3Sharding(t *testing.T) {
	store := NewImageStore(filepath.Join(t.TempDir(), "clipboard-images"))
	src := image.NewRGBA(image.Rect(0, 0, 600, 300))
	src.Set(0, 0, color.RGBA{R: 255, A: 255})
	var raw bytes.Buffer
	if err := png.Encode(&raw, src); err != nil {
		t.Fatal(err)
	}

	stored, err := store.Store(raw.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.FileName) != 68 {
		t.Fatalf("file name = %q", stored.FileName)
	}
	if filepath.Base(filepath.Dir(store.OriginPath(stored.FileName))) != stored.FileName[:2] {
		t.Fatalf("origin path = %q", store.OriginPath(stored.FileName))
	}
	thumbnail, err := store.EnsureThumbnail(stored.FileName)
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(thumbnail)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	decoded, err := png.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Bounds().Dx() != 300 || decoded.Bounds().Dy() != 150 {
		t.Fatalf("thumbnail = %v", decoded.Bounds())
	}
}

func TestStoreIsIdempotentAndDeleteRemovesBoth(t *testing.T) {
	store := NewImageStore(t.TempDir())
	src := image.NewRGBA(image.Rect(0, 0, 1, 1))
	var raw bytes.Buffer
	if err := png.Encode(&raw, src); err != nil {
		t.Fatal(err)
	}
	first, err := store.Store(raw.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Store(raw.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if first.FileName != second.FileName {
		t.Fatal("same bytes must map to the same file")
	}
	if _, err := store.EnsureThumbnail(first.FileName); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(first.FileName); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(store.OriginPath(first.FileName)); !os.IsNotExist(err) {
		t.Fatal("origin not removed")
	}
	if _, err := os.Stat(store.ThumbnailPath(first.FileName)); !os.IsNotExist(err) {
		t.Fatal("thumbnail not removed")
	}
}

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

func TestImageStoreUsesPublishedBLAKE3NameAndShard(t *testing.T) {
	const emptyBLAKE3 = "af1349b9f5f9a1a6a0404dea36dcc9499bcb25c9adc112b7cc9a93cae41f3262"
	if got := imageDigest(nil); got != emptyBLAKE3 {
		t.Fatalf("image digest = %q, want %q", got, emptyBLAKE3)
	}
	store := NewImageStore(t.TempDir())
	data := testPNG(t, 5, 4)
	stored, err := store.StorePNG(data)
	if err != nil {
		t.Fatal(err)
	}
	if stored.FileName != stored.Digest+".png" {
		t.Errorf("file name = %q, digest = %q", stored.FileName, stored.Digest)
	}
	want := filepath.Join(store.root, "origin", stored.Digest[:2], stored.FileName)
	if got := store.OriginPath(stored.FileName); got != want {
		t.Errorf("origin path = %q, want %q", got, want)
	}
	if _, err := os.Stat(store.OriginPath(stored.FileName)); err != nil {
		t.Fatal(err)
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

func TestRemoveUnreferencedImages(t *testing.T) {
	root := t.TempDir()
	store := NewImageStore(root)
	kept, err := store.StorePNG(testPNG(t, 32, 32))
	if err != nil {
		t.Fatal(err)
	}
	orphan, err := store.StorePNG(testPNG(t, 16, 16))
	if err != nil {
		t.Fatal(err)
	}
	files, removedBytes, err := store.RemoveUnreferenced(map[string]struct{}{kept.FileName: {}})
	if err != nil {
		t.Fatal(err)
	}
	if files != 2 || removedBytes <= 0 {
		t.Fatalf("removed files=%d bytes=%d", files, removedBytes)
	}
	if _, err := os.Stat(store.OriginPath(kept.FileName)); err != nil {
		t.Fatalf("referenced image removed: %v", err)
	}
	if _, err := os.Stat(store.OriginPath(orphan.FileName)); !os.IsNotExist(err) {
		t.Fatalf("orphan image kept: %v", err)
	}
	if _, err := os.Stat(store.ThumbPath(orphan.FileName)); !os.IsNotExist(err) {
		t.Fatalf("orphan thumbnail kept: %v", err)
	}
}

package resources

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"github.com/zeebo/blake3"
	xdraw "golang.org/x/image/draw"

	"github.com/EcoPasteHub/EcoPaste/internal/imageutil"
)

const (
	originDir    = "origin"
	thumbnailDir = "thumbnails"
	thumbnailMax = 300
)

// ImageStore owns clipboard PNG resources under resources/clipboard-images.
type ImageStore struct {
	root string
}

// StoredImage describes one image written into the resource tree.
type StoredImage struct {
	FileName string
	Width    int
	Height   int
	Size     int64
}

// NewImageStore binds a resource root.
func NewImageStore(root string) *ImageStore {
	return &ImageStore{root: root}
}

// Store normalizes data to PNG and writes it if not already present.
func (s *ImageStore) Store(data []byte) (StoredImage, error) {
	image, err := imageutil.Decode(data)
	if err != nil {
		return StoredImage{}, err
	}
	sum := blake3.Sum256(image.Bytes)
	name := fmt.Sprintf("%x.png", sum[:])
	path := s.OriginPath(name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return StoredImage{}, err
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(path, image.Bytes, 0o644); err != nil {
			return StoredImage{}, err
		}
	}
	return StoredImage{
		FileName: name,
		Width:    image.Width,
		Height:   image.Height,
		Size:     int64(len(image.Bytes)),
	}, nil
}

// OriginPath resolves the sharded origin path for a file name.
func (s *ImageStore) OriginPath(name string) string {
	return filepath.Join(s.root, originDir, shard(name), name)
}

// ThumbnailPath resolves the sharded thumbnail path for a file name.
func (s *ImageStore) ThumbnailPath(name string) string {
	return filepath.Join(s.root, thumbnailDir, shard(name), name)
}

// EnsureThumbnail lazily creates a PNG whose longest side is at most 300.
func (s *ImageStore) EnsureThumbnail(name string) (string, error) {
	path := s.ThumbnailPath(name)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	raw, err := os.ReadFile(s.OriginPath(name))
	if err != nil {
		return "", err
	}
	src, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	width, height := fit(src.Bounds().Dx(), src.Bounds().Dy(), thumbnailMax)
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Src, nil)

	var out bytes.Buffer
	if err := png.Encode(&out, dst); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, out.Bytes(), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// Delete removes origin and thumbnail files, ignoring missing entries.
func (s *ImageStore) Delete(name string) error {
	for _, path := range []string{s.OriginPath(name), s.ThumbnailPath(name)} {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		removeEmptyParents(path, s.root)
	}
	return nil
}

func shard(name string) string {
	if len(name) < 2 {
		return "00"
	}
	return name[:2]
}

func fit(width, height, max int) (int, int) {
	if width <= max && height <= max {
		return width, height
	}
	if width >= height {
		return max, maxInt(1, height*max/width)
	}
	return maxInt(1, width*max/height), max
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func removeEmptyParents(path, stop string) {
	for dir := filepath.Dir(path); dir != stop && len(dir) > len(stop); dir = filepath.Dir(dir) {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			return
		}
		if err := os.Remove(dir); err != nil {
			return
		}
	}
}

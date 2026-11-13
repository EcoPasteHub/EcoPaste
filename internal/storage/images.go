package storage

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zeebo/blake3"
)

// ImageStore keeps clipboard images as PNG files sharded by the first
// two hex digits of their content hash.
type ImageStore struct {
	root string
}

// StoredImage is what ingesting one image produced.
type StoredImage struct {
	FileName string
	Digest   string
	Width    int64
	Height   int64
	Size     int64
}

// NewImageStore returns the image store under a data root.
func NewImageStore(resources string) *ImageStore {
	return &ImageStore{root: filepath.Join(resources, "clipboard-images")}
}

// StorePNG writes pngBytes if absent and returns the file name stored in
// the item's content column.
func (s *ImageStore) StorePNG(pngBytes []byte) (StoredImage, error) {
	if len(pngBytes) == 0 {
		return StoredImage{}, fmt.Errorf("empty image")
	}
	digest := imageDigest(pngBytes)
	name := digest + ".png"
	path := s.OriginPath(name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return StoredImage{}, fmt.Errorf("create image shard: %w", err)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		tmp := path + ".tmp"
		if err := os.WriteFile(tmp, pngBytes, 0o644); err != nil {
			return StoredImage{}, fmt.Errorf("write image: %w", err)
		}
		if err := os.Rename(tmp, path); err != nil {
			return StoredImage{}, fmt.Errorf("promote image: %w", err)
		}
	}
	width, height := PNGDimensions(pngBytes)
	return StoredImage{
		FileName: name,
		Digest:   digest,
		Width:    int64(width),
		Height:   int64(height),
		Size:     int64(len(pngBytes)),
	}, nil
}

// Read returns the original PNG bytes of a stored image.
func (s *ImageStore) Read(fileName string) ([]byte, error) {
	return os.ReadFile(s.OriginPath(fileName))
}

// OriginPath returns the absolute path of a stored image.
func (s *ImageStore) OriginPath(fileName string) string {
	shard := fileName
	if len(shard) > 2 {
		shard = shard[:2]
	}
	return filepath.Join(s.root, "origin", shard, fileName)
}

// Remove deletes an image's original and thumbnail, ignoring missing
// files.
func (s *ImageStore) Remove(fileName string) error {
	shard := fileName
	if len(shard) > 2 {
		shard = shard[:2]
	}
	for _, dir := range []string{"origin", "thumbnails"} {
		if err := os.Remove(filepath.Join(s.root, dir, shard, fileName)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove image: %w", err)
		}
	}
	return nil
}

// PNGDimensions reads a PNG's width and height without decoding pixels.
func PNGDimensions(data []byte) (width, height uint32) {
	const signature = "\x89PNG\r\n\x1a\n"
	if len(data) < 24 || string(data[:8]) != signature || string(data[12:16]) != "IHDR" {
		return 0, 0
	}
	width = uint32(data[16])<<24 | uint32(data[17])<<16 | uint32(data[18])<<8 | uint32(data[19])
	height = uint32(data[20])<<24 | uint32(data[21])<<16 | uint32(data[22])<<8 | uint32(data[23])
	return width, height
}

func imageDigest(data []byte) string {
	hasher := blake3.New()
	_, _ = hasher.Write(data)
	return hex.EncodeToString(hasher.Sum(nil))
}

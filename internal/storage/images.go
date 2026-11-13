package storage

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"github.com/zeebo/blake3"
	"golang.org/x/image/draw"
)

// ThumbnailMaxEdge is the longest edge of a list thumbnail.
const ThumbnailMaxEdge = 300

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
	if err := writeIfMissing(path, pngBytes); err != nil {
		return StoredImage{}, fmt.Errorf("store image: %w", err)
	}
	if err := s.writeThumbnail(name, pngBytes); err != nil {
		return StoredImage{}, err
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

// EnsureThumbnail creates a missing list thumbnail from the original,
// so history imported from older builds gets one on first display.
func (s *ImageStore) EnsureThumbnail(fileName string) error {
	thumbPath := s.ThumbPath(fileName)
	if _, err := os.Stat(thumbPath); err == nil {
		return nil
	}
	data, err := os.ReadFile(s.OriginPath(fileName))
	if err != nil {
		return fmt.Errorf("read image for thumbnail: %w", err)
	}
	return s.writeThumbnail(fileName, data)
}

// OriginPath returns the absolute path of a stored image.
func (s *ImageStore) OriginPath(fileName string) string {
	return filepath.Join(s.root, "origin", shardName(fileName), fileName)
}

// ThumbPath returns the absolute path of a stored image's list thumbnail.
func (s *ImageStore) ThumbPath(fileName string) string {
	return filepath.Join(s.root, "thumbnails", shardName(fileName), fileName)
}

// Remove deletes an image's original and thumbnail, ignoring missing
// files.
func (s *ImageStore) Remove(fileName string) error {
	for _, dir := range []string{"origin", "thumbnails"} {
		if err := os.Remove(filepath.Join(s.root, dir, shardName(fileName), fileName)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove image: %w", err)
		}
	}
	return nil
}

// RemoveUnreferenced deletes originals and thumbnails whose file names
// no history row references. It returns the removed file count and
// bytes.
func (s *ImageStore) RemoveUnreferenced(referenced map[string]struct{}) (int, int64, error) {
	var (
		files int
		bytes int64
	)
	for _, dir := range []string{"origin", "thumbnails"} {
		root := filepath.Join(s.root, dir)
		entries, err := os.ReadDir(root)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return files, bytes, fmt.Errorf("list images: %w", err)
		}
		for _, shard := range entries {
			if !shard.IsDir() {
				continue
			}
			shardPath := filepath.Join(root, shard.Name())
			names, err := os.ReadDir(shardPath)
			if err != nil {
				return files, bytes, fmt.Errorf("list image shard: %w", err)
			}
			for _, entry := range names {
				if entry.IsDir() {
					continue
				}
				if _, ok := referenced[entry.Name()]; ok {
					continue
				}
				info, err := entry.Info()
				if err != nil {
					return files, bytes, fmt.Errorf("read image info: %w", err)
				}
				if err := os.Remove(filepath.Join(shardPath, entry.Name())); err != nil {
					return files, bytes, fmt.Errorf("remove unreferenced image: %w", err)
				}
				files++
				bytes += info.Size()
			}
			if empty, err := dirEmpty(shardPath); err == nil && empty {
				os.Remove(shardPath)
			}
		}
	}
	return files, bytes, nil
}

func dirEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

func shardName(fileName string) string {
	if len(fileName) > 2 {
		return fileName[:2]
	}
	return fileName
}

func (s *ImageStore) writeThumbnail(fileName string, data []byte) error {
	thumb, err := thumbnailPNG(data)
	if err != nil {
		return err
	}
	if err := writeIfMissing(s.ThumbPath(fileName), thumb); err != nil {
		return fmt.Errorf("store thumbnail: %w", err)
	}
	return nil
}

func writeIfMissing(path string, data []byte) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func thumbnailPNG(data []byte) ([]byte, error) {
	source, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image for thumbnail: %w", err)
	}
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= ThumbnailMaxEdge && height <= ThumbnailMaxEdge {
		return data, nil
	}
	scale := float64(ThumbnailMaxEdge) / float64(max(width, height))
	thumbWidth := max(1, int(float64(width)*scale))
	thumbHeight := max(1, int(float64(height)*scale))
	target := image.NewRGBA(image.Rect(0, 0, thumbWidth, thumbHeight))
	draw.CatmullRom.Scale(target, target.Bounds(), source, bounds, draw.Over, nil)
	var buf bytes.Buffer
	if err := png.Encode(&buf, target); err != nil {
		return nil, fmt.Errorf("encode thumbnail: %w", err)
	}
	return buf.Bytes(), nil
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

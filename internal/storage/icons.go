package storage

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
)

// pngIconStore keeps PNG icons as flat files named by their content hash.
// The distinct icon count is bounded by installed apps and file types, so
// sharding the directory adds complexity without a practical benefit.
type pngIconStore struct {
	root string
	kind string
}

func newPNGIconStore(root, kind string) *pngIconStore {
	return &pngIconStore{root: root, kind: kind}
}

// StorePNG writes an icon if it is not already present and returns the
// file name used by the database.
func (s *pngIconStore) StorePNG(data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("empty %s", s.kind)
	}
	name := imageDigest(data) + ".png"
	if err := writeIfMissing(filepath.Join(s.root, name), data); err != nil {
		return "", fmt.Errorf("store %s: %w", s.kind, err)
	}
	return name, nil
}

// Path returns the absolute path for an icon file name. Invalid names are
// rejected so imported data cannot escape the icon directory.
func (s *pngIconStore) Path(name string) string {
	if name == "" || filepath.Base(name) != name {
		return ""
	}
	return filepath.Join(s.root, name)
}

// Exists reports whether the icon file is still present.
func (s *pngIconStore) Exists(name string) bool {
	path := s.Path(name)
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

// RemoveUnreferenced deletes PNG icons not present in the referenced set.
// Missing directories are treated as already clean.
func (s *pngIconStore) RemoveUnreferenced(referenced map[string]struct{}) (int, int64, error) {
	entries, err := os.ReadDir(s.root)
	if os.IsNotExist(err) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, fmt.Errorf("read %ss: %w", s.kind, err)
	}
	var (
		removed int
		bytes   int64
	)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".png") {
			continue
		}
		if _, ok := referenced[name]; ok {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return removed, bytes, fmt.Errorf("read %s info: %w", s.kind, err)
		}
		if err := os.Remove(filepath.Join(s.root, name)); err != nil && !os.IsNotExist(err) {
			return removed, bytes, fmt.Errorf("remove %s %s: %w", s.kind, name, err)
		}
		removed++
		bytes += info.Size()
	}
	return removed, bytes, nil
}

// DownscalePNG fits an image inside maxEdge and encodes it as an 8-bit PNG.
// Images already within the bound still pass through png.Encode so 16-bit
// source icons do not remain oversized.
func DownscalePNG(data []byte, maxEdge int) ([]byte, error) {
	source, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode PNG: %w", err)
	}
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 || maxEdge <= 0 {
		return nil, fmt.Errorf("invalid PNG dimensions")
	}
	scale := min(1.0, float64(maxEdge)/float64(max(width, height)))
	targetWidth := max(1, int(float64(width)*scale))
	targetHeight := max(1, int(float64(height)*scale))
	target := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	draw.CatmullRom.Scale(target, target.Bounds(), source, bounds, draw.Over, nil)
	var out bytes.Buffer
	if err := png.Encode(&out, target); err != nil {
		return nil, fmt.Errorf("encode PNG: %w", err)
	}
	return out.Bytes(), nil
}

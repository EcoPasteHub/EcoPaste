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

// AppIconStore keeps application icons as flat PNG files named by their
// content hash. Unlike clipboard images, the number of distinct icons is
// bounded by the applications the user copies from, so sharding would add
// complexity without a practical benefit.
type AppIconStore struct {
	root string
}

// AppIconMaxEdge is the square edge used for cached application icons.
const AppIconMaxEdge = 128

// NewAppIconStore returns the app icon store under a resources directory.
func NewAppIconStore(resources string) *AppIconStore {
	return &AppIconStore{root: filepath.Join(resources, "app-icons")}
}

// StorePNG writes an application icon if it is not already present and
// returns the file name stored in clipboard_apps.icon_file.
func (s *AppIconStore) StorePNG(data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("empty app icon")
	}
	name := imageDigest(data) + ".png"
	if err := writeIfMissing(filepath.Join(s.root, name), data); err != nil {
		return "", fmt.Errorf("store app icon: %w", err)
	}
	return name, nil
}

// Path returns the absolute path for an icon file name. Invalid names are
// rejected so imported data cannot escape the icon directory.
func (s *AppIconStore) Path(name string) string {
	if name == "" || filepath.Base(name) != name {
		return ""
	}
	return filepath.Join(s.root, name)
}

// Exists reports whether the icon file is still present.
func (s *AppIconStore) Exists(name string) bool {
	path := s.Path(name)
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

// RemoveUnreferenced deletes PNG icons not present in the referenced set.
// Missing directories are treated as already clean.
func (s *AppIconStore) RemoveUnreferenced(referenced map[string]struct{}) (int, error) {
	entries, err := os.ReadDir(s.root)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read app icons: %w", err)
	}
	removed := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".png") {
			continue
		}
		if _, ok := referenced[name]; ok {
			continue
		}
		if err := os.Remove(filepath.Join(s.root, name)); err != nil && !os.IsNotExist(err) {
			return removed, fmt.Errorf("remove app icon %s: %w", name, err)
		}
		removed++
	}
	return removed, nil
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

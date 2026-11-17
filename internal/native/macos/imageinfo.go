//go:build darwin

package macos

import (
	"image"
	"os"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// ImagePixelSize reports an image file's natural size, which a
// file-copied image uses to size its list preview. Formats the standard
// decoders cannot read report false so the row falls back to a file entry.
func ImagePixelSize(path string) (int, int, bool) {
	if path == "" {
		return 0, 0, false
	}
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, false
	}
	defer file.Close()
	config, _, err := image.DecodeConfig(file)
	if err != nil || config.Width <= 0 || config.Height <= 0 {
		return 0, 0, false
	}
	return config.Width, config.Height, true
}

//go:build darwin

package macos

import (
	"bytes"
	"image/png"
	"os"
	"testing"
)

func TestFileIconPNGCapsPixels(t *testing.T) {
	path := "/System/Applications/Notes.app"
	if _, err := os.Stat(path); err != nil {
		t.Skip("Notes.app is not installed")
	}
	data := FileIconPNG(path)
	if len(data) == 0 {
		t.Fatal("icon is empty")
	}
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode icon: %v", err)
	}
	if config.Width > iconPixels || config.Height > iconPixels {
		t.Fatalf("icon = %dx%d, want at most %dpx", config.Width, config.Height, iconPixels)
	}
}

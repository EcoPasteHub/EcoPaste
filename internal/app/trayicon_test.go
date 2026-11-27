package app

import (
	"bytes"
	"image"
	"image/png"
	"testing"
)

func TestTrayIconDrawsTheLegacyMark(t *testing.T) {
	img, err := png.Decode(bytes.NewReader(trayIcon()))
	if err != nil {
		t.Fatalf("decode tray icon: %v", err)
	}
	if got, want := img.Bounds().Size(), (image.Point{256, 256}); got != want {
		t.Errorf("size = %v, want %v", got, want)
	}

	// A template image only uses alpha: the mark is opaque and the canvas
	// around it is clear.
	opaque := 0
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a > 0x8000 {
				opaque++
			}
		}
	}
	if share := float64(opaque) / (256 * 256); share < 0.2 || share > 0.6 {
		t.Errorf("opaque share = %.2f, want the mark's alpha", share)
	}
	if _, _, _, a := img.At(0, 0).RGBA(); a != 0 {
		t.Error("corner should be transparent")
	}
}

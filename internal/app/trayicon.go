package app

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
)

// trayIcon draws a clipboard glyph as a macOS template image: only the
// alpha channel matters, so it follows the menu bar's appearance.
func trayIcon() []byte {
	const size = 36
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	black := color.RGBA{A: 0xff}

	stroke := func(x0, y0, x1, y1, width float64) {
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				if distanceToSegment(float64(x)+0.5, float64(y)+0.5, x0, y0, x1, y1) <= width/2 {
					img.SetRGBA(x, y, black)
				}
			}
		}
	}

	stroke(9, 8, 9, 31, 2.2)
	stroke(27, 8, 27, 31, 2.2)
	stroke(9, 31, 27, 31, 2.2)
	stroke(9, 8, 15, 8, 2.2)
	stroke(21, 8, 27, 8, 2.2)
	stroke(14, 5, 22, 5, 2.2)
	stroke(14, 5, 14, 10, 2.2)
	stroke(22, 5, 22, 10, 2.2)
	stroke(14, 10, 22, 10, 2.2)
	stroke(13, 16, 23, 16, 1.8)
	stroke(13, 21, 23, 21, 1.8)
	stroke(13, 26, 19, 26, 1.8)

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func distanceToSegment(px, py, x0, y0, x1, y1 float64) float64 {
	dx, dy := x1-x0, y1-y0
	length := dx*dx + dy*dy
	if length == 0 {
		return math.Hypot(px-x0, py-y0)
	}
	t := ((px-x0)*dx + (py-y0)*dy) / length
	t = math.Max(0, math.Min(1, t))
	return math.Hypot(px-(x0+t*dx), py-(y0+t*dy))
}

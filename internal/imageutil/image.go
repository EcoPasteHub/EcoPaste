package imageutil

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
)

// Image is normalized PNG data and its dimensions.
type Image struct {
	Bytes  []byte
	Width  int
	Height int
}

// Decode keeps PNG bytes as-is and transcodes other decodable images to PNG.
func Decode(data []byte) (Image, error) {
	if len(data) == 0 {
		return Image{}, errors.New("empty image data")
	}
	if width, height, ok := PNGSize(data); ok {
		return Image{Bytes: data, Width: width, Height: height}, nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return Image{}, err
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return Image{}, err
	}
	bounds := img.Bounds()
	return Image{Bytes: out.Bytes(), Width: bounds.Dx(), Height: bounds.Dy()}, nil
}

// PNGSize reads IHDR dimensions without decoding pixels.
func PNGSize(data []byte) (int, int, bool) {
	if len(data) < 24 || !bytes.Equal(data[:8], []byte{137, 80, 78, 71, 13, 10, 26, 10}) {
		return 0, 0, false
	}
	if string(data[12:16]) != "IHDR" {
		return 0, 0, false
	}
	width := int(binary.BigEndian.Uint32(data[16:20]))
	height := int(binary.BigEndian.Uint32(data[20:24]))
	return width, height, width > 0 && height > 0
}

package storage

import (
	"bytes"
	"image"
	"image/png"
	"testing"
)

func BenchmarkImageStoreRepeatedPNG(b *testing.B) {
	img := image.NewNRGBA(image.Rect(0, 0, 1600, 1000))
	for index := range img.Pix {
		img.Pix[index] = byte(index*31 + index/1600)
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		b.Fatal(err)
	}
	data := encoded.Bytes()
	store := NewImageStore(b.TempDir())
	if _, err := store.StorePNG(data); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := store.StorePNG(data); err != nil {
			b.Fatal(err)
		}
	}
}

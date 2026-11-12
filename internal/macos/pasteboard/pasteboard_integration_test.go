//go:build darwin

package pasteboard

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

func TestNamedPasteboardRoundTrip(t *testing.T) {
	pb, err := Named("com.ayangweb.ecopaste.go.test")
	if err != nil {
		t.Fatal(err)
	}
	capture := settings.Default().Clipboard.Capture

	pb.WriteText("plain body", "<b>plain body</b>", `{\rtf1 plain body}`, false)
	payload, ok, err := pb.Read(capture)
	if err != nil || !ok {
		t.Fatalf("read text: ok=%v err=%v", ok, err)
	}
	if payload.Text.Plain != "plain body" {
		t.Fatalf("plain = %q", payload.Text.Plain)
	}
	if payload.Text.HTML == "" || payload.Text.RTF == "" {
		t.Fatalf("rich text missing: %+v", payload.Text)
	}

	before := pb.ChangeCount()
	pb.WriteText("second", "", "", true)
	if pb.ChangeCount() == before {
		t.Fatal("changeCount did not advance")
	}
	payload, ok, err = pb.Read(capture)
	if err != nil || !ok || payload.Text.Plain != "second" {
		t.Fatalf("plain read after write: %+v %v %v", payload, ok, err)
	}

	var imageData bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 4, 2))
	img.Set(0, 0, color.RGBA{G: 255, A: 255})
	if err := png.Encode(&imageData, img); err != nil {
		t.Fatal(err)
	}
	if err := pb.WriteImage(imageData.Bytes()); err != nil {
		t.Fatal(err)
	}
	payload, ok, err = pb.Read(capture)
	if err != nil || !ok {
		t.Fatalf("read image: ok=%v err=%v", ok, err)
	}
	if payload.Kind != "image" || payload.Image.Width != 4 || payload.Image.Height != 2 {
		t.Fatalf("image payload = %+v", payload)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(path, []byte("note"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := pb.WriteFiles([]string{path}); err != nil {
		t.Fatal(err)
	}
	payload, ok, err = pb.Read(capture)
	if err != nil || !ok {
		t.Fatalf("read files: ok=%v err=%v", ok, err)
	}
	if payload.Kind != "files" || len(payload.Files) != 1 || payload.Files[0] != path {
		t.Fatalf("files payload = %+v", payload)
	}
}

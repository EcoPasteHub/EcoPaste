package clipboard

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/domain"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

func TestBuildItemHTMLKeepsSourceAndPlainSearchText(t *testing.T) {
	item, ok, err := BuildItem(Payload{
		Kind: domain.KindText,
		Text: Text{Plain: "Hello World", HTML: "<b>Hello</b> World"},
	}, settings.Default().Clipboard.Capture, settings.Default().Clipboard.Sensitive)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if item.SubKind != "html" || item.Content != "<b>Hello</b> World" {
		t.Fatalf("item = %+v", item)
	}
	if item.SearchText != "Hello World" || item.Summary != "Hello World" {
		t.Fatalf("search/summary = %q / %q", item.SearchText, item.Summary)
	}
}

func TestBuildItemFiles(t *testing.T) {
	dir := t.TempDir()
	item, ok, err := BuildItem(Payload{
		Kind:  domain.KindFiles,
		Files: []string{dir, dir + "/missing.txt"},
	}, settings.Default().Clipboard.Capture, settings.Default().Clipboard.Sensitive)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if item.FileTypes != "d,f" {
		t.Fatalf("file types = %q", item.FileTypes)
	}
	if item.Content != dir+"\n"+dir+"/missing.txt" {
		t.Fatalf("content = %q", item.Content)
	}
}

func TestBuildItemImageUsesPNGName(t *testing.T) {
	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 2, 3))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeImage(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Width != 2 || decoded.Height != 3 {
		t.Fatalf("size = %dx%d", decoded.Width, decoded.Height)
	}
	item, ok, err := BuildItem(Payload{Kind: domain.KindImage, Image: decoded},
		settings.Default().Clipboard.Capture, settings.Default().Clipboard.Sensitive)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if len(item.Content) != 68 || item.Content[len(item.Content)-4:] != ".png" {
		t.Fatalf("content = %q", item.Content)
	}
}

func TestSensitiveTextIsSkippedWhenCollectionDisabled(t *testing.T) {
	capture := settings.Default().Clipboard.Capture
	sensitive := settings.Default().Clipboard.Sensitive
	sensitive.CollectSecrets = false
	_, ok, err := BuildItem(Payload{
		Kind: domain.KindText,
		Text: Text{Plain: "api_key = abcdefgh12345678"},
	}, capture, sensitive)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("sensitive text must be skipped")
	}
}

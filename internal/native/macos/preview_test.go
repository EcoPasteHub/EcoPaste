//go:build darwin

package macos

import (
	"strings"
	"testing"
)

func TestAttributedPreview(t *testing.T) {
	html := attributedPreview("<p>Hello <b>world</b></p>", "html")
	if html == 0 {
		t.Fatal("HTML preview did not produce an attributed string")
	}
	if text := strings.TrimSpace(goString(html.Send(sel("string")))); text != "Hello world" {
		t.Fatalf("HTML preview text = %q", text)
	}

	rtf := attributedPreview(`{\rtf1\ansi Hello \b world\b0}`, "rtf")
	if rtf == 0 {
		t.Fatal("RTF preview did not produce an attributed string")
	}
	if text := strings.TrimSpace(goString(rtf.Send(sel("string")))); text != "Hello world" {
		t.Fatalf("RTF preview text = %q", text)
	}
}

func TestPreviewSizeFollowsTextHeight(t *testing.T) {
	width, height := previewSize(PreviewContent{Text: "hello"}, 0)
	if width != previewWidth {
		t.Fatalf("preview width = %v, want %v", width, previewWidth)
	}
	if height < 180 || height > 460 {
		t.Fatalf("preview height = %v, want 180...460", height)
	}
}

func TestPreviewSizeClampsLongText(t *testing.T) {
	_, height := previewSize(PreviewContent{Text: strings.Repeat("line\n", 200)}, 0)
	if height != 460 {
		t.Fatalf("preview height = %v, want the 460 cap", height)
	}
}

func TestClampFloat(t *testing.T) {
	if got := clampFloat(120, 180, 460); got != 180 {
		t.Errorf("clamp below = %v", got)
	}
	if got := clampFloat(500, 180, 460); got != 460 {
		t.Errorf("clamp above = %v", got)
	}
	if got := clampFloat(300, 180, 460); got != 300 {
		t.Errorf("clamp inside = %v", got)
	}
}

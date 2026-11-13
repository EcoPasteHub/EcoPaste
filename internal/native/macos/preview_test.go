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

//go:build darwin

package macos

import (
	"os"
	"testing"
)

// TestPasteboardRoundTrip touches the real system pasteboard, so it only
// runs when explicitly asked for; it restores the previous text.
func TestPasteboardRoundTrip(t *testing.T) {
	if os.Getenv("ECOPASTE_PASTEBOARD_TEST") != "1" {
		t.Skip("set ECOPASTE_PASTEBOARD_TEST=1 to use the system pasteboard")
	}
	original, _ := ReadPasteboard()
	t.Cleanup(func() {
		if original.Text != "" {
			_ = WritePasteboard(WriteContent{Text: original.Text})
		}
	})

	if err := WritePasteboard(WriteContent{
		Text: "plain fallback",
		HTML: "<b>plain fallback</b>",
		RTF:  `{\rtf1\ansi plain fallback}`,
	}); err != nil {
		t.Fatal(err)
	}
	got, err := ReadPasteboard()
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "plain fallback" {
		t.Errorf("text = %q", got.Text)
	}
	if got.HTML != "<b>plain fallback</b>" {
		t.Errorf("html = %q", got.HTML)
	}
	if got.RTF == "" {
		t.Error("rtf missing")
	}
}

//go:build darwin

package macos

import (
	"strings"
	"testing"

	"github.com/ebitengine/purego/objc"
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

func TestNormalizePreviewHTMLFlattensListParagraphs(t *testing.T) {
	input := `<ol start="1"><li><p>first</p></li><li><p>second</p></li></ol>`
	want := `<ol start="1"><li>first</li><li>second</li></ol>`
	if got := normalizePreviewHTML(input); got != want {
		t.Fatalf("normalized HTML = %q, want %q", got, want)
	}
}

func TestAttributedPreviewDropsImportedListMarkers(t *testing.T) {
	attr := attributedPreview(`<ol><li><p>first</p></li><li><p>second</p></li></ol>`, "html")
	if attr == 0 {
		t.Fatal("HTML list preview did not produce an attributed string")
	}
	style := objc.Send[objc.ID](attr, sel("attribute:atIndex:effectiveRange:"),
		nsString("NSParagraphStyle"), 0, objc.ID(0))
	if style == 0 {
		t.Fatal("list preview has no paragraph style")
	}
	lists := style.Send(sel("textLists"))
	if lists != 0 && objc.Send[uint](lists, sel("count")) != 0 {
		t.Fatalf("list markers remain: %d", objc.Send[uint](lists, sel("count")))
	}
}

func TestStyledPreviewHTMLUsesSystemTypography(t *testing.T) {
	text := `<html><head><title>Preview</title></head><body><p>Hello <b>world</b></p></body></html>`
	got := styledPreviewHTML(text)
	if !strings.Contains(got, `font-family: -apple-system`) {
		t.Fatalf("preview style missing from %q", got)
	}
	if strings.Index(got, "font-family") > strings.Index(got, "</head>") {
		t.Fatalf("preview style must be inserted inside the document head: %q", got)
	}
	if !strings.Contains(got, "<p>Hello <b>world</b></p>") {
		t.Fatalf("preview markup changed: %q", got)
	}
}

func TestAttributedPreviewKeepsNativeDefaultFont(t *testing.T) {
	attr := attributedPreview("<p>Hello <b>world</b></p>", "html")
	if attr == 0 {
		t.Fatal("HTML preview did not produce an attributed string")
	}
	font := objc.Send[objc.ID](attr, sel("attribute:atIndex:effectiveRange:"),
		nsString("NSFont"), 0, objc.ID(0))
	if font == 0 {
		t.Fatal("HTML preview has no font attribute")
	}
	if family := goString(font.Send(sel("familyName"))); strings.Contains(family, "Times") {
		t.Fatalf("default HTML preview fell back to %q", family)
	}
}

func TestAttributedPreviewPreservesExplicitFont(t *testing.T) {
	attr := attributedPreview(`<p style="font-family: Georgia">Georgia</p>`, "html")
	if attr == 0 {
		t.Fatal("HTML preview did not produce an attributed string")
	}
	font := objc.Send[objc.ID](attr, sel("attribute:atIndex:effectiveRange:"),
		nsString("NSFont"), 0, objc.ID(0))
	if font == 0 {
		t.Fatal("HTML preview has no font attribute")
	}
	if family := goString(font.Send(sel("familyName"))); family != "Georgia" {
		t.Fatalf("explicit HTML font family = %q, want Georgia", family)
	}
}

func TestAttributedPreviewPreservesExplicitColor(t *testing.T) {
	attr := attributedPreview(`<p>plain <span style="color: red">red</span></p>`, "html")
	if attr == 0 {
		t.Fatal("HTML preview did not produce an attributed string")
	}
	color := objc.Send[objc.ID](attr, sel("attribute:atIndex:effectiveRange:"),
		nsString("NSColor"), 6, objc.ID(0))
	if color == 0 {
		t.Fatal("colored preview text has no color attribute")
	}
	description := goString(color.Send(sel("description")))
	if !strings.Contains(description, "1 0 0 1") {
		t.Fatalf("explicit red was lost: %s", description)
	}
}

func TestAttributedPreviewColorsUnstyledText(t *testing.T) {
	attr := attributedPreview(`<p>plain</p>`, "html")
	if attr == 0 {
		t.Fatal("HTML preview did not produce an attributed string")
	}
	color := objc.Send[objc.ID](attr, sel("attribute:atIndex:effectiveRange:"),
		nsString("NSColor"), 0, objc.ID(0))
	if color == 0 {
		t.Fatal("unstyled preview text has no semantic color attribute")
	}
}

func TestPreviewInkNeedsLabelForDefaultContrast(t *testing.T) {
	if !previewInkNeedsLabel(0.1, 0.1, 0.1, true) {
		t.Fatal("dark ink should use the label color in dark appearance")
	}
	if !previewInkNeedsLabel(0.98, 0.98, 0.98, false) {
		t.Fatal("light ink should use the label color in light appearance")
	}
	if previewInkNeedsLabel(0.98, 0.1, 0.1, true) {
		t.Fatal("colored ink should keep its explicit color")
	}
	if previewInkNeedsLabel(0.1, 0.1, 0.1, false) {
		t.Fatal("dark ink has sufficient contrast in light appearance")
	}
}

func TestPreviewSizeFollowsTextHeight(t *testing.T) {
	width, height := previewSize(PreviewContent{Text: "hello"}, 0)
	if width != previewWidth {
		t.Fatalf("preview width = %v, want %v", width, previewWidth)
	}
	if height < previewMinHeight || height > previewMaxHeight {
		t.Fatalf("preview height = %v, want %v...%v", height, previewMinHeight, previewMaxHeight)
	}
	if height >= 188 {
		t.Fatalf("short preview height = %v, want less than the previous minimum", height)
	}
}

func TestPreviewSizeKeepsSmallImagesReadable(t *testing.T) {
	image := objc.ID(class("NSImage")).Send(sel("new"))
	defer image.Send(sel("release"))
	image.Send(sel("setSize:"), CGSize{Width: 128, Height: 128})

	_, height := previewSize(PreviewContent{ImagePath: "ignored.png"}, image)
	if height != previewImageMinHeight {
		t.Fatalf("small image preview height = %v, want %v", height, previewImageMinHeight)
	}
}

func TestPreviewSizeClampsLongText(t *testing.T) {
	_, height := previewSize(PreviewContent{Text: strings.Repeat("line\n", 200)}, 0)
	if height != previewMaxHeight {
		t.Fatalf("preview height = %v, want the %v cap", height, previewMaxHeight)
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

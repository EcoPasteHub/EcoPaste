//go:build darwin

package macos

import (
	"math"
	"strings"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

// PreviewContent is the content shown in the native preview popover.
type PreviewContent struct {
	Active     bool
	Kind       string
	Title      string
	Meta       string
	Text       string
	RichText   string
	RichFormat string
	ImagePath  string
}

const (
	previewTitleTag = 200
	previewMetaTag  = 201

	// The popover content is fully transparent: it has no backdrop, card
	// fill or divider. A single vertical rhythm — a compact title/meta header,
	// a short gap, then the body — carries the hierarchy instead.
	previewPadding        = popoverContentPadding
	previewWidth          = 380.0
	previewMinWidth       = 240.0
	previewMaxWidth       = 420.0
	previewTitleHeight    = 18.0
	previewMetaHeight     = 15.0
	previewHeaderGap      = 3.0
	previewContentGap     = 10.0
	previewMinHeight      = 92.0
	previewImageMinHeight = 180.0
	previewMaxHeight      = 460.0
	previewScreenMargin   = 72.0
	previewTextInset      = 1.0
	previewTextFontSize   = 13.0
	previewMetaFontSize   = 11.0
	previewTitleFontSize  = 13.0
	previewLineSpacing    = 3.0

	previewHTMLStyle = `<style>
body { font-family: -apple-system, BlinkMacSystemFont, "Helvetica Neue", sans-serif; font-size: 13px; line-height: 1.55; }
code, pre { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
</style>`
)

var (
	previewViewClass     objc.Class
	popoverDelegateClass objc.Class

	panelsByPreviewView = map[objc.ID]*Panel{}
	panelsByPopover     = map[objc.ID]*Panel{}
)

func registerPreviewTargets() {
	var err error
	previewViewClass, err = objc.RegisterClass("EcoPastePreviewView", class("NSView"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("acceptsFirstResponder"), Fn: func(_ objc.ID, _ objc.SEL) bool { return true }},
			{Cmd: sel("keyDown:"), Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
				p := panelsByPreviewView[self]
				if p != nil && objc.Send[uint16](event, sel("keyCode")) == keyEscape {
					p.cx.Emit("hidePreview", nil)
					return
				}
				sendSuper[objc.ID](self, previewViewClass, sel("keyDown:"), event)
			}},
			{Cmd: sel("cancelOperation:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if p := panelsByPreviewView[self]; p != nil {
					p.cx.Emit("hidePreview", nil)
				}
			}},
		})
	if err != nil {
		panic(err)
	}
	popoverDelegateClass, err = objc.RegisterClass("EcoPastePopoverDelegate", class("NSObject"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("popoverDidClose:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				p := panelsByPopover[self]
				// Re-anchoring to another row is not a dismissal; AppKit can
				// transiently reconfigure the popover while it moves.
				if p == nil || !p.previewing || p.previewReanchor {
					return
				}
				p.previewing = false
				p.previewRowIndex = -1
				p.cx.Emit("hidePreview", nil)
				window := p.view.Send(sel("window"))
				if window != 0 && !objc.Send[bool](window, sel("isKeyWindow")) &&
					!p.pinned && objc.Send[bool](window, sel("isVisible")) {
					p.cx.Emit("blurHide", nil)
				}
			}},
		})
	if err != nil {
		panic(err)
	}
}

func (p *Panel) buildPreview() {
	p.preview = objc.ID(previewViewClass).Send(sel("new"))
	p.preview.Send(sel("setAutoresizesSubviews:"), false)
	setFrame(p.preview, 0, 0, previewWidth, previewMinHeight)
	p.buildPreviewContent()
	p.buildPreviewPopover()
	panelsByPreviewView[p.preview] = p
	panelsByPopover[p.previewDelegate] = p
}

func (p *Panel) buildPreviewContent() {
	p.previewTitle = objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	p.previewTitle.Send(sel("setTag:"), int64(previewTitleTag))
	p.previewTitle.Send(sel("setFont:"), objc.ID(classNSFont).Send(sel("boldSystemFontOfSize:"), previewTitleFontSize))
	p.previewTitle.Send(sel("setLineBreakMode:"), int64(4))
	p.previewTitle.Send(sel("setMaximumNumberOfLines:"), int64(1))

	p.previewMeta = objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	p.previewMeta.Send(sel("setTag:"), int64(previewMetaTag))
	p.previewMeta.Send(sel("setFont:"), systemFont(previewMetaFontSize))
	p.previewMeta.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	p.previewMeta.Send(sel("setLineBreakMode:"), int64(4))
	p.previewMeta.Send(sel("setMaximumNumberOfLines:"), int64(1))

	p.previewText = objc.ID(class("NSTextView")).Send(sel("alloc")).Send(sel("initWithFrame:"),
		CGRect{})
	p.previewText.Send(sel("setEditable:"), false)
	p.previewText.Send(sel("setSelectable:"), true)
	p.previewText.Send(sel("setDrawsBackground:"), false)
	p.previewText.Send(sel("setFont:"), systemFont(previewTextFontSize))
	p.previewText.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("labelColor")))
	// Horizontal padding lives in the text view, not in the scroll view, so the
	// scroller stays pinned to the popover's right edge.
	p.previewText.Send(sel("setTextContainerInset:"), CGSize{Width: previewPadding, Height: previewTextInset})
	p.previewText.Send(sel("setVerticallyResizable:"), true)
	p.previewText.Send(sel("setHorizontallyResizable:"), false)
	p.previewText.Send(sel("setAutoresizingMask:"), int64(2))
	if container := p.previewText.Send(sel("textContainer")); container != 0 {
		container.Send(sel("setWidthTracksTextView:"), true)
		container.Send(sel("setContainerSize:"), CGSize{Width: 10000, Height: 10000})
		container.Send(sel("setLineFragmentPadding:"), 0.0)
	}

	// Full-width scroll view keeps its scroller on the right edge; the text
	// inset above provides the left/right content padding without shrinking it.
	p.previewScroll = newPopoverContentScrollView()
	p.previewScroll.Send(sel("setDocumentView:"), p.previewText)
	p.previewText.Send(sel("release"))

	p.previewImage = objc.ID(class("NSImageView")).Send(sel("new"))
	p.previewImage.Send(sel("setImageScaling:"), int64(3))
	p.previewImage.Send(sel("setImageAlignment:"), int64(5))
	p.previewImage.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.PreviewImageLabel)))

	// No backdrop or divider: the popover's own translucent chrome is the
	// only surface, so the content sits directly on the main panel material.
	p.preview.Send(sel("addSubview:"), p.previewTitle)
	p.preview.Send(sel("addSubview:"), p.previewMeta)
	p.preview.Send(sel("addSubview:"), p.previewScroll)
	p.preview.Send(sel("addSubview:"), p.previewImage)
	p.previewScroll.Send(sel("release"))
	p.previewImage.Send(sel("release"))
}

func (p *Panel) buildPreviewPopover() {
	p.previewController = newPopoverController(p.preview)

	p.previewPopover = objc.ID(class("NSPopover")).Send(sel("new"))
	p.previewPopover.Send(sel("setBehavior:"), int64(1))
	p.previewPopover.Send(sel("setAnimates:"), true)
	p.previewPopover.Send(sel("setContentViewController:"), p.previewController)

	p.previewDelegate = objc.ID(popoverDelegateClass).Send(sel("new"))
	p.previewPopover.Send(sel("setDelegate:"), p.previewDelegate)
}

func (p *Panel) setPreview(content PreviewContent) {
	if !content.Active {
		if p.previewing {
			p.previewing = false
			p.previewRowIndex = -1
			p.closePreview()
			p.updateEmptyState()
			p.makeSearchFirstResponder()
		}
		return
	}

	p.previewing = true
	p.previewTitle.Send(sel("setStringValue:"), nsString(content.Title))
	p.previewMeta.Send(sel("setStringValue:"), nsString(content.Meta))

	image := objc.ID(0)
	if content.ImagePath != "" {
		image = loadImage(content.ImagePath)
	}
	width, height := previewSize(content, image, p.maxPreviewHeight())
	setPopoverContentSize(p.previewPopover, p.preview, width, height)
	p.setTextPreview(content)
	p.setPreviewImage(image, width, height, content.Meta)
	layoutPreview(p, content, width, height)
	p.showPreviewAtSelection()

	p.updateEmptyState()
}

// showPreviewAtSelection shows the popover anchored to the selected row, or
// re-anchors an already visible popover when keyboard navigation moved the
// selection to a different row. Keeping the anchor on the current row makes
// the preview follow the highlight instead of pointing at the previous item.
func (p *Panel) showPreviewAtSelection() {
	index := p.selectedIndex()
	shown := objc.Send[bool](p.previewPopover, sel("isShown"))
	if shown {
		if index == p.previewRowIndex {
			return
		}
		// Re-anchoring an open popover is not a dismissal; guard the close
		// delegate in case AppKit briefly reconfigures the popover.
		p.previewReanchor = true
	}
	showPopover(p.previewPopover, p.table, p.previewAnchor(), uint64(2))
	p.previewReanchor = false
	p.previewRowIndex = index
}

func (p *Panel) setPreviewImage(image objc.ID, width, height float64, meta string) {
	hasImage := image != 0
	p.previewScroll.Send(sel("setHidden:"), hasImage)
	p.previewImage.Send(sel("setHidden:"), !hasImage)
	if !hasImage {
		p.previewImage.Send(sel("setImage:"), objc.ID(0))
		p.previewImageSize = CGSize{}
		return
	}
	size := objc.Send[CGSize](image, sel("size"))
	if size.Width <= 0 || size.Height <= 0 {
		p.previewImage.Send(sel("setImage:"), objc.ID(0))
		p.previewImageSize = CGSize{}
		return
	}
	scale := math.Min(1, math.Min((width-2*previewPadding)/size.Width,
		previewContentHeight(height, meta)/size.Height))
	p.previewImageSize = CGSize{Width: size.Width * scale, Height: size.Height * scale}
	p.previewImage.Send(sel("setImage:"), image)
}

func (p *Panel) closePreview() {
	if p.previewPopover != 0 && objc.Send[bool](p.previewPopover, sel("isShown")) {
		p.previewPopover.Send(sel("close"))
	}
}

func (p *Panel) previewAnchor() CGRect {
	if p.table == 0 {
		return CGRect{Size: CGSize{Width: 1, Height: 1}}
	}
	index := p.selectedIndex()
	if index < 0 {
		rect := bounds(p.table)
		return CGRect{
			Origin: CGPoint{X: rect.Size.Width - 1, Y: rect.Size.Height / 2},
			Size:   CGSize{Width: 1, Height: 1},
		}
	}
	return objc.Send[CGRect](p.table, sel("rectOfRow:"), int64(index))
}

// previewKeyAction maps a key press to an action while the preview popover
// is open. Space and Escape explicitly close it, Return activates the
// selected row, and every other key reuses the list keyboard contract.
func previewKeyAction(key uint16, flags uint64, row HistoryRow, spacePreview bool) (string, bool) {
	switch key {
	case keyEscape, keySpace:
		return "hide", true
	case keyReturn, keyEnter:
		return "activate", true
	}
	return historyKeyAction(key, flags, row, spacePreview)
}

// selectedRow returns the row marked as selected, or an empty row when the
// list has no selection.
func (p *Panel) selectedRow() HistoryRow {
	if index := p.selectedIndex(); index >= 0 {
		return p.rows[index]
	}
	return HistoryRow{}
}

// handlePreviewKey keeps the list's keyboard contract while the popover
// holds focus. Navigation moves the selection and refreshes the preview in
// place instead of dismissing it; only Space, Escape, Return and the item
// actions close the popover.
func (p *Panel) handlePreviewKey(key uint16, flags uint64, repeat bool) bool {
	if !p.previewing {
		return false
	}
	row := p.selectedRow()
	action, handled := previewKeyAction(key, flags, row, p.behavior.SpacePreview)
	if !handled {
		return false
	}
	switch action {
	case "hide", "preview":
		// A held Space auto-repeats; only the first press toggles so the
		// popover cannot flicker open and shut.
		if repeat && key == keySpace {
			return true
		}
		p.cx.Emit("hidePreview", nil)
	case "focusSearch":
		p.cx.Emit("hidePreview", nil)
		p.focusSearch()
	case "firstRow", "lastRow", "pageUp", "pageDown", "previousRow", "nextRow":
		before := p.selection
		p.applyNavigation(action)
		if p.selection != before {
			p.cx.Emit("previewSelection", p.selection)
		}
	case "activate":
		if repeat {
			return true
		}
		p.cx.Emit("hidePreview", nil)
		p.cx.Emit("activate", p.selection)
	case "delete":
		if repeat {
			return true
		}
		if p.deleteRow(row) {
			p.cx.Emit("hidePreview", nil)
		}
	default:
		p.cx.Emit("hidePreview", nil)
		p.cx.Emit(action, p.selection)
	}
	return true
}

func (p *Panel) setTextPreview(content PreviewContent) {
	if attr := attributedPreview(content.RichText, content.RichFormat); attr != 0 {
		if storage := p.previewText.Send(sel("textStorage")); storage != 0 {
			storage.Send(sel("setAttributedString:"), attr)
			p.previewText.Send(sel("scrollRangeToVisible:"), NSRange{})
		}
		return
	}
	if content.Text == "" {
		p.previewText.Send(sel("setString:"), nsString(""))
		return
	}
	if content.Kind == "files" {
		p.setFilePreviewText(content.Text)
		return
	}
	// Build plain preview text with the same type scale and line spacing the
	// height measurement uses, so short text sits tight and long text reads
	// comfortably.
	paragraph := previewParagraphStyle()
	attributes := objc.ID(class("NSMutableDictionary")).Send(sel("dictionary"))
	attributes.Send(sel("setObject:forKey:"), systemFont(previewTextFontSize), nsString("NSFont"))
	attributes.Send(sel("setObject:forKey:"), objc.ID(classNSColor).Send(sel("labelColor")), nsString("NSColor"))
	attributes.Send(sel("setObject:forKey:"), paragraph, nsString("NSParagraphStyle"))
	paragraph.Send(sel("release"))
	attr := objc.ID(class("NSAttributedString")).Send(sel("alloc")).
		Send(sel("initWithString:attributes:"), nsString(content.Text), attributes)
	if attr != 0 {
		if storage := p.previewText.Send(sel("textStorage")); storage != 0 {
			storage.Send(sel("setAttributedString:"), attr)
		}
		attr.Send(sel("release"))
	}
	p.previewText.Send(sel("scrollRangeToVisible:"), NSRange{})
}

// setFilePreviewText renders a file list with a clear hierarchy: the file name
// line uses the primary label at body size, while each path line drops to the
// secondary color at a smaller size. It avoids cards, icons or bullets so the
// list stays as light as the rest of the preview.
func (p *Panel) setFilePreviewText(text string) {
	lines := strings.Split(text, "\n")
	paragraph := previewParagraphStyle()
	result := objc.ID(class("NSMutableAttributedString")).Send(sel("alloc")).
		Send(sel("initWithString:"), nsString(""))
	for index, line := range lines {
		body := line
		if index < len(lines)-1 {
			body += "\n"
		}
		if body == "" {
			continue
		}
		font := systemFont(previewTextFontSize)
		color := objc.ID(classNSColor).Send(sel("labelColor"))
		if isPreviewPathLine(line) {
			font = systemFont(previewMetaFontSize)
			color = objc.ID(classNSColor).Send(sel("secondaryLabelColor"))
		}
		attributes := objc.ID(class("NSMutableDictionary")).Send(sel("dictionary"))
		attributes.Send(sel("setObject:forKey:"), font, nsString("NSFont"))
		attributes.Send(sel("setObject:forKey:"), color, nsString("NSColor"))
		attributes.Send(sel("setObject:forKey:"), paragraph, nsString("NSParagraphStyle"))
		piece := objc.ID(class("NSAttributedString")).Send(sel("alloc")).
			Send(sel("initWithString:attributes:"), nsString(body), attributes)
		result.Send(sel("appendAttributedString:"), piece)
		piece.Send(sel("release"))
	}
	paragraph.Send(sel("release"))
	if storage := p.previewText.Send(sel("textStorage")); storage != 0 {
		storage.Send(sel("setAttributedString:"), result)
	}
	result.Send(sel("release"))
	p.previewText.Send(sel("scrollRangeToVisible:"), NSRange{})
}

// isPreviewPathLine reports whether a file-list line is a filesystem path
// rather than the name/detail headline.
func isPreviewPathLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, "~")
}

// previewParagraphStyle is the shared line rhythm for plain preview text.
// Height measurement uses the same style so the popover never clips a line.
func previewParagraphStyle() objc.ID {
	style := objc.ID(class("NSMutableParagraphStyle")).Send(sel("new"))
	style.Send(sel("setLineSpacing:"), previewLineSpacing)
	style.Send(sel("setParagraphSpacing:"), 0.0)
	return style
}

func attributedPreview(text, format string) objc.ID {
	if text == "" {
		return 0
	}
	switch format {
	case "html":
		return htmlPreview(text)
	case "rtf":
		data := nsData([]byte(text))
		return objc.ID(class("NSAttributedString")).Send(sel("alloc")).
			Send(sel("initWithRTF:documentAttributes:"), data, objc.ID(0))
	default:
		return 0
	}
}

// htmlPreview imports clipboard HTML and then flattens the parts AppKit
// repeats at draw time. The importer already inserts list markers into the
// string, so leaving the paragraph text lists in place draws them twice.
func htmlPreview(text string) objc.ID {
	data := nsData([]byte(normalizePreviewHTML(styledPreviewHTML(text))))
	parsed := objc.ID(class("NSAttributedString")).Send(sel("alloc")).
		Send(sel("initWithHTML:documentAttributes:"), data, objc.ID(0))
	if parsed == 0 {
		return 0
	}
	result := parsed.Send(sel("mutableCopy"))
	parsed.Send(sel("release"))
	if result == 0 {
		return 0
	}
	full := NSRange{Length: uint64(objc.Send[uint](result, sel("length")))}
	if full.Length > 0 {
		flattenPreviewLists(result, full)
		applyPreviewDefaultColor(result, full)
		flattenPreviewLinks(result, full)
	}
	return result.Send(sel("autorelease"))
}

// styledPreviewHTML supplies native typography defaults only. Clipboard
// markup keeps its own inline families, sizes, emphasis and lists.
func styledPreviewHTML(text string) string {
	lower := strings.ToLower(text)
	if end := strings.Index(lower, "</head>"); end >= 0 {
		return text[:end] + previewHTMLStyle + text[end:]
	}
	return previewHTMLStyle + text
}

type NSRange struct {
	Location uint64
	Length   uint64
}

// applyPreviewDefaultColor keeps explicit clipboard colors while making
// unstyled HTML readable in both light and dark appearances.
func applyPreviewDefaultColor(text objc.ID, full NSRange) {
	dark := previewUsesDarkAppearance()
	block := objc.NewBlock(func(_ objc.Block, value objc.ID, r NSRange, _ *bool) {
		if r.Length == 0 {
			return
		}
		if value == 0 || previewColorNeedsLabel(value, dark) {
			text.Send(sel("addAttribute:value:range:"), nsString("NSColor"),
				objc.ID(classNSColor).Send(sel("labelColor")), r)
		}
	})
	text.Send(sel("enumerateAttribute:inRange:options:usingBlock:"),
		nsString("NSColor"), full, uint64(0), objc.ID(block))
	block.Release()
}

func previewUsesDarkAppearance() bool {
	app := objc.ID(class("NSApplication")).Send(sel("sharedApplication"))
	if app == 0 {
		return false
	}
	appearance := app.Send(sel("effectiveAppearance"))
	if appearance == 0 {
		return false
	}
	return strings.Contains(strings.ToLower(goString(appearance.Send(sel("name")))), "dark")
}

func previewColorNeedsLabel(color objc.ID, dark bool) bool {
	rgb := color.Send(sel("colorUsingColorSpaceName:"),
		nsString("NSCalibratedRGBColorSpace"))
	if rgb == 0 {
		return false
	}
	return previewInkNeedsLabel(
		objc.Send[float64](rgb, sel("redComponent")),
		objc.Send[float64](rgb, sel("greenComponent")),
		objc.Send[float64](rgb, sel("blueComponent")),
		dark,
	)
}

func previewInkNeedsLabel(red, green, blue float64, dark bool) bool {
	low := math.Min(red, math.Min(green, blue))
	high := math.Max(red, math.Max(green, blue))
	if high-low > 0.03 {
		return false
	}
	if dark {
		return high < 0.22
	}
	return low > 0.92
}

func flattenPreviewLists(text objc.ID, full NSRange) {
	block := objc.NewBlock(func(_ objc.Block, value objc.ID, r NSRange, _ *bool) {
		if value == 0 || r.Length == 0 {
			return
		}
		style := value.Send(sel("mutableCopy"))
		style.Send(sel("setTextLists:"), objc.ID(class("NSArray")).Send(sel("array")))
		style.Send(sel("setLineSpacing:"), 2.0)
		style.Send(sel("setParagraphSpacing:"), 6.0)
		text.Send(sel("addAttribute:value:range:"), nsString("NSParagraphStyle"), style, r)
		style.Send(sel("release"))
	})
	text.Send(sel("enumerateAttribute:inRange:options:usingBlock:"),
		nsString("NSParagraphStyle"), full, uint64(0), objc.ID(block))
	block.Release()
}

func flattenPreviewLinks(text objc.ID, full NSRange) {
	block := objc.NewBlock(func(_ objc.Block, value objc.ID, r NSRange, _ *bool) {
		if value == 0 || r.Length == 0 {
			return
		}
		text.Send(sel("addAttribute:value:range:"), nsString("NSColor"),
			objc.ID(classNSColor).Send(sel("linkColor")), r)
	})
	text.Send(sel("enumerateAttribute:inRange:options:usingBlock:"),
		nsString("NSLink"), full, uint64(0), objc.ID(block))
	block.Release()
}

// normalizePreviewHTML removes the paragraph wrappers the clipboard HTML
// routinely puts inside list items, so imported list items keep one
// paragraph style instead of nested styles with their own spacing.
func normalizePreviewHTML(text string) string {
	replacements := []struct{ old, new string }{
		{"<li><p>", "<li>"},
		{"</p></li>", "</li>"},
		{"<li><p ", "<li "},
	}
	for _, replacement := range replacements {
		text = strings.ReplaceAll(text, replacement.old, replacement.new)
	}
	return text
}

// previewHeaderHeight is the title/meta block above the body. The meta line
// is only reserved when there is metadata to show.
func previewHeaderHeight(meta string) float64 {
	if meta == "" {
		return previewTitleHeight
	}
	return previewTitleHeight + previewHeaderGap + previewMetaHeight
}

// previewChrome is everything outside the body: outer padding, header and the
// gap between header and body.
func previewChrome(meta string) float64 {
	return 2*previewPadding + previewHeaderHeight(meta) + previewContentGap
}

// maxPreviewHeight keeps the popover inside the screen it is anchored to,
// leaving room for the anchor row and menu bar instead of running off-screen.
func (p *Panel) maxPreviewHeight() float64 {
	limit := previewMaxHeight
	if window := p.view.Send(sel("window")); window != 0 {
		if screen := objc.Send[objc.ID](window, sel("screen")); screen != 0 {
			frame := objc.Send[CGRect](screen, sel("visibleFrame"))
			if frame.Size.Height > 0 {
				available := frame.Size.Height - previewScreenMargin
				if available < limit {
					limit = available
				}
			}
		}
	}
	return clampFloat(limit, previewImageMinHeight, previewMaxHeight)
}

func previewSize(content PreviewContent, image objc.ID, maxHeight float64) (float64, float64) {
	if maxHeight < previewMinHeight {
		maxHeight = previewMinHeight
	}
	chrome := previewChrome(content.Meta)
	if image != 0 {
		size := objc.Send[CGSize](image, sel("size"))
		if size.Width > 0 && size.Height > 0 {
			availWidth := previewMaxWidth - 2*previewPadding
			availHeight := math.Max(1, maxHeight-chrome)
			scale := math.Min(1, math.Min(availWidth/size.Width, availHeight/size.Height))
			width := clampFloat(size.Width*scale+2*previewPadding, previewMinWidth, previewMaxWidth)
			height := clampFloat(size.Height*scale+chrome, previewImageMinHeight, maxHeight)
			return width, height
		}
	}
	bodyHeight := previewTextHeight(content)
	return previewWidth, clampFloat(chrome+bodyHeight, previewMinHeight, maxHeight)
}

func previewTextHeight(content PreviewContent) float64 {
	width := previewWidth - 2*previewPadding
	if attr := attributedPreview(content.RichText, content.RichFormat); attr != 0 {
		rect := objc.Send[CGRect](attr, sel("boundingRectWithSize:options:"),
			CGSize{Width: width, Height: 10000}, uint64(3))
		return rect.Size.Height + 2*previewTextInset
	}
	if content.Text == "" {
		return 40
	}
	paragraph := previewParagraphStyle()
	defer paragraph.Send(sel("release"))
	attributes := objc.ID(class("NSMutableDictionary")).Send(sel("dictionary"))
	attributes.Send(sel("setObject:forKey:"), systemFont(previewTextFontSize), nsString("NSFont"))
	attributes.Send(sel("setObject:forKey:"),
		objc.ID(classNSColor).Send(sel("labelColor")), nsString("NSColor"))
	attributes.Send(sel("setObject:forKey:"), paragraph, nsString("NSParagraphStyle"))
	rect := objc.Send[CGRect](nsString(content.Text), sel("boundingRectWithSize:options:attributes:"),
		CGSize{Width: width, Height: 10000}, uint64(3), attributes)
	return rect.Size.Height + 2*previewTextInset
}

func (p *Panel) makeSearchFirstResponder() {
	window := p.view.Send(sel("window"))
	if window != 0 {
		window.Send(sel("makeFirstResponder:"), p.search)
	}
}

func layoutPreview(p *Panel, content PreviewContent, width, height float64) {
	headerHeight := previewHeaderHeight(content.Meta)
	titleY := height - previewPadding - previewTitleHeight
	setFrame(p.previewTitle, previewPadding, titleY, width-2*previewPadding, previewTitleHeight)
	if content.Meta != "" {
		metaY := titleY - previewHeaderGap - previewMetaHeight
		setFrame(p.previewMeta, previewPadding, metaY, width-2*previewPadding, previewMetaHeight)
		p.previewMeta.Send(sel("setHidden:"), false)
	} else {
		p.previewMeta.Send(sel("setHidden:"), true)
	}

	contentY := previewPadding
	contentHeight := height - 2*previewPadding - headerHeight - previewContentGap
	if contentHeight < 0 {
		contentHeight = 0
	}
	layoutPopoverContentScroll(p.previewScroll, contentY, width, contentHeight)
	setFrame(p.previewImage,
		(width-p.previewImageSize.Width)/2,
		contentY+(contentHeight-p.previewImageSize.Height)/2,
		p.previewImageSize.Width, p.previewImageSize.Height)
}

func previewContentHeight(height float64, meta string) float64 {
	content := height - previewChrome(meta)
	if content < 0 {
		return 0
	}
	return content
}

func clampFloat(value, low, high float64) float64 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

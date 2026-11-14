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

	previewPadding = 16.0
	previewWidth   = 380.0
	previewChrome  = 70.0
	previewGap     = 8.0
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
				if p == nil || !p.previewing {
					return
				}
				p.previewing = false
				p.hint.Send(sel("setStringValue:"), nsString(i18n.T(i18n.PanelHint)))
				p.cx.Emit("hidePreview", nil)
				window := p.view.Send(sel("window"))
				if window != 0 && !objc.Send[bool](window, sel("isKeyWindow")) &&
					!p.pinned && !p.onboardingActive &&
					objc.Send[bool](window, sel("isVisible")) {
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

	backdrop := objc.ID(class("NSVisualEffectView")).Send(sel("new"))
	backdrop.Send(sel("setMaterial:"), int64(10))
	backdrop.Send(sel("setBlendingMode:"), int64(1))
	backdrop.Send(sel("setState:"), int64(1))
	backdrop.Send(sel("setAutoresizingMask:"), int64(18))
	p.preview.Send(sel("addSubview:"), backdrop)
	p.previewBackdrop = backdrop
	backdrop.Send(sel("release"))

	p.previewTitle = objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	p.previewTitle.Send(sel("setTag:"), int64(previewTitleTag))
	p.previewTitle.Send(sel("setFont:"), objc.ID(classNSFont).Send(sel("boldSystemFontOfSize:"), 14.0))
	p.previewTitle.Send(sel("setLineBreakMode:"), int64(4))
	p.previewTitle.Send(sel("setMaximumNumberOfLines:"), int64(1))

	p.previewMeta = objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	p.previewMeta.Send(sel("setTag:"), int64(previewMetaTag))
	p.previewMeta.Send(sel("setFont:"), systemFont(11))
	p.previewMeta.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	p.previewMeta.Send(sel("setLineBreakMode:"), int64(4))
	p.previewMeta.Send(sel("setMaximumNumberOfLines:"), int64(1))

	p.previewDivider = objc.ID(classNSBox).Send(sel("new"))
	p.previewDivider.Send(sel("setBoxType:"), 2)
	p.previewDivider.Send(sel("setTitlePosition:"), 0)

	p.previewText = objc.ID(class("NSTextView")).Send(sel("alloc")).Send(sel("initWithFrame:"),
		CGRect{})
	p.previewText.Send(sel("setEditable:"), false)
	p.previewText.Send(sel("setSelectable:"), true)
	p.previewText.Send(sel("setDrawsBackground:"), false)
	p.previewText.Send(sel("setFont:"), systemFont(13))
	p.previewText.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("labelColor")))
	p.previewText.Send(sel("setTextContainerInset:"), CGSize{Width: 0, Height: 2})
	p.previewText.Send(sel("setVerticallyResizable:"), true)
	p.previewText.Send(sel("setHorizontallyResizable:"), false)
	p.previewText.Send(sel("setAutoresizingMask:"), int64(2))
	if container := p.previewText.Send(sel("textContainer")); container != 0 {
		container.Send(sel("setWidthTracksTextView:"), true)
		container.Send(sel("setContainerSize:"), CGSize{Width: 10000, Height: 10000})
		container.Send(sel("setLineFragmentPadding:"), 0.0)
	}

	p.previewScroll = objc.ID(class("NSScrollView")).Send(sel("new"))
	p.previewScroll.Send(sel("setHasVerticalScroller:"), true)
	p.previewScroll.Send(sel("setAutohidesScrollers:"), true)
	p.previewScroll.Send(sel("setDrawsBackground:"), false)
	p.previewScroll.Send(sel("setBorderType:"), int64(0))
	p.previewScroll.Send(sel("setDocumentView:"), p.previewText)

	p.previewImage = objc.ID(class("NSImageView")).Send(sel("new"))
	p.previewImage.Send(sel("setImageScaling:"), int64(3))
	p.previewImage.Send(sel("setImageAlignment:"), int64(5))
	p.previewImage.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.PreviewImageLabel)))

	p.preview.Send(sel("addSubview:"), p.previewTitle)
	p.preview.Send(sel("addSubview:"), p.previewMeta)
	p.preview.Send(sel("addSubview:"), p.previewDivider)
	p.preview.Send(sel("addSubview:"), p.previewScroll)
	p.preview.Send(sel("addSubview:"), p.previewImage)

	p.previewController = objc.ID(class("NSViewController")).Send(sel("new"))
	p.previewController.Send(sel("setView:"), p.preview)

	p.previewPopover = objc.ID(class("NSPopover")).Send(sel("new"))
	p.previewPopover.Send(sel("setBehavior:"), int64(1))
	p.previewPopover.Send(sel("setAnimates:"), true)
	p.previewPopover.Send(sel("setContentViewController:"), p.previewController)

	p.previewDelegate = objc.ID(popoverDelegateClass).Send(sel("new"))
	p.previewPopover.Send(sel("setDelegate:"), p.previewDelegate)

	panelsByPreviewView[p.preview] = p
	panelsByPopover[p.previewDelegate] = p
}

func (p *Panel) setPreview(content PreviewContent) {
	if !content.Active {
		if p.previewing {
			p.previewing = false
			p.hint.Send(sel("setStringValue:"), nsString(i18n.T(i18n.PanelHint)))
			p.closePreview()
			p.updateStatus()
			p.makeSearchFirstResponder()
		}
		return
	}

	p.previewing = true
	p.hint.Send(sel("setStringValue:"), nsString(i18n.T(i18n.PreviewHintBack)))
	p.previewTitle.Send(sel("setStringValue:"), nsString(content.Title))
	p.previewMeta.Send(sel("setStringValue:"), nsString(content.Meta))

	image := objc.ID(0)
	if content.ImagePath != "" {
		image = loadImage(content.ImagePath)
	}
	width, height := previewSize(content, image)
	p.previewPopover.Send(sel("setContentSize:"), CGSize{Width: width, Height: height})
	p.setTextPreview(content)
	p.setPreviewImage(image, width, height)
	layoutPreview(p, width, height)

	if !objc.Send[bool](p.previewPopover, sel("isShown")) {
		p.previewPopover.Send(sel("showRelativeToRect:ofView:preferredEdge:"),
			p.previewAnchor(), p.table, uint64(2))
	}
	p.updateStatus()
}

func (p *Panel) setPreviewImage(image objc.ID, width, height float64) {
	hasImage := image != 0
	p.previewScroll.Send(sel("setHidden:"), hasImage)
	p.previewImage.Send(sel("setHidden:"), !hasImage)
	if !hasImage {
		p.previewImage.Send(sel("setImage:"), objc.ID(0))
		p.previewImageSize = CGSize{}
		return
	}
	size := objc.Send[CGSize](image, sel("size"))
	scale := math.Min(1, math.Min((width-2*previewPadding)/size.Width,
		previewContentHeight(height)/size.Height))
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

// handlePreviewKey keeps the list's keyboard contract while the popover
// holds focus: Space and Escape dismiss it, Return copies, and the
// command shortcuts still act on the selected row.
func (p *Panel) handlePreviewKey(key uint16, flags uint64) bool {
	if !p.previewing {
		return false
	}
	switch key {
	case keySpace, keyEscape:
		p.cx.Emit("hidePreview", nil)
		return true
	case keyReturn, keyEnter:
		p.cx.Emit("hidePreview", nil)
		p.cx.Emit("activate", p.selection)
		return true
	}
	row := HistoryRow{}
	for _, candidate := range p.rows {
		if candidate.ID == p.selection {
			row = candidate
			break
		}
	}
	action, handled := historyKeyAction(key, flags, row, p.behavior.SpacePreview)
	if !handled {
		return false
	}
	switch action {
	case "hide", "preview":
		p.cx.Emit("hidePreview", nil)
	case "focusSearch":
		p.cx.Emit("hidePreview", nil)
		p.focusSearch()
	case "delete":
		if row.CanDelete && (!row.DeleteConfirm || confirmDelete()) {
			p.cx.Emit("hidePreview", nil)
			p.cx.Emit("delete", p.selection)
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
		}
		return
	}
	p.previewText.Send(sel("setString:"), nsString(content.Text))
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
	data := nsData([]byte(normalizePreviewHTML(text)))
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
		result.Send(sel("addAttribute:value:range:"), nsString("NSColor"),
			objc.ID(classNSColor).Send(sel("labelColor")), full)
		flattenPreviewLinks(result, full)
	}
	return result.Send(sel("autorelease"))
}

type NSRange struct {
	Location uint64
	Length   uint64
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

func previewSize(content PreviewContent, image objc.ID) (float64, float64) {
	if image != 0 {
		size := objc.Send[CGSize](image, sel("size"))
		width := clampFloat(size.Width+2*previewPadding, 260, 420)
		height := size.Height
		if size.Width > 0 {
			height = size.Height * math.Min(1, (width-2*previewPadding)/size.Width)
		}
		return width, clampFloat(height, 150, 380) + previewChrome + previewGap
	}
	height := previewTextHeight(content)
	return previewWidth, clampFloat(height+previewChrome+previewGap, 188, 460)
}

func previewTextHeight(content PreviewContent) float64 {
	width := previewWidth - 2*previewPadding
	if attr := attributedPreview(content.RichText, content.RichFormat); attr != 0 {
		rect := objc.Send[CGRect](attr, sel("boundingRectWithSize:options:"),
			CGSize{Width: width, Height: 10000}, uint64(3))
		return rect.Size.Height + 8
	}
	if content.Text == "" {
		return 120
	}
	attributes := objc.ID(class("NSMutableDictionary")).Send(sel("dictionary"))
	attributes.Send(sel("setObject:forKey:"), systemFont(13), nsString("NSFont"))
	attributes.Send(sel("setObject:forKey:"),
		objc.ID(classNSColor).Send(sel("labelColor")), nsString("NSColor"))
	rect := objc.Send[CGRect](nsString(content.Text), sel("boundingRectWithSize:options:attributes:"),
		CGSize{Width: width, Height: 10000}, uint64(3), attributes)
	return rect.Size.Height + 8
}

func (p *Panel) makeSearchFirstResponder() {
	window := p.view.Send(sel("window"))
	if window != 0 {
		window.Send(sel("makeFirstResponder:"), p.search)
	}
}

func layoutPreview(p *Panel, width, height float64) {
	setFrame(p.preview, 0, 0, width, height)
	setFrame(p.previewBackdrop, 0, 0, width, height)
	setFrame(p.previewTitle, previewPadding, height-30, width-2*previewPadding, 18)
	setFrame(p.previewMeta, previewPadding, height-48, width-2*previewPadding, 15)
	setFrame(p.previewDivider, previewPadding, height-57, width-2*previewPadding, 1)

	contentY := 12.0
	contentHeight := previewContentHeight(height)
	setFrame(p.previewScroll, previewPadding, contentY, width-2*previewPadding, contentHeight)
	setFrame(p.previewImage,
		(width-p.previewImageSize.Width)/2,
		contentY+(contentHeight-p.previewImageSize.Height)/2,
		p.previewImageSize.Width, p.previewImageSize.Height)
}

func previewContentHeight(height float64) float64 {
	return height - previewChrome - previewGap
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

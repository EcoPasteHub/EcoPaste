//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

// PreviewContent is the in-panel content preview.
type PreviewContent struct {
	Active    bool
	Title     string
	Meta      string
	Text      string
	ImagePath string
}

const (
	previewTitleTag = 200
	previewMetaTag  = 201
)

var (
	previewViewClass objc.Class
	previewTextClass objc.Class

	panelsByPreviewView = map[objc.ID]*Panel{}
	panelsByPreviewText = map[objc.ID]*Panel{}
)

func registerPreviewTargets() {
	var err error
	previewViewClass, err = objc.RegisterClass("EcoPastePreviewView", class("NSView"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("acceptsFirstResponder"), Fn: func(_ objc.ID, _ objc.SEL) bool { return true }},
			{Cmd: sel("keyDown:"), Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
				p := panelsByPreviewView[self]
				if p != nil && objc.Send[uint16](event, sel("keyCode")) == 53 {
					p.cx.Emit("hidePreview", nil)
					return
				}
				sendSuper[objc.ID](self, previewViewClass, sel("keyDown:"), event)
			}},
		})
	if err != nil {
		panic(err)
	}

	previewTextClass, err = objc.RegisterClass("EcoPastePreviewText", class("NSTextView"), nil, nil,
		[]objc.MethodDef{{Cmd: sel("keyDown:"), Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
			p := panelsByPreviewText[self]
			if p != nil && objc.Send[uint16](event, sel("keyCode")) == 53 {
				p.cx.Emit("hidePreview", nil)
				return
			}
			sendSuper[objc.ID](self, previewTextClass, sel("keyDown:"), event)
		}}})
	if err != nil {
		panic(err)
	}
}

func (p *Panel) buildPreview() {
	p.preview = objc.ID(previewViewClass).Send(sel("new"))
	p.preview.Send(sel("setAutoresizesSubviews:"), false)
	p.preview.Send(sel("setHidden:"), true)

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

	p.previewText = objc.ID(previewTextClass).Send(sel("alloc")).Send(sel("initWithFrame:"), CGRect{})
	p.previewText.Send(sel("setEditable:"), false)
	p.previewText.Send(sel("setSelectable:"), true)
	p.previewText.Send(sel("setRichText:"), false)
	p.previewText.Send(sel("setDrawsBackground:"), false)
	p.previewText.Send(sel("setFont:"), systemFont(13))
	p.previewText.Send(sel("setTextContainerInset:"), CGSize{Width: 10, Height: 10})
	p.previewText.Send(sel("setVerticallyResizable:"), true)
	p.previewText.Send(sel("setHorizontallyResizable:"), false)
	p.previewText.Send(sel("setAutoresizingMask:"), int64(2))
	if container := p.previewText.Send(sel("textContainer")); container != 0 {
		container.Send(sel("setWidthTracksTextView:"), true)
		container.Send(sel("setContainerSize:"), CGSize{Width: 10000, Height: 10000})
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
	p.previewImage.Send(sel("setAccessibilityLabel:"), nsString("剪贴板图片预览"))

	p.preview.Send(sel("addSubview:"), p.previewTitle)
	p.preview.Send(sel("addSubview:"), p.previewMeta)
	p.preview.Send(sel("addSubview:"), p.previewScroll)
	p.preview.Send(sel("addSubview:"), p.previewImage)

	panelsByPreviewView[p.preview] = p
	panelsByPreviewText[p.previewText] = p
}

func (p *Panel) setPreview(content PreviewContent) {
	p.previewing = content.Active
	hidden := p.previewing || p.onboardingActive
	p.search.Send(sel("setHidden:"), hidden)
	p.pin.Send(sel("setHidden:"), hidden)
	p.filter.Send(sel("setHidden:"), hidden)
	p.groupsPopup.Send(sel("setHidden:"), hidden)
	p.groupActions.Send(sel("setHidden:"), hidden)
	p.divider.Send(sel("setHidden:"), hidden)
	p.scroll.Send(sel("setHidden:"), hidden)
	p.preview.Send(sel("setHidden:"), !p.previewing)
	if !p.previewing {
		if p.onboardingActive {
			return
		}
		p.hint.Send(sel("setStringValue:"), nsString("↵ 复制   esc 关闭"))
		p.makeSearchFirstResponder()
		p.updateStatus()
		return
	}

	p.hint.Send(sel("setStringValue:"), nsString("esc 返回"))
	p.previewTitle.Send(sel("setStringValue:"), nsString(content.Title))
	p.previewMeta.Send(sel("setStringValue:"), nsString(content.Meta))

	image := objc.ID(0)
	if content.ImagePath != "" {
		image = loadImage(content.ImagePath)
	}
	hasImage := image != 0
	p.previewText.Send(sel("setString:"), nsString(content.Text))
	p.previewScroll.Send(sel("setHidden:"), hasImage)
	p.previewImage.Send(sel("setHidden:"), !hasImage)
	if hasImage {
		p.previewImage.Send(sel("setImage:"), image)
	} else {
		p.previewImage.Send(sel("setImage:"), objc.ID(0))
		p.makePreviewFirstResponder(p.previewText)
	}
	p.updateStatus()
}

func (p *Panel) makePreviewFirstResponder(view objc.ID) {
	window := p.view.Send(sel("window"))
	if window != 0 {
		window.Send(sel("makeFirstResponder:"), view)
	}
}

func (p *Panel) makeSearchFirstResponder() {
	window := p.view.Send(sel("window"))
	if window != 0 {
		window.Send(sel("makeFirstResponder:"), p.search)
	}
}

func layoutPreview(p *Panel, width, height float64) {
	const pad = 16.0
	setFrame(p.previewTitle, pad, height-30, width-2*pad, 18)
	setFrame(p.previewMeta, pad, height-48, width-2*pad, 15)
	contentHeight := height - 72
	setFrame(p.previewScroll, pad, 12, width-2*pad, contentHeight)
	setFrame(p.previewImage, pad, 12, width-2*pad, contentHeight)
}

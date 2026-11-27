//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

const (
	panelSidePadding  = 14.0
	panelFooterHeight = 30.0
	panelFooterText   = 16.0
	panelShortcutSize = 26.0

	// panelHeaderMargin is the visible gap the panel keeps around the
	// header row on every side: left of the search text, right of the
	// toolbar icons, above the row and between the row and the list.
	panelHeaderMargin = 16.0

	// panelIconInset is how far an SF Symbol keeps from the edge of its
	// button; the symbol is centered inside the button.
	panelIconInset = 6.5

	// panelSearchInset is how far the text keeps from the top-left
	// corner of the borderless search field, which draws its glyphs just
	// under the frame's top edge.
	panelSearchInset = 2.0

	// panelSearchInkHeight is the height of the placeholder glyphs in the
	// search field at the panel's system font size.
	panelSearchInkHeight = 14.0

	// panelHeaderFrameMargin is the gap from the panel's edge to the
	// header row's icon frames: the row's top and the toolbar's last
	// button. The glyphs add panelIconInset on top of it.
	panelHeaderFrameMargin = panelHeaderMargin - panelIconInset

	// panelHeaderSearchMargin is the gap from the panel's left edge to
	// the search field's frame.
	panelHeaderSearchMargin = panelHeaderMargin - panelSearchInset

	// panelHeaderListGap keeps the same visible margin between the
	// toolbar icons and the first card as the panel keeps on the other
	// sides of the header row.
	panelHeaderListGap = panelHeaderMargin - panelIconInset - historyRowInsetY - rowLeadingInset

	panelIconSize   = 26.0
	panelSearchText = 22.0
	panelIconGap    = 2.0
	panelToolbarGap = 10.0
)

// headerLayout holds the geometry of the panel's header row: the search
// field, the toolbar buttons and the top edge of the list below.
type headerLayout struct {
	searchX     float64
	searchY     float64
	searchWidth float64
	iconY       float64
	clearX      float64
	filterX     float64
	settingsX   float64
	pinX        float64
	listTop     float64
}

// headerToolbar holds the x of the header's icon buttons, from the clear
// button on the left to the pin on the right.
type headerToolbar struct {
	clear    float64
	filter   float64
	settings float64
	pin      float64
}

func layoutPanel(p *Panel, width, height float64) {
	if p.background != 0 {
		setFrame(p.background, 0, 0, width, height)
	}

	header := layoutHeader(width, height)
	setFrame(p.search, header.searchX, header.searchY, header.searchWidth, panelSearchText)
	setFrame(p.clearButton, header.clearX, header.iconY, panelIconSize, panelIconSize)
	setFrame(p.filterButton, header.filterX, header.iconY, panelIconSize, panelIconSize)
	setFrame(p.settingsButton, header.settingsX, header.iconY, panelIconSize, panelIconSize)
	setFrame(p.pin, header.pinX, header.iconY, panelIconSize, panelIconSize)

	layoutFooter(p, width)

	scrollY := panelFooterHeight + 4
	scrollHeight := header.listTop - scrollY
	setFrame(p.scroll, 0, scrollY, width, scrollHeight)
	if p.table != 0 {
		p.syncTableFrame()
	}

	layoutEmptyState(p, width, scrollY, scrollHeight)
}

func layoutFooter(p *Panel, width float64) {
	footerWidth := width - 2*panelSidePadding
	if p.errText != "" {
		p.shortcutButton.Send(sel("setHidden:"), true)
	} else {
		p.shortcutButton.Send(sel("setHidden:"), false)
		footerWidth -= panelShortcutSize + 8
	}
	setFrame(p.footer, panelSidePadding, (panelFooterHeight-panelFooterText)/2,
		footerWidth, panelFooterText)
	setFrame(p.shortcutButton, width-panelSidePadding-panelShortcutSize,
		(panelFooterHeight-panelShortcutSize)/2, panelShortcutSize, panelShortcutSize)
}

// layoutHeader places the header row. The toolbar icons set the height of
// the row, the search text is centered on the same axis and the list
// starts one margin below, so the row keeps the same visible gap on all
// four sides.
func layoutHeader(width, height float64) headerLayout {
	toolbar := headerToolbarX(width)
	headerTop := height - panelHeaderFrameMargin
	headerBottom := headerTop - panelIconSize
	// The placeholder is drawn just under the field's top edge, so the
	// field is placed to keep the text on the icons' axis.
	searchY := headerBottom + panelIconSize/2 + panelSearchInset +
		panelSearchInkHeight/2 - panelSearchText
	return headerLayout{
		searchX:     panelHeaderSearchMargin,
		searchY:     searchY,
		searchWidth: panelSearchWidth(width),
		iconY:       headerBottom,
		clearX:      toolbar.clear,
		filterX:     toolbar.filter,
		settingsX:   toolbar.settings,
		pinX:        toolbar.pin,
		listTop:     headerBottom - panelHeaderListGap,
	}
}

// headerToolbarX places the icon buttons from the right edge inwards.
func headerToolbarX(width float64) headerToolbar {
	pin := width - panelHeaderFrameMargin - panelIconSize
	settings := pin - panelIconGap - panelIconSize
	filter := settings - panelIconGap - panelIconSize
	clear := filter - panelIconGap - panelIconSize
	return headerToolbar{clear: clear, filter: filter, settings: settings, pin: pin}
}

// panelSearchWidth leaves room for the clear, filter, settings and pin buttons.
func panelSearchWidth(width float64) float64 {
	return headerToolbarX(width).clear - panelHeaderSearchMargin - panelToolbarGap
}

func layoutEmptyState(p *Panel, width, scrollY, scrollHeight float64) {
	if objc.Send[bool](p.empty, sel("isHidden")) {
		return
	}
	const titleHeight = 17.0
	const detailHeight = 15.0
	hasDetail := !objc.Send[bool](p.emptyDetail, sel("isHidden"))
	blockHeight := titleHeight
	blockHeight += emptyKubaiIconSize + 8
	if hasDetail {
		blockHeight += 4 + detailHeight
	}
	top := scrollY + scrollHeight/2 + blockHeight/2
	top -= emptyKubaiIconSize
	for _, icon := range p.emptyIcons {
		if objc.Send[bool](icon.view, sel("isHidden")) {
			continue
		}
		setFrame(icon.view, (width-emptyKubaiIconSize)/2, top, emptyKubaiIconSize, emptyKubaiIconSize)
	}
	top -= 8
	top -= titleHeight
	setFrame(p.empty, 20, top, width-40, titleHeight)
	if hasDetail {
		top -= 4 + detailHeight
		setFrame(p.emptyDetail, 20, top, width-40, detailHeight)
	}
}

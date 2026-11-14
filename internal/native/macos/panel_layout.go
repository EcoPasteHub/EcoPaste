//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

const (
	panelSidePadding  = 14.0
	panelSearchHeight = 34.0
	panelSearchText   = 22.0
	panelSearchIcon   = 18.0
	panelIconSize     = 26.0
	panelIconGap      = 2.0
	panelFooterHeight = 30.0
	panelTopPadding   = 14.0
	panelShortcutSize = 16.0
)

func layoutPanel(p *Panel, width, height float64) {
	if p.background != 0 {
		setFrame(p.background, 0, 0, width, height)
	}

	searchY := height - panelTopPadding - panelSearchHeight
	searchIconY := searchY + (panelSearchHeight-panelSearchIcon)/2
	searchTextY := searchY + (panelSearchHeight-panelSearchText)/2
	iconY := searchY + (panelSearchHeight-panelIconSize)/2
	pinX := width - panelSidePadding - panelIconSize
	settingsX := pinX - panelIconGap - panelIconSize
	filterX := settingsX - panelIconGap - panelIconSize
	setFrame(p.searchIcon, panelSidePadding, searchIconY, panelSearchIcon, panelSearchIcon)
	setFrame(p.search, panelSidePadding+panelSearchIcon+8, searchTextY,
		panelSearchWidth(width)-panelSearchIcon-8, panelSearchText)
	setFrame(p.filterButton, filterX, iconY, panelIconSize, panelIconSize)
	setFrame(p.settingsButton, settingsX, iconY, panelIconSize, panelIconSize)
	setFrame(p.pin, pinX, iconY, panelIconSize, panelIconSize)

	p.divider.Send(sel("setHidden:"), true)
	p.footerDivider.Send(sel("setHidden:"), true)

	footerWidth := width/2 - panelSidePadding
	hintWidth := width/2 - panelSidePadding
	if p.errText != "" {
		footerWidth = width - 2*panelSidePadding
		hintWidth = 0
		p.shortcutButton.Send(sel("setHidden:"), true)
	} else {
		p.shortcutButton.Send(sel("setHidden:"), false)
		hintWidth -= panelShortcutSize + 6
	}
	p.hint.Send(sel("setHidden:"), p.errText != "")
	setFrame(p.footer, panelSidePadding, 9, footerWidth, 16)
	setFrame(p.hint, width/2, 9, hintWidth, 16)
	setFrame(p.shortcutButton, width-panelSidePadding-panelShortcutSize, 7, panelShortcutSize, panelShortcutSize)

	scrollY := panelFooterHeight + 4
	scrollHeight := searchY - 6 - scrollY
	setFrame(p.scroll, 0, scrollY, width, scrollHeight)
	if p.table != 0 {
		p.syncTableFrame()
	}

	layoutEmptyState(p, width, scrollY, scrollHeight)
}

// panelSearchWidth leaves room for the filter, settings and pin buttons.
func panelSearchWidth(width float64) float64 {
	return width - 2*panelSidePadding - 3*panelIconSize - 2*panelIconGap - 10
}

func layoutEmptyState(p *Panel, width, scrollY, scrollHeight float64) {
	if objc.Send[bool](p.empty, sel("isHidden")) {
		return
	}
	const iconSize = 30.0
	const titleHeight = 17.0
	const detailHeight = 15.0
	hasDetail := !objc.Send[bool](p.emptyDetail, sel("isHidden"))
	blockHeight := titleHeight
	blockHeight += iconSize + 8
	if hasDetail {
		blockHeight += 4 + detailHeight
	}
	top := scrollY + scrollHeight/2 + blockHeight/2
	top -= iconSize
	setFrame(p.emptyIcon, (width-iconSize)/2, top, iconSize, iconSize)
	top -= 8
	top -= titleHeight
	setFrame(p.empty, 20, top, width-40, titleHeight)
	if hasDetail {
		top -= 4 + detailHeight
		setFrame(p.emptyDetail, 20, top, width-40, detailHeight)
	}
}

//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

const (
	panelSidePadding  = 14.0
	panelSearchHeight = 34.0
	panelIconSize     = 26.0
	panelIconGap      = 2.0
	panelFooterHeight = 30.0
	panelFooterText   = 16.0
	panelTopPadding   = 14.0
	panelShortcutSize = 26.0
)

func layoutPanel(p *Panel, width, height float64) {
	if p.background != 0 {
		setFrame(p.background, 0, 0, width, height)
	}

	searchY := height - panelTopPadding - panelSearchHeight
	iconY := searchY + (panelSearchHeight-panelIconSize)/2
	pinX := width - panelSidePadding - panelIconSize
	settingsX := pinX - panelIconGap - panelIconSize
	filterX := settingsX - panelIconGap - panelIconSize
	clearX := filterX - panelIconGap - panelIconSize
	setFrame(p.search, panelSidePadding, searchY, panelSearchWidth(width), panelSearchHeight)
	setFrame(p.clearButton, clearX, iconY, panelIconSize, panelIconSize)
	setFrame(p.filterButton, filterX, iconY, panelIconSize, panelIconSize)
	setFrame(p.settingsButton, settingsX, iconY, panelIconSize, panelIconSize)
	setFrame(p.pin, pinX, iconY, panelIconSize, panelIconSize)

	p.divider.Send(sel("setHidden:"), true)
	p.footerDivider.Send(sel("setHidden:"), true)

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

	scrollY := panelFooterHeight + 4
	scrollHeight := searchY - 6 - scrollY
	setFrame(p.scroll, 0, scrollY, width, scrollHeight)
	if p.table != 0 {
		p.syncTableFrame()
	}

	layoutEmptyState(p, width, scrollY, scrollHeight)
}

// panelSearchWidth leaves room for the clear, filter, settings and pin buttons.
func panelSearchWidth(width float64) float64 {
	return width - 2*panelSidePadding - 4*panelIconSize - 3*panelIconGap - 10
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
	if p.emptyKubai != 0 && !objc.Send[bool](p.emptyKubai, sel("isHidden")) {
		top -= emptyKubaiIconSize
		setFrame(p.emptyKubai, (width-emptyKubaiIconSize)/2, top, emptyKubaiIconSize, emptyKubaiIconSize)
	} else {
		top -= iconSize
		setFrame(p.emptyIcon, (width-iconSize)/2, top, iconSize, iconSize)
	}
	top -= 8
	top -= titleHeight
	setFrame(p.empty, 20, top, width-40, titleHeight)
	if hasDetail {
		top -= 4 + detailHeight
		setFrame(p.emptyDetail, 20, top, width-40, detailHeight)
	}
}

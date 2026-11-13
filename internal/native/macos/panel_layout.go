//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

func layoutPanel(p *Panel, width, height float64) {
	const pad = 14.0
	searchHeight := 28.0
	searchY := height - pad - searchHeight
	pinWidth := 26.0
	setFrame(p.search, pad, searchY, width-2*pad-pinWidth-6, searchHeight)
	setFrame(p.pin, width-pad-pinWidth, searchY, pinWidth, searchHeight)

	controlHeight := 24.0
	controlY := searchY - controlHeight - 8
	actionWidth := 26.0
	groupWidth := 106.0
	gap := 8.0
	filterWidth := width - 2*pad - actionWidth - groupWidth - 2*gap
	setFrame(p.filter, pad, controlY, filterWidth, controlHeight)
	setFrame(p.groupsPopup, pad+filterWidth+gap, controlY, groupWidth, controlHeight)
	setFrame(p.groupActions, width-pad-actionWidth, controlY, actionWidth, controlHeight)

	dividerY := controlY - 10
	setFrame(p.divider, 0, dividerY, width, 1)

	footerHeight := 28.0
	footerWidth := width/2 - pad
	hintWidth := width/2 - pad
	if p.errText != "" {
		footerWidth = width - 2*pad
		hintWidth = 0
	}
	p.hint.Send(sel("setHidden:"), p.errText != "")
	setFrame(p.footer, pad, 8, footerWidth, 16)
	setFrame(p.hint, width/2, 8, hintWidth, 16)
	setFrame(p.footerDivider, 0, footerHeight+2, width, 1)
	scrollY := footerHeight + 4
	scrollHeight := dividerY - scrollY - 4
	setFrame(p.scroll, 0, scrollY, width, scrollHeight)
	if p.table != 0 {
		setFrame(p.table, 0, 0, width, scrollHeight)
		if column := p.table.Send(sel("tableColumnWithIdentifier:"), nsString("history")); column != 0 {
			column.Send(sel("setWidth:"), width)
		}
	}
	p.layoutOnboarding(width, height)

	layoutEmptyState(p, width, scrollY, scrollHeight)
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

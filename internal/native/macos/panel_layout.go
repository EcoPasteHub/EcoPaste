//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

const (
	panelSidePadding   = 14.0
	panelActionWidth   = 26.0
	panelGroupWidth    = 106.0
	panelToolbarGap    = 8.0
	panelSearchHeight  = 28.0
	panelControlHeight = 24.0
)

func layoutPanel(p *Panel, width, height float64) {
	searchY := height - panelSidePadding - panelSearchHeight
	pinWidth := 26.0
	setFrame(p.search, panelSidePadding, searchY,
		width-2*panelSidePadding-pinWidth-6, panelSearchHeight)
	setFrame(p.pin, width-panelSidePadding-pinWidth, searchY, pinWidth, panelSearchHeight)

	controlY := searchY - panelControlHeight - 8
	showGroups := len(p.groups) > 0
	p.groupsPopup.Send(sel("setHidden:"), !showGroups)
	filterWidth := panelFilterWidth(width, showGroups)
	setFrame(p.filter, panelSidePadding, controlY, filterWidth, panelControlHeight)
	if showGroups {
		setFrame(p.groupsPopup, panelSidePadding+filterWidth+panelToolbarGap,
			controlY, panelGroupWidth, panelControlHeight)
	}
	setFrame(p.groupActions, width-panelSidePadding-panelActionWidth,
		controlY, panelActionWidth, panelControlHeight)

	dividerY := controlY - 10
	setFrame(p.divider, 0, dividerY, width, 1)

	footerHeight := 28.0
	footerWidth := width/2 - panelSidePadding
	hintWidth := width/2 - panelSidePadding
	if p.errText != "" {
		footerWidth = width - 2*panelSidePadding
		hintWidth = 0
	}
	p.hint.Send(sel("setHidden:"), p.errText != "")
	setFrame(p.footer, panelSidePadding, 8, footerWidth, 16)
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

func panelFilterWidth(width float64, showGroups bool) float64 {
	filterWidth := width - 2*panelSidePadding - panelActionWidth - panelToolbarGap
	if showGroups {
		filterWidth -= panelGroupWidth + panelToolbarGap
	}
	if filterWidth < 0 {
		return 0
	}
	return filterWidth
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

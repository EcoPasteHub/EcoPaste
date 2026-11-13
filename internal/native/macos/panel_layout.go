//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

func layoutPanel(p *Panel, width, height float64) {
	const pad = 12.0
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
	scrollY := footerHeight + 4
	scrollHeight := dividerY - scrollY - 4
	setFrame(p.scroll, 0, scrollY, width, scrollHeight)
	if p.table != 0 {
		setFrame(p.table, 0, 0, width, scrollHeight)
		if column := p.table.Send(sel("tableColumnWithIdentifier:"), nsString("history")); column != 0 {
			column.Send(sel("setWidth:"), width)
		}
	}
	if p.preview != 0 {
		previewHeight := height - (footerHeight + 4) - pad
		setFrame(p.preview, 0, footerHeight+4, width, previewHeight)
		layoutPreview(p, width, previewHeight)
	}
	p.layoutOnboarding(width, height)

	p.empty.Send(sel("sizeToFit"))
	text := objc.Send[CGRect](p.empty, sel("frame"))
	emptyCenter := scrollY + scrollHeight/2
	setFrame(p.empty, (width-text.Size.Width)/2, emptyCenter-text.Size.Height/2, text.Size.Width, text.Size.Height)
}

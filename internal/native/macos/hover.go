//go:build darwin

package macos

import (
	"strings"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

const (
	hoverBarHeight  = 26.0
	hoverButtonW    = 28.0
	hoverBarPadding = 5.0
	maxHoverActions = 6
)

// hoverAction is one button the hover bar shows for a row.
type hoverAction struct {
	Event   string
	Symbol  string
	Tooltip string
}

type hoverRule struct {
	symbol        string
	event         string
	tooltip       i18n.Key
	allowed       func(HistoryRow) bool
	activeSymbol  string
	activeTooltip i18n.Key
	active        func(HistoryRow) bool
}

var hoverRules = map[string]hoverRule{
	"paste": {
		symbol: "arrow.down.doc", event: "activate", tooltip: i18n.MenuPaste,
	},
	"pastePlain": {
		symbol: "textformat", event: "pastePlain", tooltip: i18n.MenuPastePlain,
		allowed: func(row HistoryRow) bool { return row.Kind == "text" },
	},
	"pastePath": {
		symbol: "point.topleft.down.to.point.bottomright.curvepath",
		event:  "pastePath", tooltip: i18n.MenuPastePath,
		allowed: func(row HistoryRow) bool { return row.Kind == "files" },
	},
	"copy": {
		symbol: "doc.on.doc", event: "copy", tooltip: i18n.MenuCopy,
	},
	"copyPlain": {
		symbol: "doc.plaintext", event: "copyPlain", tooltip: i18n.MenuCopyPlain,
		allowed: func(row HistoryRow) bool { return row.Kind == "text" },
	},
	"openLink": {
		symbol: "arrow.up.right.square", event: "open", tooltip: i18n.MenuOpenLink,
		allowed: func(row HistoryRow) bool { return row.SubKind == "url" },
	},
	"sendEmail": {
		symbol: "envelope", event: "open", tooltip: i18n.MenuSendEmail,
		allowed: func(row HistoryRow) bool { return row.SubKind == "email" },
	},
	"reveal": {
		symbol: "folder", event: "reveal", tooltip: i18n.MenuReveal,
		allowed: func(row HistoryRow) bool { return row.Kind == "files" || row.SubKind == "path" },
	},
	"note": {
		symbol: "note.text", event: "editNote", tooltip: i18n.RowNote,
	},
	"star": {
		symbol: "star", event: "toggleFavorite", tooltip: i18n.MenuFavorite,
		activeSymbol: "star.fill", activeTooltip: i18n.MenuUnfavorite,
		active: func(row HistoryRow) bool { return row.Favorite },
	},
	"pinItem": {
		symbol: "pin", event: "togglePinned", tooltip: i18n.MenuPin,
		activeSymbol: "pin.fill", activeTooltip: i18n.MenuUnpin,
		active: func(row HistoryRow) bool { return row.Pinned },
	},
	"delete": {
		symbol: "trash", event: "delete", tooltip: i18n.CommonDelete,
		allowed: func(row HistoryRow) bool { return row.CanDelete },
	},
}

// buildHoverBar creates the floating action bar shown over the hovered
// row. Buttons are created once and activated per hovered row.
func (p *Panel) buildHoverBar() {
	bar := objc.ID(class("NSVisualEffectView")).Send(sel("new"))
	bar.Send(sel("setHidden:"), true)
	bar.Send(sel("setMaterial:"), int64(5))
	bar.Send(sel("setBlendingMode:"), int64(1))
	bar.Send(sel("setState:"), int64(1))
	bar.Send(sel("setWantsLayer:"), true)
	if layer := bar.Send(sel("layer")); layer != 0 {
		layer.Send(sel("setCornerRadius:"), 8.0)
		layer.Send(sel("setMasksToBounds:"), true)
		layer.Send(sel("setBorderWidth:"), 0.5)
	}
	updateHoverBarBorder(bar)
	p.hoverBar = bar
	panelsByControl[bar] = p
	for index := 0; index < maxHoverActions; index++ {
		button := newIconButton()
		button.Send(sel("setBordered:"), false)
		button.Send(sel("setImagePosition:"), int64(1))
		button.Send(sel("setRefusesFirstResponder:"), true)
		button.Send(sel("setTarget:"), searchTarget)
		button.Send(sel("setAction:"), sel("hoverActionPressed:"))
		button.Send(sel("setTag:"), int64(index))
		button.Send(sel("setHidden:"), true)
		bar.Send(sel("addSubview:"), button)
		button.Send(sel("release"))
		p.hoverButtons = append(p.hoverButtons, button)
		panelsByControl[button] = p
	}
	p.view.Send(sel("addSubview:"), bar)
}

// updateHoverAtWindowPoint shows the action bar for the row under a point in
// window coordinates. A mouse-moved event calls it; hover needs no polling.
func (p *Panel) updateHoverAtWindowPoint(windowPoint CGPoint) {
	row, ok := p.hoverRowAtWindowPoint(windowPoint)
	if !ok {
		p.hideHover()
		return
	}
	p.updateHover(row)
}

// refreshHoverFromMouse re-anchors the existing bar after the list scrolls,
// when the cursor has not moved and so no mouse-moved event fires.
func (p *Panel) refreshHoverFromMouse() {
	window := p.view.Send(sel("window"))
	if window == 0 {
		return
	}
	location := objc.Send[CGPoint](objc.ID(class("NSEvent")), sel("mouseLocation"))
	windowPoint := objc.Send[CGPoint](window, sel("convertPointFromScreen:"), location)
	row, ok := p.hoverRowAtWindowPoint(windowPoint)
	if !ok {
		p.hideHover()
		return
	}
	if row == p.hoverRow && p.hoverBarVisible() {
		p.positionHoverBar()
		return
	}
	p.updateHover(row)
}

// hoverRowAtWindowPoint resolves the row under a window point, or false when
// the point is outside the list or the panel is not showing.
func (p *Panel) hoverRowAtWindowPoint(windowPoint CGPoint) (int, bool) {
	window := p.view.Send(sel("window"))
	if window == 0 || !objc.Send[bool](window, sel("isVisible")) || p.previewing {
		return 0, false
	}
	point := objc.Send[CGPoint](p.view, sel("convertPoint:fromView:"), windowPoint, objc.ID(0))
	frame := objc.Send[CGRect](p.scroll, sel("frame"))
	if !pointInRect(point, frame) {
		return 0, false
	}
	local := objc.Send[CGPoint](p.table, sel("convertPoint:fromView:"), windowPoint, objc.ID(0))
	return int(objc.Send[int64](p.table, sel("rowAtPoint:"), local)), true
}

func (p *Panel) hoverBarVisible() bool {
	return p.hoverBar != 0 && !objc.Send[bool](p.hoverBar, sel("isHidden"))
}

// positionHoverBar moves the already-built bar onto its row. Scrolling changes
// the row's rect without changing the row, so only the frame is refreshed.
func (p *Panel) positionHoverBar() {
	if p.hoverRow < 0 || p.hoverBarWidth <= 0 {
		return
	}
	rect := objc.Send[CGRect](p.table, sel("rectOfRow:"), int64(p.hoverRow))
	rowRect := objc.Send[CGRect](p.view, sel("convertRect:fromView:"),
		objc.Send[CGRect](p.table, sel("convertRect:toView:"), rect, objc.ID(0)), objc.ID(0))
	bounds := objc.Send[CGRect](p.view, sel("bounds"))
	setFrame(p.hoverBar, bounds.Size.Width-12-p.hoverBarWidth,
		rowRect.Origin.Y+(rowRect.Size.Height-hoverBarHeight)/2, p.hoverBarWidth, hoverBarHeight)
}

func (p *Panel) updateHover(row int) {
	if row < 0 || row >= len(p.rows) {
		p.hideHover()
		return
	}
	// Moving within the same row keeps the current bar; rebuilding its buttons
	// on every mouse move would repeat work for nothing.
	if row == p.hoverRow && p.hoverBarVisible() {
		return
	}
	specs := hoverActionsFor(p.rows[row], p.actions)
	if len(specs) == 0 {
		p.clearHoverInset()
		p.setHoveredRow(row)
		p.setOriginalRow(row)
		p.hideHoverBar()
		return
	}
	width := 2*hoverBarPadding + hoverButtonW*float64(len(specs))
	p.hoverRow = row
	p.hoverSpecs = specs
	p.hoverBarWidth = width
	p.setHoveredRow(row)
	p.setOriginalRow(row)
	p.positionHoverBar()
	for index, button := range p.hoverButtons {
		hidden := index >= len(specs)
		button.Send(sel("setHidden:"), hidden)
		if hidden {
			continue
		}
		spec := specs[index]
		button.Send(sel("setImage:"), systemSymbol(spec.Symbol))
		button.Send(sel("setToolTip:"), nsString(spec.Tooltip))
		button.Send(sel("setAccessibilityLabel:"), nsString(spec.Tooltip))
		setFrame(button, hoverBarPadding+hoverButtonW*float64(index), 0, hoverButtonW, hoverBarHeight)
	}
	updateHoverBarBorder(p.hoverBar)
	p.hoverBar.Send(sel("setHidden:"), false)
}

// updateHoverBarBorder resolves the separator color in the current
// appearance; a CGColor captured at creation does not follow theme changes.
func updateHoverBarBorder(bar objc.ID) {
	if bar == 0 {
		return
	}
	if layer := bar.Send(sel("layer")); layer != 0 {
		border := objc.ID(classNSColor).Send(sel("separatorColor"))
		layer.Send(sel("setBorderColor:"), border.Send(sel("CGColor")))
	}
}

func (p *Panel) hideHover() {
	p.clearHoverInset()
	p.setHoveredRow(-1)
	p.hideHoverBar()
	p.setOriginalRow(-1)
}

func (p *Panel) hideHoverBar() {
	if p.hoverBar != 0 {
		p.hoverBar.Send(sel("setHidden:"), true)
	}
	p.hoverRow = -1
	p.hoverSpecs = nil
	p.hoverBarWidth = 0
}

func (p *Panel) setOriginalRow(row int) {
	if row < 0 || row >= len(p.rows) || strings.TrimSpace(p.rows[row].Note) == "" || !p.display.ShowOriginal {
		row = -1
	}
	if p.originalRow == row {
		return
	}
	previous := p.originalRow
	p.originalRow = row
	p.refreshHistoryRow(previous)
	p.refreshHistoryRow(row)
}

func (p *Panel) refreshHistoryRow(row int) {
	if row < 0 || row >= len(p.rows) {
		return
	}
	rows := objc.ID(class("NSIndexSet")).Send(sel("indexSetWithIndex:"), int64(row))
	columns := objc.ID(class("NSIndexSet")).Send(sel("indexSetWithIndex:"), int64(0))
	p.table.Send(sel("reloadDataForRowIndexes:columnIndexes:"), rows, columns)
	p.applyHoveredRow(row, row == p.hoveredRow)
}

func (p *Panel) setHoveredRow(row int) {
	if row < 0 || row >= len(p.rows) {
		row = -1
	}
	if p.hoveredRow == row {
		return
	}
	previous := p.hoveredRow
	p.hoveredRow = row
	p.applyHoveredRow(previous, false)
	p.applyHoveredRow(row, true)
}

func (p *Panel) applyHoveredRow(row int, hovered bool) {
	if row < 0 || row >= len(p.rows) {
		return
	}
	inset := 0.0
	if hovered && row == p.hoverRow && p.hoverBarWidth > 0 {
		inset = p.hoverBarWidth + 14
	}
	cell := p.table.Send(sel("viewAtColumn:row:makeIfNecessary:"), int64(0), int64(row), false)
	if cell != 0 {
		setHistoryRowHovered(cell, hovered)
		setRowTrailingInset(cell, inset)
		layoutHistoryRow(cell, objc.Send[CGRect](cell, sel("bounds")).Size)
	}
}

func (p *Panel) clearHoverInset() {
	if p.hoverRow < 0 {
		return
	}
	cell := p.table.Send(sel("viewAtColumn:row:makeIfNecessary:"), int64(0), int64(p.hoverRow), false)
	if cell != 0 {
		setRowTrailingInset(cell, 0)
		layoutHistoryRow(cell, objc.Send[CGRect](cell, sel("bounds")).Size)
	}
}

func (p *Panel) runHoverAction(index int) {
	if p.hoverRow < 0 || p.hoverRow >= len(p.rows) || index < 0 || index >= len(p.hoverSpecs) {
		return
	}
	spec := p.hoverSpecs[index]
	id := p.rows[p.hoverRow].ID
	p.hideHover()
	if spec.Event == "editNote" {
		p.showNotePopover(id)
		return
	}
	p.cx.Emit(spec.Event, id)
}

func hoverActionsFor(row HistoryRow, actions []string) []hoverAction {
	specs := make([]hoverAction, 0, maxHoverActions)
	for _, action := range actions {
		if spec, ok := hoverActionFor(action, row); ok {
			specs = append(specs, spec)
			if len(specs) == maxHoverActions {
				break
			}
		}
	}
	return specs
}

func hoverActionFor(action string, row HistoryRow) (hoverAction, bool) {
	rule, ok := hoverRules[action]
	if !ok {
		return hoverAction{}, false
	}
	if rule.allowed != nil && !rule.allowed(row) {
		return hoverAction{}, false
	}
	spec := hoverAction{Symbol: rule.symbol, Event: rule.event, Tooltip: i18n.T(rule.tooltip)}
	if rule.active != nil && rule.active(row) {
		spec.Symbol = rule.activeSymbol
		spec.Tooltip = i18n.T(rule.activeTooltip)
	}
	return spec, true
}

func pointInRect(point CGPoint, rect CGRect) bool {
	return point.X >= rect.Origin.X && point.X <= rect.Origin.X+rect.Size.Width &&
		point.Y >= rect.Origin.Y && point.Y <= rect.Origin.Y+rect.Size.Height
}

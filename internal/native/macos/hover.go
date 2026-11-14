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

var hoverSymbols = map[string]string{
	"paste":      "arrow.down.doc",
	"pastePlain": "textformat",
	"pastePath":  "point.topleft.down.to.point.bottomright.curvepath",
	"copy":       "doc.on.doc",
	"copyPlain":  "doc.plaintext",
	"openLink":   "arrow.up.right.square",
	"sendEmail":  "envelope",
	"reveal":     "folder",
	"note":       "note.text",
	"star":       "star",
	"pinItem":    "pin",
	"delete":     "trash",
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
		border := objc.ID(classNSColor).Send(sel("separatorColor"))
		layer.Send(sel("setBorderColor:"), border.Send(sel("CGColor")))
	}
	p.hoverBar = bar
	panelsByControl[bar] = p
	for index := 0; index < maxHoverActions; index++ {
		button := objc.ID(class("NSButton")).Send(sel("alloc")).
			Send(sel("initWithFrame:"), CGRect{})
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

// pollHover keeps the action bar under the pointer without installing a
// tracking area, which purego cannot route reliably for this view.
func (p *Panel) pollHover() {
	window := p.view.Send(sel("window"))
	if window == 0 || !objc.Send[bool](window, sel("isVisible")) || p.previewing || p.onboardingActive {
		p.hideHover()
		return
	}
	location := objc.Send[CGPoint](objc.ID(class("NSEvent")), sel("mouseLocation"))
	screen := objc.Send[CGPoint](window, sel("convertPointFromScreen:"), location)
	point := objc.Send[CGPoint](p.view, sel("convertPoint:fromView:"), screen, objc.ID(0))
	frame := objc.Send[CGRect](p.scroll, sel("frame"))
	local := objc.Send[CGPoint](p.table, sel("convertPoint:fromView:"), screen, objc.ID(0))
	row := int(objc.Send[int64](p.table, sel("rowAtPoint:"), local))
	if !pointInRect(point, frame) {
		p.hideHover()
		return
	}
	p.updateHover(row)
}
func (p *Panel) updateHover(row int) {
	if row < 0 || row >= len(p.rows) {
		p.hideHover()
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
	rect := objc.Send[CGRect](p.table, sel("rectOfRow:"), int64(row))
	rowRect := objc.Send[CGRect](p.view, sel("convertRect:fromView:"),
		objc.Send[CGRect](p.table, sel("convertRect:toView:"), rect, objc.ID(0)), objc.ID(0))
	bounds := objc.Send[CGRect](p.view, sel("bounds"))
	setFrame(p.hoverBar, bounds.Size.Width-12-width,
		rowRect.Origin.Y+(rowRect.Size.Height-hoverBarHeight)/2, width, hoverBarHeight)
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
	p.hoverBar.Send(sel("setHidden:"), false)
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
	rowView := p.table.Send(sel("rowViewAtRow:makeIfNecessary:"), int64(row), false)
	if rowView != 0 {
		setHistoryRowHovered(rowView, hovered)
	}
	inset := 0.0
	if hovered && row == p.hoverRow && p.hoverBarWidth > 0 {
		inset = p.hoverBarWidth + 14
	}
	cell := p.table.Send(sel("viewAtColumn:row:makeIfNecessary:"), int64(0), int64(row), false)
	if cell != 0 {
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
	symbol, ok := hoverSymbols[action]
	if !ok {
		return hoverAction{}, false
	}
	spec := hoverAction{Symbol: symbol}
	switch action {
	case "paste":
		spec.Event, spec.Tooltip = "activate", i18n.T(i18n.MenuPaste)
	case "pastePlain":
		if row.Kind != "text" {
			return hoverAction{}, false
		}
		spec.Event, spec.Tooltip = "pastePlain", i18n.T(i18n.MenuPastePlain)
	case "pastePath":
		if row.Kind != "files" {
			return hoverAction{}, false
		}
		spec.Event, spec.Tooltip = "pastePath", i18n.T(i18n.MenuPastePath)
	case "copy":
		spec.Event, spec.Tooltip = "copy", i18n.T(i18n.MenuCopy)
	case "copyPlain":
		if row.Kind != "text" {
			return hoverAction{}, false
		}
		spec.Event, spec.Tooltip = "copyPlain", i18n.T(i18n.MenuCopyPlain)
	case "openLink":
		if row.SubKind != "url" {
			return hoverAction{}, false
		}
		spec.Event, spec.Tooltip = "open", i18n.T(i18n.MenuOpenLink)
	case "sendEmail":
		if row.SubKind != "email" {
			return hoverAction{}, false
		}
		spec.Event, spec.Tooltip = "open", i18n.T(i18n.MenuSendEmail)
	case "reveal":
		if row.Kind != "files" && row.SubKind != "path" {
			return hoverAction{}, false
		}
		spec.Event, spec.Tooltip = "reveal", i18n.T(i18n.MenuReveal)
	case "note":
		spec.Event, spec.Tooltip = "editNote", i18n.T(i18n.RowNote)
	case "star":
		spec.Event, spec.Tooltip = "toggleFavorite", i18n.T(i18n.MenuFavorite)
		if row.Favorite {
			spec.Symbol, spec.Tooltip = "star.fill", i18n.T(i18n.MenuUnfavorite)
		}
	case "pinItem":
		spec.Event, spec.Tooltip = "togglePinned", i18n.T(i18n.MenuPin)
		if row.Pinned {
			spec.Symbol, spec.Tooltip = "pin.fill", i18n.T(i18n.MenuUnpin)
		}
	case "delete":
		if !row.CanDelete {
			return hoverAction{}, false
		}
		spec.Event, spec.Tooltip = "delete", i18n.T(i18n.CommonDelete)
	default:
		return hoverAction{}, false
	}
	return spec, true
}

func pointInRect(point CGPoint, rect CGRect) bool {
	return point.X >= rect.Origin.X && point.X <= rect.Origin.X+rect.Size.Width &&
		point.Y >= rect.Origin.Y && point.Y <= rect.Origin.Y+rect.Size.Height
}

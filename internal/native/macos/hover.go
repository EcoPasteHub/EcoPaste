//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

const (
	hoverBarHeight  = 24.0
	hoverButtonW    = 26.0
	hoverBarPadding = 4.0
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
		layer.Send(sel("setCornerRadius:"), 6.0)
		layer.Send(sel("setMasksToBounds:"), true)
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
		p.hideHover()
		return
	}
	p.hoverRow = row
	p.hoverSpecs = specs
	width := 2*hoverBarPadding + hoverButtonW*float64(len(specs))
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
	if p.hoverBar != 0 {
		p.hoverBar.Send(sel("setHidden:"), true)
	}
	p.hoverRow = -1
	p.hoverSpecs = nil
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
		spec.Event, spec.Tooltip = "activate", "粘贴"
	case "pastePlain":
		if row.Kind != "text" {
			return hoverAction{}, false
		}
		spec.Event, spec.Tooltip = "pastePlain", "纯文本粘贴"
	case "pastePath":
		if row.Kind != "files" {
			return hoverAction{}, false
		}
		spec.Event, spec.Tooltip = "pastePath", "粘贴为路径"
	case "copy":
		spec.Event, spec.Tooltip = "copy", "复制"
	case "copyPlain":
		if row.Kind != "text" {
			return hoverAction{}, false
		}
		spec.Event, spec.Tooltip = "copyPlain", "复制为纯文本"
	case "openLink":
		if row.SubKind != "url" {
			return hoverAction{}, false
		}
		spec.Event, spec.Tooltip = "open", "打开链接"
	case "sendEmail":
		if row.SubKind != "email" {
			return hoverAction{}, false
		}
		spec.Event, spec.Tooltip = "open", "发送邮件"
	case "reveal":
		if row.Kind != "files" && row.SubKind != "path" {
			return hoverAction{}, false
		}
		spec.Event, spec.Tooltip = "reveal", "在 Finder 中显示"
	case "note":
		spec.Event, spec.Tooltip = "editNote", "备注"
	case "star":
		spec.Event, spec.Tooltip = "toggleFavorite", "收藏"
		if row.Favorite {
			spec.Symbol, spec.Tooltip = "star.fill", "取消收藏"
		}
	case "pinItem":
		spec.Event, spec.Tooltip = "togglePinned", "置顶"
		if row.Pinned {
			spec.Symbol, spec.Tooltip = "pin.fill", "取消置顶"
		}
	case "delete":
		if !row.CanDelete {
			return hoverAction{}, false
		}
		spec.Event, spec.Tooltip = "delete", "删除"
	default:
		return hoverAction{}, false
	}
	return spec, true
}

func pointInRect(point CGPoint, rect CGRect) bool {
	return point.X >= rect.Origin.X && point.X <= rect.Origin.X+rect.Size.Width &&
		point.Y >= rect.Origin.Y && point.Y <= rect.Origin.Y+rect.Size.Height
}

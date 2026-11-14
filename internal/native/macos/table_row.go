//go:build darwin

package macos

import (
	"strings"
	"sync"
	"unsafe"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

const (
	historyRowInsetX   = 10.0
	historyRowInsetY   = 2.0
	historyRowRadius   = 10.0
	historyRowTextLeft = 15.0
	historyRowGap      = 11.0
)

var (
	historyRowClass     objc.Class
	historyRowViewClass objc.Class

	associatedRowHoveredKey  byte
	associatedRowSelectedKey byte

	historyRowViewsOnce sync.Once
	historyRowViewsErr  error
)

func registerHistoryRowViews() error {
	historyRowViewsOnce.Do(func() {
		historyRowViewsErr = registerHistoryRowViewsClass()
	})
	return historyRowViewsErr
}

func registerHistoryRowViewsClass() error {
	var err error
	historyRowClass, err = objc.RegisterClass("EcoPasteHistoryRow", class("NSTableCellView"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("setFrameSize:"), Fn: func(self objc.ID, _ objc.SEL, size CGSize) {
				sendSuper[objc.ID](self, historyRowClass, sel("setFrameSize:"), size)
				layoutHistoryRow(self, size)
			}},
			{Cmd: sel("viewDidChangeEffectiveAppearance"), Fn: func(self objc.ID, _ objc.SEL) {
				sendSuper[objc.ID](self, historyRowClass, sel("viewDidChangeEffectiveAppearance"))
				self.Send(sel("setNeedsDisplay:"), true)
			}},
			{Cmd: sel("drawRect:"), Fn: drawHistoryRowCard},
		})
	if err != nil {
		return err
	}
	historyRowViewClass, err = objc.RegisterClass("EcoPasteHistoryRowView", class("NSTableRowView"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("drawBackgroundInRect:"), Fn: func(objc.ID, objc.SEL, CGRect) {}},
			{Cmd: sel("drawSelectionInRect:"), Fn: func(objc.ID, objc.SEL, CGRect) {}},
			{Cmd: sel("drawSeparatorInRect:"), Fn: func(objc.ID, objc.SEL, CGRect) {}},
		})
	return err
}

func newHistoryRow() objc.ID {
	cell := objc.ID(historyRowClass).Send(sel("new"))
	cell.Send(sel("setIdentifier:"), nsString(rowIdentifier))

	icon := objc.ID(class("NSImageView")).Send(sel("new"))
	icon.Send(sel("setTag:"), int64(rowIconTag))
	icon.Send(sel("setImageScaling:"), int64(3))
	icon.Send(sel("setAccessibilityElement:"), false)
	cell.Send(sel("addSubview:"), icon)
	icon.Send(sel("release"))

	badge := objc.ID(class("NSImageView")).Send(sel("new"))
	badge.Send(sel("setTag:"), int64(rowBadgeTag))
	badge.Send(sel("setImageScaling:"), int64(3))
	badge.Send(sel("setAccessibilityElement:"), false)
	cell.Send(sel("addSubview:"), badge)
	badge.Send(sel("release"))

	title := objc.ID(classNSTextField).Send(sel("wrappingLabelWithString:"), nsString(""))
	title.Send(sel("setTag:"), int64(rowTitleTag))
	title.Send(sel("setFont:"), objc.ID(classNSFont).Send(sel("systemFontOfSize:weight:"), historyRowTitleSize, 0.2))
	title.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("labelColor")))
	title.Send(sel("setLineBreakMode:"), int64(4))
	title.Send(sel("setMaximumNumberOfLines:"), int64(1))
	title.Send(sel("setUsesSingleLineMode:"), false)
	title.Send(sel("setSelectable:"), false)
	if titleCell := title.Send(sel("cell")); titleCell != 0 {
		titleCell.Send(sel("setWraps:"), true)
		titleCell.Send(sel("setTruncatesLastVisibleLine:"), true)
	}
	cell.Send(sel("addSubview:"), title)

	preview := objc.ID(classNSTextField).Send(sel("wrappingLabelWithString:"), nsString(""))
	preview.Send(sel("setTag:"), int64(rowPreviewTag))
	preview.Send(sel("setFont:"), historyRowFont(historyRowPreviewSize))
	preview.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("labelColor")))
	preview.Send(sel("setLineBreakMode:"), int64(0))
	preview.Send(sel("setMaximumNumberOfLines:"), int64(1))
	preview.Send(sel("setSelectable:"), false)
	cell.Send(sel("addSubview:"), preview)

	detail := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	detail.Send(sel("setTag:"), int64(rowDetailTag))
	detail.Send(sel("setFont:"), historyRowFont(historyRowDetailSize))
	detail.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("tertiaryLabelColor")))
	detail.Send(sel("setLineBreakMode:"), int64(4))
	detail.Send(sel("setMaximumNumberOfLines:"), int64(1))
	detail.Send(sel("setSelectable:"), false)
	cell.Send(sel("addSubview:"), detail)

	return cell.Send(sel("autorelease"))
}

func fillHistoryRow(cell objc.ID, row HistoryRow, display PanelDisplay, revealOriginal, selected bool) {
	icon := cell.Send(sel("viewWithTag:"), int64(rowIconTag))
	badge := cell.Send(sel("viewWithTag:"), int64(rowBadgeTag))
	title := cell.Send(sel("viewWithTag:"), int64(rowTitleTag))
	preview := cell.Send(sel("viewWithTag:"), int64(rowPreviewTag))
	detail := cell.Send(sel("viewWithTag:"), int64(rowDetailTag))
	titleText, previewText, detailText := historyRowText(row, display, revealOriginal)
	title.Send(sel("setStringValue:"), nsString(titleText))
	preview.Send(sel("setStringValue:"), nsString(previewText))
	preview.Send(sel("setHidden:"), previewText == "")
	detail.Send(sel("setStringValue:"), nsString(detailText))
	setHistoryRowTextColors(cell, selected)
	setHistoryRowSelected(cell, selected)
	weight := 0.2
	if row.Kind == "text" {
		weight = 0
	}
	title.Send(sel("setFont:"), historyRowFontWeight(historyRowTitleSize, weight))
	lines := int64(1)
	breakMode := int64(4)
	maxLines := int64(1)
	if row.Kind == "text" && previewText == "" && display.TextMaxLines > 1 {
		lines = int64(display.TextMaxLines)
		maxLines = lines
		breakMode = 0
	}
	title.Send(sel("setMaximumNumberOfLines:"), maxLines)
	title.Send(sel("setLineBreakMode:"), breakMode)
	previewLines := 0
	if previewText != "" {
		previewLines = min(max(display.TextMaxLines-1, 1), 2)
		preview.Send(sel("setMaximumNumberOfLines:"), int64(previewLines))
	}
	storeRowMetrics(cell, iconSizeForRow(row, display), int(lines), previewLines)

	image := objc.ID(0)
	if row.ImagePath != "" {
		image = loadImage(row.ImagePath)
	}
	sourceIcon := objc.ID(0)
	if row.SourceIconPath != "" {
		sourceIcon = loadImage(row.SourceIconPath)
	}
	badge.Send(sel("setHidden:"), true)
	switch {
	case image != 0:
		icon.Send(sel("setImage:"), image)
		icon.Send(sel("setContentTintColor:"), objc.ID(0))
		if sourceIcon != 0 {
			badge.Send(sel("setImage:"), sourceIcon)
			badge.Send(sel("setContentTintColor:"), objc.ID(0))
			badge.Send(sel("setHidden:"), false)
		}
	case sourceIcon != 0:
		icon.Send(sel("setImage:"), sourceIcon)
		icon.Send(sel("setContentTintColor:"), objc.ID(0))
		badge.Send(sel("setImage:"), symbolImage(row.Kind))
		badge.Send(sel("setContentTintColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
		badge.Send(sel("setHidden:"), false)
	default:
		icon.Send(sel("setImage:"), symbolImage(row.Kind))
		icon.Send(sel("setContentTintColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	}
}

// setHistoryRowTextColors switches a row between normal and selected
// text, because the selected card is drawn with the accent color.
func setHistoryRowTextColors(cell objc.ID, selected bool) {
	title := cell.Send(sel("viewWithTag:"), int64(rowTitleTag))
	preview := cell.Send(sel("viewWithTag:"), int64(rowPreviewTag))
	detail := cell.Send(sel("viewWithTag:"), int64(rowDetailTag))
	titleColor := objc.ID(classNSColor).Send(sel("labelColor"))
	previewColor := objc.ID(classNSColor).Send(sel("secondaryLabelColor"))
	detailColor := objc.ID(classNSColor).Send(sel("secondaryLabelColor"))
	if !selected {
		detailColor = objc.ID(classNSColor).Send(sel("tertiaryLabelColor"))
	}
	if selected {
		titleColor = objc.ID(classNSColor).Send(sel("alternateSelectedControlTextColor"))
		previewColor = titleColor.Send(sel("colorWithAlphaComponent:"), 0.9)
		detailColor = titleColor.Send(sel("colorWithAlphaComponent:"), 0.82)
	}
	title.Send(sel("setTextColor:"), titleColor)
	preview.Send(sel("setTextColor:"), previewColor)
	detail.Send(sel("setTextColor:"), detailColor)
}

// historyRowText keeps a note as the card's default content while hovering
// temporarily reveals the copied content, matching the legacy note card.
func historyRowText(row HistoryRow, display PanelDisplay, revealOriginal bool) (string, string, string) {
	note := strings.TrimSpace(row.Note)
	if note == "" {
		return row.Title, row.Preview, row.Detail
	}
	if !display.ShowOriginal {
		return row.Title, "", appendHistoryDetail(row.Detail, note)
	}
	detail := appendHistoryDetail(row.Detail, i18n.T(i18n.NoteDetail))
	if revealOriginal {
		return row.Title, row.Preview, detail
	}
	return i18n.T(i18n.NoteDetailPrefix) + note, "", detail
}

func appendHistoryDetail(detail, value string) string {
	if strings.TrimSpace(detail) == "" {
		return value
	}
	return detail + " · " + value
}

func iconSizeForRow(row HistoryRow, display PanelDisplay) float64 {
	if row.Kind != "image" {
		return historyRowIconSize
	}
	size := display.ImageMaxHeight
	if size < 20 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return float64(size)
}

const (
	historyRowIconSize    = 30.0
	historyRowTitleSize   = 13.0
	historyRowPreviewSize = 12.0
	historyRowDetailSize  = 11.0
	historyRowLineHeight  = 16.0
)

func historyRowFont(size float64) objc.ID {
	return historyRowFontWeight(size, 0)
}

func historyRowFontWeight(size, weight float64) objc.ID {
	return objc.ID(classNSFont).Send(sel("systemFontOfSize:weight:"), size, weight)
}

const associationRetainNonatomic = 1

// storeRowMetrics keeps the visual metrics of a reused cell view on the
// view itself. NSTableCellView has no writable tag, and the layout pass
// only receives the cell and its size.
func storeRowMetrics(cell objc.ID, iconSize float64, lines, previewLines int) {
	packed := int64(iconSize)*100 + int64(lines)*10 + int64(previewLines)
	number := objc.ID(class("NSNumber")).Send(sel("numberWithInteger:"), packed)
	objcSetAssociatedObject(uintptr(cell), uintptr(unsafe.Pointer(&associatedRowMetricsKey)),
		uintptr(number), associationRetainNonatomic)
}

func rowMetrics(cell objc.ID) (iconSize float64, lines, previewLines int) {
	value := objc.ID(objcGetAssociatedObject(uintptr(cell), uintptr(unsafe.Pointer(&associatedRowMetricsKey))))
	if value == 0 {
		return historyRowIconSize, 1, 0
	}
	packed := objc.Send[int64](value, sel("integerValue"))
	iconSize = float64(packed / 100)
	lines = int(packed / 10 % 10)
	previewLines = int(packed % 10)
	if iconSize <= 0 {
		iconSize = historyRowIconSize
	}
	if lines < 1 {
		lines = 1
	}
	return iconSize, lines, previewLines
}

// tableRowHeight turns the configured density into one row height. The
// table uses a uniform height because AppKit's per-row height callback
// cannot return a floating-point value through the purego bridge.
func tableRowHeight(display PanelDisplay) float64 {
	height := rowHeight
	if value := historyTextRowHeight(display.TextMaxLines); value > height {
		height = value
	}
	if value := historyImageRowHeight(display); value > height {
		height = value
	}
	return height
}

func historyTextRowHeight(lines int) float64 {
	height := rowHeight
	if lines > 1 {
		if value := 41 + historyRowLineHeight*float64(lines); value > height {
			height = value
		}
	}
	return height
}

func historyImageRowHeight(display PanelDisplay) float64 {
	height := rowHeight
	if value := iconSizeForRow(HistoryRow{Kind: "image"}, display) + 24; value > height {
		height = value
	}
	return height
}

func layoutHistoryRow(cell objc.ID, size CGSize) {
	const badgeSize = 16.0
	if visible := objc.Send[CGRect](cell, sel("visibleRect")); visible.Size.Width > 0 {
		if available := visible.Origin.X + visible.Size.Width; available < size.Width {
			size.Width = available
		}
	}
	left := historyRowTextLeft
	iconSize, lines, previewLines := rowMetrics(cell)
	textX := left + iconSize + historyRowGap
	detailRight := size.Width - textX - historyRowTextInset
	titleRight := detailRight - rowTrailingInset(cell)
	if detailRight < 48 {
		detailRight = 48
	}
	if titleRight < 48 {
		titleRight = 48
	}
	icon := cell.Send(sel("viewWithTag:"), int64(rowIconTag))
	badge := cell.Send(sel("viewWithTag:"), int64(rowBadgeTag))
	title := cell.Send(sel("viewWithTag:"), int64(rowTitleTag))
	preview := cell.Send(sel("viewWithTag:"), int64(rowPreviewTag))
	detail := cell.Send(sel("viewWithTag:"), int64(rowDetailTag))
	iconY := (size.Height - iconSize) / 2
	setFrame(icon, left, iconY, iconSize, iconSize)
	setFrame(badge, left+iconSize-badgeSize+2, iconY+2, badgeSize, badgeSize)
	titleHeight := historyRowLineHeight * float64(lines)
	if titleHeight < 17 {
		titleHeight = 17
	}
	hasPreview := !objc.Send[bool](preview, sel("isHidden"))
	blockHeight := titleHeight + 3 + 14
	if hasPreview {
		blockHeight += 2 + historyRowLineHeight*float64(previewLines)
	}
	detailY := (size.Height - blockHeight) / 2
	if detailY < 6 {
		detailY = 6
	}
	setFrame(detail, textX, detailY, detailRight, 15)
	y := detailY + 15 + 2
	if hasPreview {
		previewHeight := historyRowLineHeight * float64(previewLines)
		setFrame(preview, textX, y, titleRight, previewHeight)
		y += previewHeight + 3
	}
	setFrame(title, textX, y, titleRight, titleHeight)
}

const historyRowTextInset = 16.0

var associatedRowTrailingKey byte

func setRowTrailingInset(cell objc.ID, inset float64) {
	if cell == 0 {
		return
	}
	number := objc.ID(class("NSNumber")).Send(sel("numberWithDouble:"), inset)
	objcSetAssociatedObject(uintptr(cell), uintptr(unsafe.Pointer(&associatedRowTrailingKey)),
		uintptr(number), associationRetainNonatomic)
}

func rowTrailingInset(cell objc.ID) float64 {
	value := objc.ID(objcGetAssociatedObject(uintptr(cell), uintptr(unsafe.Pointer(&associatedRowTrailingKey))))
	if value == 0 {
		return 0
	}
	return objc.Send[float64](value, sel("doubleValue"))
}

// drawHistoryRowCard paints the single translucent layer each record
// gets. The window material stays visible through it, so the list reads
// as cards on glass instead of another panel inside the window.
func drawHistoryRowCard(self objc.ID, _ objc.SEL, _ CGRect) {
	selected := historyRowSelected(self)
	hovered := historyRowHoveredVal(self)
	rect := visibleRowWidth(self)
	if selected {
		fillHistoryRowCard(rect, historyRowSelectionColor(), historyRowRadius)
		return
	}
	fillHistoryRowCard(rect, historyRowCardColor(self, hovered), historyRowRadius)
}

func setHistoryRowSelected(cell objc.ID, selected bool) {
	value := objc.ID(class("NSNumber")).Send(sel("numberWithBool:"), selected)
	objcSetAssociatedObject(uintptr(cell), uintptr(unsafe.Pointer(&associatedRowSelectedKey)),
		uintptr(value), associationRetainNonatomic)
	cell.Send(sel("setNeedsDisplay:"), true)
}

func historyRowSelected(cell objc.ID) bool {
	value := objc.ID(objcGetAssociatedObject(uintptr(cell), uintptr(unsafe.Pointer(&associatedRowSelectedKey))))
	return value != 0 && objc.Send[bool](value, sel("boolValue"))
}

func historyRowHoveredVal(cell objc.ID) bool {
	value := objc.ID(objcGetAssociatedObject(uintptr(cell), uintptr(unsafe.Pointer(&associatedRowHoveredKey))))
	return value != 0 && objc.Send[bool](value, sel("boolValue"))
}

// historyRowCardColor is a barely-there fill in both appearances: a
// light lift over the light glass, a faint white veil over the dark one.
func historyRowCardColor(rowView objc.ID, hovered bool) objc.ID {
	white := objc.ID(classNSColor).Send(sel("whiteColor"))
	if appearanceIsDark(rowView) {
		if hovered {
			return white.Send(sel("colorWithAlphaComponent:"), 0.12)
		}
		return white.Send(sel("colorWithAlphaComponent:"), 0.06)
	}
	if hovered {
		return white.Send(sel("colorWithAlphaComponent:"), 0.44)
	}
	return white.Send(sel("colorWithAlphaComponent:"), 0.24)
}

// historyRowSelectionColor uses the system's emphasized selection fill
// so the selected card stays distinct in both appearances.
func historyRowSelectionColor() objc.ID {
	return objc.ID(classNSColor).Send(sel("selectedContentBackgroundColor"))
}

func fillHistoryRowCard(rect CGRect, color objc.ID, radius float64) {
	inset := historyRowSelectionRect(rect)
	path := objc.ID(class("NSBezierPath")).Send(sel("bezierPathWithRoundedRect:xRadius:yRadius:"),
		inset, radius, radius)
	color.Send(sel("setFill"))
	path.Send(sel("fill"))
}

func historyRowSelectionRect(rect CGRect) CGRect {
	return CGRect{
		Origin: CGPoint{X: rect.Origin.X + historyRowInsetX, Y: rect.Origin.Y + historyRowInsetY},
		Size: CGSize{
			Width:  rect.Size.Width - 2*historyRowInsetX,
			Height: rect.Size.Height - 2*historyRowInsetY,
		},
	}
}

// visibleRowWidth trims the cell's frame to the part the table actually
// shows. The cell can be wider than the clip view, and drawing its full
// width would leave no right margin beside the scroll edge.
func visibleRowWidth(view objc.ID) CGRect {
	rect := bounds(view)
	visible := objc.Send[CGRect](view, sel("visibleRect"))
	if visible.Size.Width <= 0 {
		return rect
	}
	left := rect.Origin.X
	right := min(rect.Origin.X+rect.Size.Width, visible.Origin.X+visible.Size.Width)
	if right <= left {
		return rect
	}
	rect.Origin.X = left
	rect.Size.Width = right - left
	return rect
}

func setHistoryRowHovered(rowView objc.ID, hovered bool) {
	value := objc.ID(class("NSNumber")).Send(sel("numberWithBool:"), hovered)
	objcSetAssociatedObject(uintptr(rowView), uintptr(unsafe.Pointer(&associatedRowHoveredKey)),
		uintptr(value), associationRetainNonatomic)
	rowView.Send(sel("setNeedsDisplay:"), true)
}

func historyRowHovered(rowView objc.ID) bool {
	value := objc.ID(objcGetAssociatedObject(uintptr(rowView), uintptr(unsafe.Pointer(&associatedRowHoveredKey))))
	return value != 0 && objc.Send[bool](value, sel("boolValue"))
}

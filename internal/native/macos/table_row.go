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
	historyRowInsetX = 8.0
	historyRowInsetY = 3.0
)

var (
	historyRowClass     objc.Class
	historyRowViewClass objc.Class

	associatedRowHoveredKey byte

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
		[]objc.MethodDef{{Cmd: sel("setFrameSize:"), Fn: func(self objc.ID, _ objc.SEL, size CGSize) {
			sendSuper[objc.ID](self, historyRowClass, sel("setFrameSize:"), size)
			layoutHistoryRow(self, size)
		}}})
	if err != nil {
		return err
	}
	historyRowViewClass, err = objc.RegisterClass("EcoPasteHistoryRowView", class("NSTableRowView"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("drawBackgroundInRect:"), Fn: drawHistoryRowBackground},
			{Cmd: sel("drawSelectionInRect:"), Fn: drawHistoryRowSelection},
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
	title.Send(sel("setFont:"), objc.ID(classNSFont).Send(sel("systemFontOfSize:weight:"), 13.0, 0.3))
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

	detail := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	detail.Send(sel("setTag:"), int64(rowDetailTag))
	detail.Send(sel("setFont:"), systemFont(11))
	detail.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	detail.Send(sel("setLineBreakMode:"), int64(4))
	detail.Send(sel("setMaximumNumberOfLines:"), int64(1))
	detail.Send(sel("setSelectable:"), false)
	cell.Send(sel("addSubview:"), detail)

	return cell.Send(sel("autorelease"))
}

func fillHistoryRow(cell objc.ID, row HistoryRow, display PanelDisplay, revealOriginal bool) {
	icon := cell.Send(sel("viewWithTag:"), int64(rowIconTag))
	badge := cell.Send(sel("viewWithTag:"), int64(rowBadgeTag))
	title := cell.Send(sel("viewWithTag:"), int64(rowTitleTag))
	detail := cell.Send(sel("viewWithTag:"), int64(rowDetailTag))
	titleText, detailText := historyRowText(row, display, revealOriginal)
	title.Send(sel("setStringValue:"), nsString(titleText))
	detail.Send(sel("setStringValue:"), nsString(detailText))
	weight := 0.3
	if row.Kind == "text" {
		weight = 0
	}
	title.Send(sel("setFont:"), objc.ID(classNSFont).Send(sel("systemFontOfSize:weight:"), 13.0, weight))
	lines := int64(1)
	breakMode := int64(4)
	maxLines := int64(1)
	if row.Kind == "text" {
		if display.TextMaxLines > 1 {
			lines = int64(display.TextMaxLines)
			maxLines = lines
			breakMode = 0
		}
	}
	title.Send(sel("setMaximumNumberOfLines:"), maxLines)
	title.Send(sel("setLineBreakMode:"), breakMode)
	storeRowMetrics(cell, iconSizeForRow(row, display), int(lines))

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

// historyRowText keeps a note as the card's default content while hovering
// temporarily reveals the copied content, matching the legacy note card.
func historyRowText(row HistoryRow, display PanelDisplay, revealOriginal bool) (string, string) {
	note := strings.TrimSpace(row.Note)
	if note == "" {
		return row.Title, row.Detail
	}
	if !display.ShowOriginal {
		return row.Title, appendHistoryDetail(row.Detail, note)
	}
	detail := appendHistoryDetail(row.Detail, i18n.T(i18n.NoteDetail))
	if revealOriginal {
		return row.Title, detail
	}
	return i18n.T(i18n.NoteDetailPrefix) + note, detail
}

func appendHistoryDetail(detail, value string) string {
	if strings.TrimSpace(detail) == "" {
		return value
	}
	return detail + " · " + value
}

func iconSizeForRow(row HistoryRow, display PanelDisplay) float64 {
	if row.Kind != "image" {
		return 32
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

const associationRetainNonatomic = 1

// storeRowMetrics keeps the visual metrics of a reused cell view on the
// view itself. NSTableCellView has no writable tag, and the layout pass
// only receives the cell and its size.
func storeRowMetrics(cell objc.ID, iconSize float64, lines int) {
	packed := int64(iconSize)*10 + int64(lines)
	number := objc.ID(class("NSNumber")).Send(sel("numberWithInteger:"), packed)
	objcSetAssociatedObject(uintptr(cell), uintptr(unsafe.Pointer(&associatedRowMetricsKey)),
		uintptr(number), associationRetainNonatomic)
}

func rowMetrics(cell objc.ID) (iconSize float64, lines int) {
	value := objc.ID(objcGetAssociatedObject(uintptr(cell), uintptr(unsafe.Pointer(&associatedRowMetricsKey))))
	if value == 0 {
		return 32, 1
	}
	packed := objc.Send[int64](value, sel("integerValue"))
	iconSize = float64(packed / 10)
	lines = int(packed % 10)
	if iconSize <= 0 {
		iconSize = 32
	}
	if lines < 1 {
		lines = 1
	}
	return iconSize, lines
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
		if value := 41 + 16*float64(lines); value > height {
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
	const left = 14.0
	const badgeSize = 16.0
	iconSize, lines := rowMetrics(cell)
	textX := left + iconSize + 13
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
	detail := cell.Send(sel("viewWithTag:"), int64(rowDetailTag))
	iconY := (size.Height - iconSize) / 2
	setFrame(icon, left, iconY, iconSize, iconSize)
	setFrame(badge, left+iconSize-badgeSize+2, iconY+2, badgeSize, badgeSize)
	titleHeight := 16 * float64(lines)
	if titleHeight < 17 {
		titleHeight = 17
	}
	blockHeight := titleHeight + 4 + 15
	detailY := (size.Height - blockHeight) / 2
	if detailY < 8 {
		detailY = 8
	}
	setFrame(detail, textX, detailY, detailRight, 15)
	setFrame(title, textX, detailY+15+4, titleRight, titleHeight)
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

func drawHistoryRowBackground(self objc.ID, _ objc.SEL, rect CGRect) {
	if !historyRowHovered(self) || objc.Send[bool](self, sel("isSelected")) {
		return
	}
	color := objc.ID(classNSColor).Send(sel("unemphasizedSelectedContentBackgroundColor")).
		Send(sel("colorWithAlphaComponent:"), 0.55)
	fillHistoryRowSelection(rect, color)
}

func drawHistoryRowSelection(self objc.ID, _ objc.SEL, rect CGRect) {
	color := objc.ID(classNSColor).Send(sel("unemphasizedSelectedContentBackgroundColor"))
	fillHistoryRowSelection(rect, color)
}

func fillHistoryRowSelection(rect CGRect, color objc.ID) {
	inset := historyRowSelectionRect(rect)
	path := objc.ID(class("NSBezierPath")).Send(sel("bezierPathWithRoundedRect:xRadius:yRadius:"),
		inset, 7.0, 7.0)
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

//go:build darwin

package macos

import (
	"math"
	"strings"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

// rowContentLeft is the x where the content column starts.
func rowContentLeft() float64 {
	return historyRowTextLeft + historyRowIconSize + historyRowGap
}

// rowContentWidth returns the content column width for a cell of width.
func rowContentWidth(width float64) float64 {
	content := width - historyRowInsetX - historyRowTextInset - rowContentLeft()
	if content < 40 {
		content = 40
	}
	return content
}

// fitImageSize scales an image's natural size to fit the content column
// and the configured maximum height without upscaling small images.
func fitImageSize(natural CGSize, maxWidth, maxHeight float64) CGSize {
	if maxWidth < 1 {
		maxWidth = 1
	}
	if maxHeight < 20 {
		maxHeight = 20
	}
	if natural.Width <= 0 || natural.Height <= 0 {
		edge := math.Min(maxHeight, maxWidth)
		return CGSize{Width: edge, Height: edge}
	}
	scale := math.Min(maxWidth/natural.Width, maxHeight/natural.Height)
	if scale > 1 {
		scale = 1
	}
	width := math.Max(natural.Width*scale, 1)
	height := math.Max(natural.Height*scale, 1)
	return CGSize{Width: width, Height: height}
}

// wrappedLines reports how many lines text occupies in a column of width
// at the given font, capped at maxLines.
func wrappedLines(text string, font objc.ID, width float64, maxLines int) int {
	if strings.TrimSpace(text) == "" {
		return 0
	}
	attributes := objc.ID(class("NSDictionary")).Send(sel("dictionaryWithObject:forKey:"),
		font, nsString("NSFont"))
	rect := objc.Send[CGRect](nsString(text), sel("boundingRectWithSize:options:attributes:"),
		CGSize{Width: width, Height: 1e7}, uint64(1|2), attributes)
	lineHeight := fontLineHeight(font)
	if lineHeight <= 0 {
		lineHeight = historyRowLineHeight
	}
	lines := int(math.Round(rect.Size.Height / lineHeight))
	if lines < 1 {
		lines = 1
	}
	if maxLines > 0 && lines > maxLines {
		lines = maxLines
	}
	return lines
}

func fontLineHeight(font objc.ID) float64 {
	ascender := objc.Send[float64](font, sel("ascender"))
	descender := objc.Send[float64](font, sel("descender"))
	leading := objc.Send[float64](font, sel("leading"))
	return ascender - descender + leading
}

func layoutHistoryRow(cell objc.ID, size CGSize) {
	if visible := objc.Send[CGRect](cell, sel("visibleRect")); visible.Size.Width > 0 {
		if available := visible.Origin.X + visible.Size.Width; available < size.Width {
			size.Width = available
		}
	}
	left := historyRowTextLeft
	icon := viewWithTag(cell, rowIconTag)
	setFrame(icon, left, (size.Height-historyRowIconSize)/2, historyRowIconSize, historyRowIconSize)

	// The quick-paste digit floats centered on the app icon while Command is
	// held, painted over the icon without touching the row layout.
	if hint := rowHintBadge(cell); hint != 0 {
		setFrame(hint, left+(historyRowIconSize-keyHintRowBadgeSize)/2, (size.Height-keyHintRowBadgeSize)/2,
			keyHintRowBadgeSize, keyHintRowBadgeSize)
	}

	contentLeft := rowContentLeft()
	contentRight := size.Width - historyRowInsetX - historyRowTextInset
	if contentRight-contentLeft < 48 {
		contentRight = contentLeft + 48
	}
	width := contentRight - contentLeft
	if width < 40 {
		width = 40
	}

	detail := viewWithTag(cell, rowDetailTag)
	title := viewWithTag(cell, rowTitleTag)
	preview := viewWithTag(cell, rowPreviewTag)
	imageView := viewWithTag(cell, rowImageTag)

	layout := rowLayoutFor(cell)
	if layout == nil {
		setFrame(detail, contentLeft, rowPadY, width, detailLineHeight)
		return
	}

	blockHeight := rowBlockHeight(layout, width)
	y := (size.Height - blockHeight) / 2
	if y < rowPadY {
		y = rowPadY
	}
	setFrame(detail, contentLeft, y, width, detailLineHeight)
	contentBottom := y + detailLineHeight + contentDetailGap

	switch {
	case layout.showImage:
		imageSize := fitImageSize(layout.imageSize, width, layout.maxImageH)
		setFrame(imageView, contentLeft, contentBottom, imageSize.Width, imageSize.Height)
		title.Send(sel("setHidden:"), true)
		preview.Send(sel("setHidden:"), true)
	case len(layout.files) > 0:
		rows := len(layout.files)
		for index := 0; index < rows; index++ {
			fileY := contentBottom + float64(rows-1-index)*(fileRowHeight+fileRowGap)
			icon := viewWithTag(cell, fileRowIconTag(index))
			label := viewWithTag(cell, fileRowLabelTag(index))
			setFrame(icon, contentLeft, fileY+(fileRowHeight-fileRowIconSize)/2, fileRowIconSize, fileRowIconSize)
			setFrame(label, contentLeft+fileRowIconSize+fileRowIconTextGap, fileY+(fileRowHeight-previewLineHeight)/2,
				width-fileRowIconSize-fileRowIconTextGap, previewLineHeight)
		}
	default:
		titleLines := wrappedLines(layout.titleText, historyRowFont(historyRowTitleSize), width, layout.titleMax)
		titleHeight := float64(max(titleLines, 1)) * titleLineHeight
		previewHeight := 0.0
		previewLines := 0
		if layout.hasPreview {
			previewLines = wrappedLines(layout.previewText, historyRowFont(historyRowPreviewSize), width, layout.previewMax)
			if previewLines > 0 {
				previewHeight = float64(previewLines) * previewLineHeight
			}
		}
		titleY := contentBottom
		if previewLines > 0 {
			setFrame(preview, contentLeft, contentBottom, width, previewHeight)
			titleY = contentBottom + previewHeight + previewTitleGap
		}
		setFrame(title, contentLeft, titleY, width, titleHeight)
	}
}

// rowBlockHeight is the height of the content block plus the metadata
// line for a laid-out row.
func rowBlockHeight(layout *rowLayout, width float64) float64 {
	switch {
	case layout.showImage:
		return fitImageSize(layout.imageSize, width, layout.maxImageH).Height + contentDetailGap + detailLineHeight
	case len(layout.files) > 0:
		rows := float64(len(layout.files))
		return rows*fileRowHeight + (rows-1)*fileRowGap + contentDetailGap + detailLineHeight
	default:
		titleLines := wrappedLines(layout.titleText, historyRowFont(historyRowTitleSize), width, layout.titleMax)
		block := float64(max(titleLines, 1)) * titleLineHeight
		if layout.hasPreview {
			if lines := wrappedLines(layout.previewText, historyRowFont(historyRowPreviewSize), width, layout.previewMax); lines > 0 {
				block += previewTitleGap + float64(lines)*previewLineHeight
			}
		}
		return block + contentDetailGap + detailLineHeight
	}
}

// historyRowHeight returns the natural height of one card: its content
// block plus symmetric vertical padding, capped by the configured density
// through the text, image and file measurements.
func historyRowHeight(row HistoryRow, display PanelDisplay, width float64) float64 {
	contentWidth := rowContentWidth(width)
	height := math.Max(rowHeightForDisplay(row, display, contentWidth, false),
		rowHeightForDisplay(row, display, contentWidth, true))
	return height + 2*rowPadY
}

func rowHeightForDisplay(row HistoryRow, display PanelDisplay, contentWidth float64, reveal bool) float64 {
	noteMode := strings.TrimSpace(row.Note) != "" && display.ShowOriginal && !reveal
	layout := &rowLayout{maxImageH: float64(display.ImageMaxHeight)}
	if noteMode {
		text := i18n.T(i18n.NoteDetailPrefix) + strings.TrimSpace(row.Note)
		lines := wrappedLines(text, historyRowFont(historyRowTitleSize), contentWidth, max(display.TextMaxLines, 1))
		return float64(max(lines, 1))*titleLineHeight + contentDetailGap + detailLineHeight
	}
	if row.ImagePath != "" {
		natural := CGSize{Width: float64(row.ImageWidth), Height: float64(row.ImageHeight)}
		return fitImageSize(natural, contentWidth, layout.maxImageH).Height + contentDetailGap + detailLineHeight
	}
	if len(row.Files) > 0 {
		rows := len(row.Files)
		if rows > maxRowFiles {
			rows = maxRowFiles
		}
		return float64(rows)*fileRowHeight + float64(rows-1)*fileRowGap + contentDetailGap + detailLineHeight
	}
	title, preview, _ := historyRowText(row, display, reveal)
	titleMax := max(display.TextMaxLines, 1)
	previewMax := 0
	if preview != "" && display.TextMaxLines > 1 {
		titleMax = 1
		previewMax = min(display.TextMaxLines-1, 4)
	} else {
		preview = ""
	}
	titleLines := wrappedLines(title, historyRowFont(historyRowTitleSize), contentWidth, titleMax)
	block := float64(max(titleLines, 1)) * titleLineHeight
	if preview != "" {
		if lines := wrappedLines(preview, historyRowFont(historyRowPreviewSize), contentWidth, previewMax); lines > 0 {
			block += previewTitleGap + float64(lines)*previewLineHeight
		}
	}
	return block + contentDetailGap + detailLineHeight
}

const historyRowTextInset = 16.0

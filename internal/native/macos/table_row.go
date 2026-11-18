//go:build darwin

package macos

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

const (
	historyRowInsetX   = 10.0
	historyRowInsetY   = 2.0
	historyRowRadius   = 10.0
	historyRowTextLeft = 15.0
	historyRowGap      = 11.0

	rowPadY            = 8.0
	titleLineHeight    = 16.0
	previewLineHeight  = 15.0
	detailLineHeight   = 14.0
	contentDetailGap   = 6.0
	previewTitleGap    = 2.0
	fileRowHeight      = 20.0
	fileRowGap         = 2.0
	fileRowIconSize    = 16.0
	fileRowIconTextGap = 6.0
	maxRowFiles        = 5
)

var (
	historyRowClass     objc.Class
	historyRowViewClass objc.Class

	associatedRowHoveredKey  byte
	associatedRowSelectedKey byte

	historyRowViewsOnce sync.Once
	historyRowViewsErr  error
)

// rowLayout is the per-cell render state the manual layout pass needs.
// NSTableCellView has no writable tag and layoutHistoryRow only receives
// the cell, so the state travels beside the view.
type rowLayout struct {
	textMode    bool
	hasPreview  bool
	titleText   string
	previewText string
	titleMax    int
	previewMax  int
	showImage   bool
	imageSize   CGSize
	maxImageH   float64
	files       []HistoryFile
}

var rowLayouts sync.Map // objc.ID -> *rowLayout

func storeRowLayout(cell objc.ID, layout *rowLayout) {
	rowLayouts.Store(cell, layout)
}

func rowLayoutFor(cell objc.ID) *rowLayout {
	if value, ok := rowLayouts.Load(cell); ok {
		return value.(*rowLayout)
	}
	return nil
}

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

	icon := newRowImageView(rowIconTag)
	cell.Send(sel("addSubview:"), icon)
	icon.Send(sel("release"))

	image := newRowImageView(rowImageTag)
	image.Send(sel("setHidden:"), true)
	cell.Send(sel("addSubview:"), image)
	image.Send(sel("release"))

	title := newRowLabel(rowTitleTag, historyRowFontWeight(historyRowTitleSize, 0.2),
		objc.ID(classNSColor).Send(sel("labelColor")), true)
	cell.Send(sel("addSubview:"), title)

	preview := newRowLabel(rowPreviewTag, historyRowFont(historyRowPreviewSize),
		objc.ID(classNSColor).Send(sel("labelColor")), true)
	cell.Send(sel("addSubview:"), preview)

	detail := newRowLabel(rowDetailTag, historyRowFont(historyRowDetailSize),
		objc.ID(classNSColor).Send(sel("tertiaryLabelColor")), false)
	cell.Send(sel("addSubview:"), detail)

	for index := 0; index < maxRowFiles; index++ {
		fileIcon := newRowImageView(fileRowIconTag(index))
		fileIcon.Send(sel("setHidden:"), true)
		cell.Send(sel("addSubview:"), fileIcon)
		fileIcon.Send(sel("release"))

		fileLabel := newRowLabel(fileRowLabelTag(index), historyRowFont(historyRowPreviewSize),
			objc.ID(classNSColor).Send(sel("labelColor")), false)
		fileLabel.Send(sel("setHidden:"), true)
		cell.Send(sel("addSubview:"), fileLabel)
	}

	return cell.Send(sel("autorelease"))
}

func newRowImageView(tag int64) objc.ID {
	view := objc.ID(class("NSImageView")).Send(sel("new"))
	view.Send(sel("setTag:"), tag)
	view.Send(sel("setImageScaling:"), int64(0))
	view.Send(sel("setAccessibilityElement:"), false)
	return view
}

func newRowLabel(tag int64, font, color objc.ID, wrapping bool) objc.ID {
	var label objc.ID
	if wrapping {
		label = objc.ID(classNSTextField).Send(sel("wrappingLabelWithString:"), nsString(""))
		if titleCell := label.Send(sel("cell")); titleCell != 0 {
			titleCell.Send(sel("setWraps:"), true)
			titleCell.Send(sel("setTruncatesLastVisibleLine:"), true)
		}
	} else {
		label = objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	}
	label.Send(sel("setTag:"), tag)
	label.Send(sel("setFont:"), font)
	label.Send(sel("setTextColor:"), color)
	label.Send(sel("setLineBreakMode:"), int64(4))
	label.Send(sel("setMaximumNumberOfLines:"), int64(1))
	label.Send(sel("setSelectable:"), false)
	return label
}

func fillHistoryRow(cell objc.ID, row HistoryRow, display PanelDisplay, revealOriginal, selected bool) {
	title := viewWithTag(cell, rowTitleTag)
	preview := viewWithTag(cell, rowPreviewTag)
	detail := viewWithTag(cell, rowDetailTag)
	imageView := viewWithTag(cell, rowImageTag)

	titleText, previewText, detailText := historyRowText(row, display, revealOriginal)
	noteMode := strings.TrimSpace(row.Note) != "" && display.ShowOriginal && !revealOriginal

	layout := &rowLayout{maxImageH: float64(display.ImageMaxHeight)}
	showImage := false
	if !noteMode && row.ImagePath != "" {
		if image := loadImage(row.ImagePath); image != 0 {
			imageView.Send(sel("setImage:"), image)
			imageView.Send(sel("setContentTintColor:"), objc.ID(0))
			layout.imageSize = CGSize{Width: float64(row.ImageWidth), Height: float64(row.ImageHeight)}
			showImage = true
		}
	}
	files := row.Files
	if !noteMode && !showImage && len(files) == 0 && row.Kind == "files" && row.ImagePath != "" {
		// The file can become unreadable after the row was built. Keep the
		// card a regular file entry rather than a lone file name.
		files = []HistoryFile{{Name: filepath.Base(row.ImagePath), IsImage: true}}
	}
	fileCount := 0
	if !noteMode && !showImage && len(files) > 0 {
		fileCount = len(files)
		if fileCount > maxRowFiles {
			fileCount = maxRowFiles
		}
		layout.files = files[:fileCount]
	}
	textMode := !noteMode && !showImage && fileCount == 0
	if noteMode {
		previewText = ""
		textMode = true
	}

	layout.textMode = textMode
	layout.showImage = showImage
	layout.titleText = titleText
	layout.titleMax = 1
	layout.previewMax = 1
	if textMode {
		if previewText != "" && display.TextMaxLines > 1 {
			layout.hasPreview = true
			layout.titleMax = 1
			layout.previewMax = min(display.TextMaxLines-1, 4)
		} else {
			layout.titleMax = max(display.TextMaxLines, 1)
			previewText = ""
		}
		layout.previewText = previewText
	}

	title.Send(sel("setStringValue:"), nsString(titleText))
	title.Send(sel("setHidden:"), !textMode)
	preview.Send(sel("setStringValue:"), nsString(previewText))
	preview.Send(sel("setHidden:"), !textMode || previewText == "")
	detail.Send(sel("setStringValue:"), nsString(detailText))
	imageView.Send(sel("setHidden:"), !showImage)

	weight := 0.2
	if row.Kind == "text" {
		weight = 0
	}
	title.Send(sel("setFont:"), historyRowFontWeight(historyRowTitleSize, weight))
	title.Send(sel("setMaximumNumberOfLines:"), int64(layout.titleMax))
	title.Send(sel("setLineBreakMode:"), int64(0))
	preview.Send(sel("setMaximumNumberOfLines:"), int64(layout.previewMax))
	preview.Send(sel("setLineBreakMode:"), int64(0))

	fillFileRows(cell, layout.files, selected)

	setHistoryRowTextColors(cell, selected)
	setHistoryRowSelected(cell, selected)
	applySourceIcon(cell, row)
	storeRowLayout(cell, layout)
	layoutHistoryRow(cell, bounds(cell).Size)
}

// applySourceIcon puts the source application icon in the fixed left
// column. Rows without a stored icon fall back to the kind symbol.
func applySourceIcon(cell objc.ID, row HistoryRow) {
	icon := viewWithTag(cell, rowIconTag)
	if row.SourceIconPath != "" {
		if image := loadImage(row.SourceIconPath); image != 0 {
			icon.Send(sel("setImage:"), image)
			icon.Send(sel("setContentTintColor:"), objc.ID(0))
			return
		}
	}
	icon.Send(sel("setImage:"), symbolImage(row.Kind))
	icon.Send(sel("setContentTintColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
}

func fillFileRows(cell objc.ID, files []HistoryFile, selected bool) {
	for index := 0; index < maxRowFiles; index++ {
		icon := viewWithTag(cell, fileRowIconTag(index))
		label := viewWithTag(cell, fileRowLabelTag(index))
		if index >= len(files) {
			icon.Send(sel("setHidden:"), true)
			label.Send(sel("setHidden:"), true)
			continue
		}
		entry := files[index]
		if entry.IconPath != "" {
			if image := loadImage(entry.IconPath); image != 0 {
				icon.Send(sel("setImage:"), image)
				icon.Send(sel("setContentTintColor:"), objc.ID(0))
			} else {
				setFileKindIcon(icon, entry)
			}
		} else {
			setFileKindIcon(icon, entry)
		}
		name := entry.Name
		color := objc.ID(classNSColor).Send(sel("labelColor"))
		if selected {
			color = objc.ID(classNSColor).Send(sel("alternateSelectedControlTextColor"))
		} else if entry.Missing {
			color = objc.ID(classNSColor).Send(sel("secondaryLabelColor"))
		}
		if entry.Missing {
			name = name + " " + i18n.T(i18n.FileMissing)
		}
		label.Send(sel("setStringValue:"), nsString(name))
		label.Send(sel("setTextColor:"), color)
		icon.Send(sel("setHidden:"), false)
		label.Send(sel("setHidden:"), false)
	}
}

func setFileKindIcon(icon objc.ID, entry HistoryFile) {
	icon.Send(sel("setImage:"), systemSymbol(fileKindSymbol(entry)))
	icon.Send(sel("setContentTintColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
}

// fileKindSymbol keeps the fallback icon recognizable when the Finder icon
// cache has no entry, including image files whose contents cannot be read.
func fileKindSymbol(entry HistoryFile) string {
	switch {
	case entry.IsDir:
		return "folder"
	case entry.IsImage:
		return "photo"
	default:
		return "doc"
	}
}

func viewWithTag(cell objc.ID, tag int64) objc.ID {
	return cell.Send(sel("viewWithTag:"), tag)
}

func fileRowIconTag(index int) int64  { return int64(rowFileIconBase + index) }
func fileRowLabelTag(index int) int64 { return int64(rowFileLabelBase + index) }

// setHistoryRowTextColors switches a row between normal and selected
// text, because the selected card is drawn with the accent color.
func setHistoryRowTextColors(cell objc.ID, selected bool) {
	title := viewWithTag(cell, rowTitleTag)
	preview := viewWithTag(cell, rowPreviewTag)
	detail := viewWithTag(cell, rowDetailTag)
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
	if layout := rowLayoutFor(cell); layout != nil {
		for index := 0; index < len(layout.files) && index < maxRowFiles; index++ {
			labelColor := objc.ID(classNSColor).Send(sel("labelColor"))
			if selected {
				labelColor = objc.ID(classNSColor).Send(sel("alternateSelectedControlTextColor"))
			} else if layout.files[index].Missing {
				labelColor = objc.ID(classNSColor).Send(sel("secondaryLabelColor"))
			}
			viewWithTag(cell, fileRowLabelTag(index)).Send(sel("setTextColor:"), labelColor)
		}
	}
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

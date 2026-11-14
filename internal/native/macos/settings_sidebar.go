//go:build darwin

package macos

import (
	"sync"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

const (
	settingsSidebarHeaderH = 48.0
	settingsSidebarRowH    = 32.0
	settingsSidebarIconTag = 300
	settingsSidebarTextTag = 301
)

var settingsTabOrder = []string{
	"general", "data", "capture", "display", "actions", "interaction", "deletion",
	"window", "history", "permissions", "apps", "shortcuts", "preview", "update",
}

var settingsTabSymbols = map[string]string{
	"general":     "gearshape",
	"data":        "externaldrive",
	"capture":     "tray.and.arrow.down",
	"display":     "textformat.size",
	"actions":     "cursorarrow.click",
	"interaction": "hand.tap",
	"deletion":    "trash",
	"window":      "rectangle.on.rectangle",
	"history":     "clock.arrow.circlepath",
	"permissions": "lock.shield",
	"apps":        "square.grid.2x2",
	"shortcuts":   "keyboard",
	"preview":     "eye",
	"update":      "arrow.triangle.2.circlepath",
}

var (
	registerSettingsSidebarOnce sync.Once
	settingsSidebarTarget       objc.ID
	settingsSidebarRowClass     objc.Class
	settingsSidebarRowViewClass objc.Class
	settingsSidebarByTable      = map[objc.ID]*SettingsView{}
)

func settingsTabIndex(name string) int {
	for index, candidate := range settingsTabOrder {
		if candidate == name {
			return index
		}
	}
	return -1
}

func settingsTabSymbol(name string) string {
	if symbol, ok := settingsTabSymbols[name]; ok {
		return symbol
	}
	return "circle"
}

func (v *SettingsView) buildSidebar() {
	registerSettingsSidebarOnce.Do(registerSettingsSidebarTargets)

	sidebar := objc.ID(class("NSVisualEffectView")).Send(sel("new"))
	sidebar.Send(sel("setMaterial:"), int64(7))
	sidebar.Send(sel("setBlendingMode:"), int64(1))
	sidebar.Send(sel("setState:"), int64(1))
	sidebar.Send(sel("setFrame:"), CGRect{
		Size: CGSize{Width: settingsSidebarW, Height: settingsHeight},
	})
	sidebar.Send(sel("setAutoresizingMask:"), int64(16))
	v.view.Send(sel("addSubview:"), sidebar)
	v.sidebar = sidebar

	appIcon := objc.Send[objc.ID](
		objc.ID(class("NSApplication")).Send(sel("sharedApplication")), sel("applicationIconImage"))
	icon := objc.ID(class("NSImageView")).Send(sel("new"))
	icon.Send(sel("setImage:"), appIcon)
	icon.Send(sel("setImageScaling:"), int64(3))
	icon.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: 14, Y: settingsHeight - 34},
		Size:   CGSize{Width: 22, Height: 22},
	})
	sidebar.Send(sel("addSubview:"), icon)
	icon.Send(sel("release"))

	name := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString("EcoPaste"))
	name.Send(sel("setFont:"), objc.ID(classNSFont).Send(sel("boldSystemFontOfSize:"), 13.0))
	name.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: 44, Y: settingsHeight - 32},
		Size:   CGSize{Width: settingsSidebarW - 58, Height: 18},
	})
	sidebar.Send(sel("addSubview:"), name)

	scroll := objc.ID(class("NSScrollView")).Send(sel("new"))
	scroll.Send(sel("setFrame:"), CGRect{
		Size: CGSize{Width: settingsSidebarW, Height: settingsHeight - settingsSidebarHeaderH},
	})
	scroll.Send(sel("setAutoresizingMask:"), int64(16))
	scroll.Send(sel("setHasVerticalScroller:"), true)
	scroll.Send(sel("setAutohidesScrollers:"), true)
	scroll.Send(sel("setDrawsBackground:"), false)
	scroll.Send(sel("setBorderType:"), int64(0))

	table := objc.ID(class("NSTableView")).Send(sel("new"))
	table.Send(sel("setRowHeight:"), settingsSidebarRowH)
	table.Send(sel("setIntercellSpacing:"), CGSize{})
	table.Send(sel("setFrame:"), CGRect{
		Size: CGSize{
			Width:  settingsSidebarW,
			Height: float64(len(settingsTabOrder)) * settingsSidebarRowH,
		},
	})
	table.Send(sel("setAutoresizingMask:"), int64(2))
	table.Send(sel("setHeaderView:"), objc.ID(0))
	table.Send(sel("setGridStyleMask:"), int64(0))
	table.Send(sel("setAllowsMultipleSelection:"), false)
	table.Send(sel("setAllowsEmptySelection:"), false)
	table.Send(sel("setSelectionHighlightStyle:"), int64(1))
	table.Send(sel("setFocusRingType:"), int64(1))
	table.Send(sel("setStyle:"), int64(3))
	table.Send(sel("setBackgroundColor:"), objc.ID(classNSColor).Send(sel("clearColor")))
	table.Send(sel("setDataSource:"), settingsSidebarTarget)
	table.Send(sel("setDelegate:"), settingsSidebarTarget)
	table.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.SettingsSidebarLabel)))

	column := objc.ID(class("NSTableColumn")).Send(sel("new"))
	column.Send(sel("setIdentifier:"), nsString("settings.sidebar"))
	column.Send(sel("setResizingMask:"), int64(1))
	column.Send(sel("setWidth:"), settingsSidebarW)
	table.Send(sel("addTableColumn:"), column)
	column.Send(sel("release"))
	table.Send(sel("setColumnAutoresizingStyle:"), int64(1))
	table.Send(sel("reloadData"))

	scroll.Send(sel("setDocumentView:"), table)
	table.Send(sel("release"))
	sidebar.Send(sel("addSubview:"), scroll)
	scroll.Send(sel("release"))

	v.sidebarTable = table
	settingsSidebarByTable[table] = v

	border := objc.ID(classNSBox).Send(sel("new"))
	border.Send(sel("setBoxType:"), 2)
	border.Send(sel("setTitlePosition:"), 0)
	border.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsSidebarW - 1, Y: 0},
		Size:   CGSize{Width: 1, Height: settingsHeight},
	})
	v.view.Send(sel("addSubview:"), border)
}

func (v *SettingsView) buildSettingsHeader() {
	v.sectionTitle = objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	v.sectionTitle.Send(sel("setFont:"), objc.ID(classNSFont).Send(sel("boldSystemFontOfSize:"), 15.0))
	v.sectionTitle.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsSidebarW + 24, Y: settingsHeight - 33},
		Size:   CGSize{Width: 280, Height: 22},
	})
	v.view.Send(sel("addSubview:"), v.sectionTitle)

	headerDivider := objc.ID(classNSBox).Send(sel("new"))
	headerDivider.Send(sel("setBoxType:"), 2)
	headerDivider.Send(sel("setTitlePosition:"), 0)
	headerDivider.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsSidebarW, Y: settingsFooterH + settingsTabAreaH},
		Size:   CGSize{Width: settingsWidth - settingsSidebarW, Height: 1},
	})
	v.view.Send(sel("addSubview:"), headerDivider)

	footerDivider := objc.ID(classNSBox).Send(sel("new"))
	footerDivider.Send(sel("setBoxType:"), 2)
	footerDivider.Send(sel("setTitlePosition:"), 0)
	footerDivider.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsSidebarW, Y: settingsFooterH - 1},
		Size:   CGSize{Width: settingsWidth - settingsSidebarW, Height: 1},
	})
	v.view.Send(sel("addSubview:"), footerDivider)
}

func (v *SettingsView) selectSidebarRow(name string) {
	if v.sidebarTable == 0 {
		return
	}
	index := settingsTabIndex(name)
	if index < 0 {
		return
	}
	v.sidebarTable.Send(sel("reloadData"))
	if int(objc.Send[int64](v.sidebarTable, sel("selectedRow"))) != index {
		indexes := objc.ID(class("NSIndexSet")).Send(sel("indexSetWithIndex:"), int64(index))
		v.sidebarTable.Send(sel("selectRowIndexes:byExtendingSelection:"), indexes, false)
		v.sidebarTable.Send(sel("scrollRowToVisible:"), int64(index))
	}
}

func registerSettingsSidebarTargets() {
	var err error
	settingsSidebarRowClass, err = objc.RegisterClass("EcoPasteSettingsSidebarRow",
		class("NSTableCellView"), nil, nil, []objc.MethodDef{
			{Cmd: sel("setFrameSize:"), Fn: func(self objc.ID, _ objc.SEL, size CGSize) {
				sendSuper[objc.ID](self, settingsSidebarRowClass, sel("setFrameSize:"), size)
				layoutSettingsSidebarRow(self, size)
			}},
		})
	if err != nil {
		panic(err)
	}

	settingsSidebarRowViewClass, err = objc.RegisterClass("EcoPasteSettingsSidebarRowView",
		class("NSTableRowView"), nil, nil, []objc.MethodDef{
			{Cmd: sel("drawSelectionInRect:"), Fn: drawSettingsSidebarSelection},
		})
	if err != nil {
		panic(err)
	}

	target, err := objc.RegisterClass("EcoPasteSettingsSidebarTarget", class("NSObject"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("numberOfRowsInTableView:"), Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) int64 {
				return int64(len(settingsTabOrder))
			}},
			{Cmd: sel("tableView:viewForTableColumn:row:"), Fn: func(_ objc.ID, _ objc.SEL, table, _ objc.ID, row int64) objc.ID {
				v := settingsSidebarByTable[table]
				if v == nil || row < 0 || int(row) >= len(settingsTabOrder) {
					return 0
				}
				cell := table.Send(sel("makeViewWithIdentifier:owner:"), nsString("settings.sidebar.row"), table)
				if cell == 0 {
					cell = newSettingsSidebarRow()
				}
				selected := objc.Send[int64](table, sel("selectedRow")) == row
				fillSettingsSidebarRow(cell, settingsTabOrder[row], selected)
				return cell
			}},
			{Cmd: sel("tableView:rowViewForRow:"), Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID, _ int64) objc.ID {
				rowView := objc.ID(settingsSidebarRowViewClass).Send(sel("new"))
				return rowView.Send(sel("autorelease"))
			}},
			{Cmd: sel("tableViewSelectionDidChange:"), Fn: func(_ objc.ID, _ objc.SEL, note objc.ID) {
				table := objc.Send[objc.ID](note, sel("object"))
				v := settingsSidebarByTable[table]
				if v == nil {
					return
				}
				row := int(objc.Send[int64](table, sel("selectedRow")))
				if row < 0 || row >= len(settingsTabOrder) {
					return
				}
				v.applyTab(settingsTabOrder[row])
				refreshSettingsSidebarRows(table)
			}},
		})
	if err != nil {
		panic(err)
	}
	settingsSidebarTarget = objc.ID(target).Send(sel("new"))
}

func drawSettingsSidebarSelection(self objc.ID, _ objc.SEL, rect CGRect) {
	inset := CGRect{
		Origin: CGPoint{X: rect.Origin.X + 8, Y: rect.Origin.Y + 2},
		Size:   CGSize{Width: rect.Size.Width - 16, Height: rect.Size.Height - 4},
	}
	path := objc.ID(class("NSBezierPath")).Send(sel("bezierPathWithRoundedRect:xRadius:yRadius:"),
		inset, 6.0, 6.0)
	colorSelector := sel("unemphasizedSelectedContentBackgroundColor")
	if objc.Send[bool](self, sel("isEmphasized")) {
		colorSelector = sel("selectedContentBackgroundColor")
	}
	objc.ID(classNSColor).Send(colorSelector).Send(sel("setFill"))
	path.Send(sel("fill"))
}

func refreshSettingsSidebarRows(table objc.ID) {
	selected := objc.Send[int64](table, sel("selectedRow"))
	for index, name := range settingsTabOrder {
		cell := table.Send(sel("viewAtColumn:row:makeIfNecessary:"), int64(0), int64(index), false)
		if cell == 0 {
			continue
		}
		fillSettingsSidebarRow(cell, name, selected == int64(index))
	}
}

func newSettingsSidebarRow() objc.ID {
	cell := objc.ID(settingsSidebarRowClass).Send(sel("new"))
	cell.Send(sel("setIdentifier:"), nsString("settings.sidebar.row"))

	icon := objc.ID(class("NSImageView")).Send(sel("new"))
	icon.Send(sel("setTag:"), int64(settingsSidebarIconTag))
	icon.Send(sel("setImageScaling:"), int64(3))
	icon.Send(sel("setImageAlignment:"), int64(5))
	cell.Send(sel("addSubview:"), icon)
	icon.Send(sel("release"))

	text := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	text.Send(sel("setTag:"), int64(settingsSidebarTextTag))
	text.Send(sel("setFont:"), systemFont(13))
	text.Send(sel("setLineBreakMode:"), int64(4))
	text.Send(sel("setMaximumNumberOfLines:"), int64(1))
	cell.Send(sel("addSubview:"), text)

	return cell.Send(sel("autorelease"))
}

func fillSettingsSidebarRow(cell objc.ID, name string, selected bool) {
	icon := cell.Send(sel("viewWithTag:"), int64(settingsSidebarIconTag))
	text := cell.Send(sel("viewWithTag:"), int64(settingsSidebarTextTag))
	tint := objc.ID(classNSColor).Send(sel("secondaryLabelColor"))
	if selected {
		tint = objc.ID(classNSColor).Send(sel("controlAccentColor"))
	}
	icon.Send(sel("setImage:"), systemSymbol(settingsTabSymbol(name)))
	icon.Send(sel("setContentTintColor:"), tint)
	text.Send(sel("setStringValue:"), nsString(settingsTabTitle(name)))
}

func layoutSettingsSidebarRow(cell objc.ID, size CGSize) {
	const iconSize = 18.0
	icon := cell.Send(sel("viewWithTag:"), int64(settingsSidebarIconTag))
	text := cell.Send(sel("viewWithTag:"), int64(settingsSidebarTextTag))
	setFrame(icon, 12, (size.Height-iconSize)/2, iconSize, iconSize)
	setFrame(text, 38, (size.Height-17)/2, size.Width-50, 17)
}

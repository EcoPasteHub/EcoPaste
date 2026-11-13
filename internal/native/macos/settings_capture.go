//go:build darwin

package macos

import (
	"fmt"
	"sync"

	"github.com/ebitengine/purego/objc"
)

const (
	settingsCaptureRowIdentifier = "ecopaste.settings-capture-row"
	settingsCaptureRowHeight     = 20.0
)

var (
	registerSettingsCaptureOnce sync.Once
	settingsCaptureTarget       objc.ID
	settingsCaptureRowClass     objc.Class
	settingsCaptureByTable      = map[objc.ID]*SettingsView{}
)

var settingsCaptureTitles = map[string]string{
	"files": "文件",
	"image": "图片",
	"html":  "HTML",
	"rtf":   "RTF",
	"text":  "纯文本",
}

func (v *SettingsView) buildCaptureOrder(content objc.ID) {
	registerSettingsCaptureOnce.Do(registerSettingsCaptureTargets)
	title := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString("采集优先级"))
	title.Send(sel("setFont:"), systemFont(13))
	title.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: 132}, Size: CGSize{Width: 120, Height: 18}})
	content.Send(sel("addSubview:"), title)

	hint := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString("从上到下依次选择可用的主表示"))
	hint.Send(sel("setFont:"), systemFont(11))
	hint.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	hint.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 142, Y: 132}, Size: CGSize{Width: 230, Height: 18}})
	content.Send(sel("addSubview:"), hint)

	scroll := objc.ID(class("NSScrollView")).Send(sel("new"))
	scroll.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 20, Y: 18}, Size: CGSize{Width: tabContentW - 108, Height: 106}})
	scroll.Send(sel("setHasVerticalScroller:"), true)
	scroll.Send(sel("setAutohidesScrollers:"), true)
	scroll.Send(sel("setDrawsBackground:"), false)
	scroll.Send(sel("setBorderType:"), int64(2))

	table := objc.ID(class("NSTableView")).Send(sel("new"))
	table.Send(sel("setRowHeight:"), settingsCaptureRowHeight)
	table.Send(sel("setIntercellSpacing:"), CGSize{})
	table.Send(sel("setHeaderView:"), objc.ID(0))
	table.Send(sel("setGridStyleMask:"), int64(0))
	table.Send(sel("setAllowsMultipleSelection:"), false)
	table.Send(sel("setAllowsEmptySelection:"), false)
	table.Send(sel("setSelectionHighlightStyle:"), int64(1))
	table.Send(sel("setFocusRingType:"), int64(1))
	table.Send(sel("setStyle:"), int64(3))
	table.Send(sel("setBackgroundColor:"), objc.ID(classNSColor).Send(sel("clearColor")))
	table.Send(sel("setDataSource:"), settingsCaptureTarget)
	table.Send(sel("setDelegate:"), settingsCaptureTarget)
	table.Send(sel("setAccessibilityLabel:"), nsString("采集优先级"))

	column := objc.ID(class("NSTableColumn")).Send(sel("new"))
	column.Send(sel("setIdentifier:"), nsString("capture"))
	column.Send(sel("setResizingMask:"), int64(1))
	column.Send(sel("setWidth:"), float64(tabContentW-112))
	table.Send(sel("addTableColumn:"), column)
	column.Send(sel("release"))
	table.Send(sel("setColumnAutoresizingStyle:"), int64(1))

	scroll.Send(sel("setDocumentView:"), table)
	table.Send(sel("release"))
	content.Send(sel("addSubview:"), scroll)
	scroll.Send(sel("release"))
	v.captureTable = table
	settingsCaptureByTable[table] = v

	up := objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString("上移"), settingsCaptureTarget, sel("captureUpPressed:"))
	up.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: tabContentW - 78, Y: 72}, Size: CGSize{Width: 58, Height: 28}})
	up.Send(sel("setAccessibilityLabel:"), nsString("上移选中采集类型"))
	content.Send(sel("addSubview:"), up)
	settingsViews[up] = v

	down := objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString("下移"), settingsCaptureTarget, sel("captureDownPressed:"))
	down.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: tabContentW - 78, Y: 38}, Size: CGSize{Width: 58, Height: 28}})
	down.Send(sel("setAccessibilityLabel:"), nsString("下移选中采集类型"))
	content.Send(sel("addSubview:"), down)
	settingsViews[down] = v
}

func (v *SettingsView) applyCaptureOrder(values SettingsValues) {
	order := make([]string, 0, len(settingsCaptureTitles))
	seen := map[string]struct{}{}
	for _, key := range values.CaptureOrder {
		if _, ok := settingsCaptureTitles[key]; !ok {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		order = append(order, key)
	}
	for _, key := range []string{"files", "image", "html", "rtf", "text"} {
		if _, ok := seen[key]; ok {
			continue
		}
		order = append(order, key)
	}
	v.captureOrder = order
	if v.captureTable == 0 {
		return
	}
	selected := int(objc.Send[int64](v.captureTable, sel("selectedRow")))
	v.captureTable.Send(sel("reloadData"))
	if selected >= len(order) {
		selected = len(order) - 1
	}
	if selected < 0 && len(order) > 0 {
		selected = 0
	}
	if selected >= 0 {
		v.selectCaptureOrder(selected)
	}
}

func (v *SettingsView) selectCaptureOrder(row int) {
	if v.captureTable == 0 || row < 0 || row >= len(v.captureOrder) {
		return
	}
	indexes := objc.ID(class("NSIndexSet")).Send(sel("indexSetWithIndex:"), int64(row))
	v.captureTable.Send(sel("selectRowIndexes:byExtendingSelection:"), indexes, false)
	v.captureTable.Send(sel("scrollRowToVisible:"), int64(row))
}

func (v *SettingsView) moveCaptureOrder(delta int) {
	if v.captureTable == 0 {
		return
	}
	row := int(objc.Send[int64](v.captureTable, sel("selectedRow")))
	next := row + delta
	if row < 0 || row >= len(v.captureOrder) || next < 0 || next >= len(v.captureOrder) {
		return
	}
	v.captureOrder[row], v.captureOrder[next] = v.captureOrder[next], v.captureOrder[row]
	v.captureTable.Send(sel("reloadData"))
	v.selectCaptureOrder(next)
	order := append([]string(nil), v.captureOrder...)
	v.cx.Emit("change", SettingsChange{Path: "capture.updateOrder", Value: order})
}

func registerSettingsCaptureTargets() {
	var err error
	settingsCaptureRowClass, err = objc.RegisterClass("EcoPasteSettingsCaptureRow", class("NSTableCellView"), nil, nil,
		[]objc.MethodDef{{Cmd: sel("setFrameSize:"), Fn: func(self objc.ID, _ objc.SEL, size CGSize) {
			sendSuper[objc.ID](self, settingsCaptureRowClass, sel("setFrameSize:"), size)
			if title := self.Send(sel("viewWithTag:"), int64(1)); title != 0 {
				title.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 8, Y: 2}, Size: CGSize{Width: size.Width - 16, Height: 17}})
			}
		}}})
	if err != nil {
		panic(err)
	}
	target, err := objc.RegisterClass("EcoPasteSettingsCaptureTarget", class("NSObject"), nil, nil, []objc.MethodDef{
		{Cmd: sel("numberOfRowsInTableView:"), Fn: func(_ objc.ID, _ objc.SEL, table objc.ID) int64 {
			if v := settingsCaptureByTable[table]; v != nil {
				return int64(len(v.captureOrder))
			}
			return 0
		}},
		{Cmd: sel("tableView:viewForTableColumn:row:"), Fn: func(_ objc.ID, _ objc.SEL, table, _ objc.ID, row int64) objc.ID {
			v := settingsCaptureByTable[table]
			if v == nil || row < 0 || int(row) >= len(v.captureOrder) {
				return 0
			}
			cell := table.Send(sel("makeViewWithIdentifier:owner:"), nsString(settingsCaptureRowIdentifier), table)
			if cell == 0 {
				cell = newSettingsCaptureRow()
			}
			key := v.captureOrder[row]
			title := settingsCaptureTitles[key]
			if title == "" {
				title = key
			}
			if label := cell.Send(sel("viewWithTag:"), int64(1)); label != 0 {
				label.Send(sel("setStringValue:"), nsString(title))
				label.Send(sel("setAccessibilityLabel:"), nsString(fmt.Sprintf("第 %d 优先级：%s", row+1, title)))
			}
			return cell
		}},
		{Cmd: sel("captureUpPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			if v := settingsViews[sender]; v != nil {
				v.moveCaptureOrder(-1)
			}
		}},
		{Cmd: sel("captureDownPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			if v := settingsViews[sender]; v != nil {
				v.moveCaptureOrder(1)
			}
		}},
	})
	if err != nil {
		panic(err)
	}
	settingsCaptureTarget = objc.ID(target).Send(sel("new"))
}

func newSettingsCaptureRow() objc.ID {
	cell := objc.ID(settingsCaptureRowClass).Send(sel("new"))
	cell.Send(sel("setIdentifier:"), nsString(settingsCaptureRowIdentifier))

	title := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	title.Send(sel("setTag:"), int64(1))
	title.Send(sel("setFont:"), systemFont(13))
	title.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("labelColor")))
	title.Send(sel("setLineBreakMode:"), int64(4))
	title.Send(sel("setMaximumNumberOfLines:"), int64(1))
	cell.Send(sel("addSubview:"), title)

	return cell.Send(sel("autorelease"))
}

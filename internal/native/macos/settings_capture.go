//go:build darwin

package macos

import (
	"sync"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

const (
	settingsCaptureRowIdentifier = "ecopaste.settings-capture-row"
	settingsCaptureRowHeight     = 20.0
	settingsCaptureListWidth     = tabContentW - 140
	settingsCaptureButtonX       = tabContentW - 112
	settingsCaptureButtonWidth   = 92
)

var (
	registerSettingsCaptureOnce sync.Once
	settingsCaptureTarget       objc.ID
	settingsCaptureRowClass     objc.Class
	settingsCaptureByTable      = map[objc.ID]*SettingsView{}
)

func settingsCaptureTitle(key string) (string, bool) {
	switch key {
	case "files":
		return i18n.T(i18n.KindFiles), true
	case "image":
		return i18n.T(i18n.KindImage), true
	case "html":
		return i18n.T(i18n.SettingsCaptureHTML), true
	case "rtf":
		return i18n.T(i18n.SettingsCaptureRTF), true
	case "text":
		return i18n.T(i18n.SettingsCaptureText), true
	}
	return "", false
}

func (v *SettingsView) buildCaptureOrder(content objc.ID) {
	registerSettingsCaptureOnce.Do(registerSettingsCaptureTargets)
	v.addSearchEntry(content, i18n.T(i18n.SettingsCaptureOrder), "clipboard.capture.order",
		i18n.T(i18n.SettingsCaptureKeywords), "capture order priority",
		CGRect{Origin: CGPoint{X: 12, Y: 10}, Size: CGSize{Width: tabContentW - 24, Height: 146}})
	title := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(i18n.T(i18n.SettingsCaptureOrder)))
	title.Send(sel("setFont:"), objc.ID(classNSFont).Send(sel("systemFontOfSize:weight:"), 11.0, 0.3))
	title.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	title.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: 132}, Size: CGSize{Width: 120, Height: 18}})
	content.Send(sel("addSubview:"), title)

	hint := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(i18n.T(i18n.SettingsCaptureHint)))
	hint.Send(sel("setFont:"), systemFont(11))
	hint.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("tertiaryLabelColor")))
	hint.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 142, Y: 132}, Size: CGSize{Width: 354, Height: 18}})
	content.Send(sel("addSubview:"), hint)

	table := addSettingsListTable(content,
		CGRect{Origin: CGPoint{X: 20, Y: 18}, Size: CGSize{Width: settingsCaptureListWidth, Height: 106}},
		settingsCaptureRowHeight, float64(settingsCaptureListWidth-4), "capture",
		i18n.T(i18n.SettingsCaptureOrder), settingsCaptureTarget)
	v.captureTable = table
	settingsCaptureByTable[table] = v

	v.captureUp = v.addSettingsMoveButton(content,
		CGRect{Origin: CGPoint{X: settingsCaptureButtonX, Y: 72}, Size: CGSize{Width: settingsCaptureButtonWidth, Height: 28}},
		i18n.T(i18n.SettingsMoveUp), i18n.T(i18n.SettingsMoveUpA11y), settingsCaptureTarget, "captureUpPressed:")
	v.captureDown = v.addSettingsMoveButton(content,
		CGRect{Origin: CGPoint{X: settingsCaptureButtonX, Y: 38}, Size: CGSize{Width: settingsCaptureButtonWidth, Height: 28}},
		i18n.T(i18n.SettingsMoveDown), i18n.T(i18n.SettingsMoveDownA11y), settingsCaptureTarget, "captureDownPressed:")
}

func (v *SettingsView) applyCaptureOrder(values SettingsValues) {
	order := make([]string, 0, 5)
	seen := map[string]struct{}{}
	for _, key := range values.CaptureOrder {
		if _, ok := settingsCaptureTitle(key); !ok {
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
	v.captureTable.Send(sel("reloadData"))
	sizeSettingsTable(v.captureTable, settingsCaptureListWidth, settingsCaptureRowHeight, len(order))
	if selected := selectedRowAfterResize(v.captureTable, len(order)); selected >= 0 {
		v.selectCaptureOrder(selected)
	}
	v.updateCaptureMoveButtons()
}

func (v *SettingsView) selectCaptureOrder(row int) {
	if v.captureTable == 0 || row < 0 || row >= len(v.captureOrder) {
		return
	}
	indexes := objc.ID(class("NSIndexSet")).Send(sel("indexSetWithIndex:"), int64(row))
	v.captureTable.Send(sel("selectRowIndexes:byExtendingSelection:"), indexes, false)
	v.captureTable.Send(sel("scrollRowToVisible:"), int64(row))
	v.updateCaptureMoveButtons()
}

func (v *SettingsView) updateCaptureMoveButtons() {
	row := -1
	if v.captureTable != 0 {
		row = int(objc.Send[int64](v.captureTable, sel("selectedRow")))
	}
	up, down := orderMoveEnabled(row, len(v.captureOrder))
	if v.captureUp != 0 {
		v.captureUp.Send(sel("setEnabled:"), up)
	}
	if v.captureDown != 0 {
		v.captureDown.Send(sel("setEnabled:"), down)
	}
}

func (v *SettingsView) moveCaptureOrder(delta int) {
	if v.captureTable == 0 {
		return
	}
	next, ok := moveSelectedRow(v.captureTable, v.captureOrder, delta)
	if !ok {
		return
	}
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
			title, ok := settingsCaptureTitle(key)
			if !ok {
				title = key
			}
			if label := cell.Send(sel("viewWithTag:"), int64(1)); label != 0 {
				label.Send(sel("setStringValue:"), nsString(title))
				label.Send(sel("setAccessibilityLabel:"),
					nsString(i18n.Tf(i18n.SettingsPriorityLabel, row+1, title)))
			}
			return cell
		}},
		{Cmd: sel("tableViewSelectionDidChange:"), Fn: func(_ objc.ID, _ objc.SEL, notification objc.ID) {
			if notification == 0 {
				return
			}
			table := notification.Send(sel("object"))
			if v := settingsCaptureByTable[table]; v != nil {
				v.updateCaptureMoveButtons()
			}
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

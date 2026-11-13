//go:build darwin

package macos

import (
	"sync"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

const (
	settingsActionRowIdentifier = "ecopaste.settings-action-row"
	settingsActionCheckTagBase  = 2000
)

// SettingsItemActions is the hover action configuration the settings
// window writes.
type SettingsItemActions struct {
	Enabled []string
	Order   []string
}

type settingsAction struct {
	key     string
	title   string
	enabled bool
}

var (
	registerSettingsActionsOnce sync.Once
	settingsActionsTarget       objc.ID
	settingsActionRowClass      objc.Class
	settingsActionsByTable      = map[objc.ID]*SettingsView{}
	settingsActionControls      = map[objc.ID]*SettingsView{}
)

func settingsActionTitle(key string) (string, bool) {
	switch key {
	case "paste":
		return i18n.T(i18n.MenuPaste), true
	case "pastePlain":
		return i18n.T(i18n.MenuPastePlain), true
	case "pastePath":
		return i18n.T(i18n.MenuPastePath), true
	case "copy":
		return i18n.T(i18n.MenuCopy), true
	case "copyPlain":
		return i18n.T(i18n.MenuCopyPlain), true
	case "openLink":
		return i18n.T(i18n.MenuOpenLink), true
	case "sendEmail":
		return i18n.T(i18n.MenuSendEmail), true
	case "reveal":
		return i18n.T(i18n.MenuReveal), true
	case "note":
		return i18n.T(i18n.SettingsActionNote), true
	case "star":
		return i18n.T(i18n.MenuFavorite), true
	case "pinItem":
		return i18n.T(i18n.MenuPin), true
	case "delete":
		return i18n.T(i18n.CommonDelete), true
	}
	return "", false
}

var settingsActionFallbackOrder = []string{
	"paste", "pastePlain", "pastePath", "copy", "copyPlain", "openLink",
	"sendEmail", "reveal", "note", "star", "pinItem", "delete",
}

func (v *SettingsView) buildActions() {
	registerSettingsActionsOnce.Do(registerSettingsActionsTargets)
	content := v.newTab("actions", settingsTabTitle("actions"))
	v.addSearchEntry(content, i18n.T(i18n.SettingsHoverActions), "clipboard.content.itemActions",
		i18n.T(i18n.SettingsHoverKeywords), "hover actions menu buttons",
		CGRect{Origin: CGPoint{X: 12, Y: 24}, Size: CGSize{Width: tabContentW - 24, Height: 420}})

	intro := objc.ID(classNSTextField).Send(sel("labelWithString:"),
		nsString(i18n.T(i18n.SettingsHoverIntro)))
	intro.Send(sel("setFont:"), systemFont(11))
	intro.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	intro.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: 438}, Size: CGSize{Width: tabContentW - 48, Height: 18}})
	content.Send(sel("addSubview:"), intro)

	scroll := objc.ID(class("NSScrollView")).Send(sel("new"))
	scroll.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 20, Y: 82}, Size: CGSize{Width: tabContentW - 40, Height: 342}})
	scroll.Send(sel("setHasVerticalScroller:"), true)
	scroll.Send(sel("setAutohidesScrollers:"), true)
	scroll.Send(sel("setDrawsBackground:"), false)
	scroll.Send(sel("setBorderType:"), int64(2))

	table := objc.ID(class("NSTableView")).Send(sel("new"))
	table.Send(sel("setRowHeight:"), 30.0)
	table.Send(sel("setIntercellSpacing:"), CGSize{})
	table.Send(sel("setHeaderView:"), objc.ID(0))
	table.Send(sel("setGridStyleMask:"), int64(0))
	table.Send(sel("setAllowsMultipleSelection:"), false)
	table.Send(sel("setAllowsEmptySelection:"), false)
	table.Send(sel("setSelectionHighlightStyle:"), int64(1))
	table.Send(sel("setFocusRingType:"), int64(1))
	table.Send(sel("setStyle:"), int64(3))
	table.Send(sel("setBackgroundColor:"), objc.ID(classNSColor).Send(sel("clearColor")))
	table.Send(sel("setDataSource:"), settingsActionsTarget)
	table.Send(sel("setDelegate:"), settingsActionsTarget)
	table.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.SettingsHoverTable)))

	column := objc.ID(class("NSTableColumn")).Send(sel("new"))
	column.Send(sel("setIdentifier:"), nsString("actions"))
	column.Send(sel("setResizingMask:"), int64(1))
	column.Send(sel("setWidth:"), float64(tabContentW-44))
	table.Send(sel("addTableColumn:"), column)
	column.Send(sel("release"))
	table.Send(sel("setColumnAutoresizingStyle:"), int64(1))

	scroll.Send(sel("setDocumentView:"), table)
	table.Send(sel("release"))
	content.Send(sel("addSubview:"), scroll)
	scroll.Send(sel("release"))
	v.actionsTable = table
	settingsActionsByTable[table] = v

	up := objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(i18n.T(i18n.SettingsMoveUp)), settingsActionsTarget, sel("actionUpPressed:"))
	up.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: tabContentW - 204, Y: 34}, Size: CGSize{Width: 90, Height: 30}})
	up.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.SettingsMoveUpA11y)))
	content.Send(sel("addSubview:"), up)
	settingsViews[up] = v

	down := objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(i18n.T(i18n.SettingsMoveDown)), settingsActionsTarget, sel("actionDownPressed:"))
	down.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: tabContentW - 104, Y: 34}, Size: CGSize{Width: 90, Height: 30}})
	down.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.SettingsMoveDownA11y)))
	content.Send(sel("addSubview:"), down)
	settingsViews[down] = v
}

func (v *SettingsView) applyActions(values SettingsValues) {
	enabled := make(map[string]struct{}, len(values.ItemActions))
	for _, key := range values.ItemActions {
		enabled[key] = struct{}{}
	}
	order := values.ItemActionOrder
	if len(order) == 0 {
		order = settingsActionFallbackOrder
	}
	actions := make([]settingsAction, 0, len(order))
	seen := map[string]struct{}{}
	for _, key := range order {
		title, ok := settingsActionTitle(key)
		if !ok {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		_, on := enabled[key]
		actions = append(actions, settingsAction{key: key, title: title, enabled: on})
	}
	for _, key := range settingsActionFallbackOrder {
		if _, ok := seen[key]; ok {
			continue
		}
		_, on := enabled[key]
		title, _ := settingsActionTitle(key)
		actions = append(actions, settingsAction{key: key, title: title, enabled: on})
	}
	v.actions = actions
	if v.actionsTable != 0 {
		selected := int(objc.Send[int64](v.actionsTable, sel("selectedRow")))
		v.actionsTable.Send(sel("reloadData"))
		if selected >= len(actions) {
			selected = len(actions) - 1
		}
		if selected < 0 && len(actions) > 0 {
			selected = 0
		}
		if selected >= 0 {
			v.selectAction(selected)
		}
	}
}

func (v *SettingsView) selectAction(row int) {
	if v.actionsTable == 0 || row < 0 || row >= len(v.actions) {
		return
	}
	indexes := objc.ID(class("NSIndexSet")).Send(sel("indexSetWithIndex:"), int64(row))
	v.actionsTable.Send(sel("selectRowIndexes:byExtendingSelection:"), indexes, false)
	v.actionsTable.Send(sel("scrollRowToVisible:"), int64(row))
}

func (v *SettingsView) emitActions() {
	enabled := make([]string, 0, len(v.actions))
	order := make([]string, 0, len(v.actions))
	for _, action := range v.actions {
		order = append(order, action.key)
		if action.enabled {
			enabled = append(enabled, action.key)
		}
	}
	v.cx.Emit("change", SettingsChange{
		Path: "actions.update", Value: SettingsItemActions{Enabled: enabled, Order: order},
	})
}

func (v *SettingsView) moveAction(delta int) {
	if v.actionsTable == 0 {
		return
	}
	row := int(objc.Send[int64](v.actionsTable, sel("selectedRow")))
	next := row + delta
	if row < 0 || row >= len(v.actions) || next < 0 || next >= len(v.actions) {
		return
	}
	v.actions[row], v.actions[next] = v.actions[next], v.actions[row]
	v.actionsTable.Send(sel("reloadData"))
	v.selectAction(next)
	v.emitActions()
}

func registerSettingsActionsTargets() {
	var err error
	settingsActionRowClass, err = objc.RegisterClass("EcoPasteSettingsActionRow", class("NSTableCellView"), nil, nil,
		[]objc.MethodDef{{Cmd: sel("setFrameSize:"), Fn: func(self objc.ID, _ objc.SEL, size CGSize) {
			sendSuper[objc.ID](self, settingsActionRowClass, sel("setFrameSize:"), size)
			layoutSettingsActionRow(self, size)
		}}})
	if err != nil {
		panic(err)
	}
	target, err := objc.RegisterClass("EcoPasteSettingsActionsTarget", class("NSObject"), nil, nil, []objc.MethodDef{
		{Cmd: sel("numberOfRowsInTableView:"), Fn: func(_ objc.ID, _ objc.SEL, table objc.ID) int64 {
			if v := settingsActionsByTable[table]; v != nil {
				return int64(len(v.actions))
			}
			return 0
		}},
		{Cmd: sel("tableView:viewForTableColumn:row:"), Fn: func(_ objc.ID, _ objc.SEL, table, _ objc.ID, row int64) objc.ID {
			v := settingsActionsByTable[table]
			if v == nil || row < 0 || int(row) >= len(v.actions) {
				return 0
			}
			cell := table.Send(sel("makeViewWithIdentifier:owner:"), nsString(settingsActionRowIdentifier), table)
			if cell == 0 {
				cell = newSettingsActionRow()
			}
			fillSettingsActionRow(v, cell, int(row), v.actions[row])
			return cell
		}},
		{Cmd: sel("actionToggle:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			v := settingsActionControls[sender]
			if v == nil {
				return
			}
			row := int(objc.Send[int64](sender, sel("tag"))) - settingsActionCheckTagBase
			if row < 0 || row >= len(v.actions) {
				return
			}
			v.actions[row].enabled = objc.Send[int64](sender, sel("state")) == 1
			v.emitActions()
		}},
		{Cmd: sel("actionUpPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			if v := settingsViews[sender]; v != nil {
				v.moveAction(-1)
			}
		}},
		{Cmd: sel("actionDownPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			if v := settingsViews[sender]; v != nil {
				v.moveAction(1)
			}
		}},
	})
	if err != nil {
		panic(err)
	}
	settingsActionsTarget = objc.ID(target).Send(sel("new"))
}

func newSettingsActionRow() objc.ID {
	cell := objc.ID(settingsActionRowClass).Send(sel("new"))
	cell.Send(sel("setIdentifier:"), nsString(settingsActionRowIdentifier))

	checkbox := objc.ID(class("NSButton")).Send(sel("new"))
	checkbox.Send(sel("setButtonType:"), int64(3))
	checkbox.Send(sel("setTitle:"), nsString(""))
	checkbox.Send(sel("setTarget:"), settingsActionsTarget)
	checkbox.Send(sel("setAction:"), sel("actionToggle:"))
	checkbox.Send(sel("setFocusRingType:"), int64(1))
	cell.Send(sel("addSubview:"), checkbox)
	checkbox.Send(sel("release"))

	title := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	title.Send(sel("setFont:"), systemFont(13))
	title.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("labelColor")))
	title.Send(sel("setLineBreakMode:"), int64(4))
	title.Send(sel("setMaximumNumberOfLines:"), int64(1))
	title.Send(sel("setTag:"), int64(1))
	cell.Send(sel("addSubview:"), title)

	return cell.Send(sel("autorelease"))
}

func fillSettingsActionRow(v *SettingsView, cell objc.ID, row int, action settingsAction) {
	subviews := objc.Send[objc.ID](cell, sel("subviews"))
	if subviews != 0 && objc.Send[uint](subviews, sel("count")) > 0 {
		checkbox := objc.Send[objc.ID](subviews, sel("objectAtIndex:"), uint(0))
		checkbox.Send(sel("setTag:"), int64(settingsActionCheckTagBase+row))
		checkbox.Send(sel("setState:"), int64(boolInt(action.enabled)))
		checkbox.Send(sel("setAccessibilityLabel:"), nsString(i18n.Tf(i18n.SettingsEnableAction, action.title)))
		settingsActionControls[checkbox] = v
	}
	if title := cell.Send(sel("viewWithTag:"), int64(1)); title != 0 {
		title.Send(sel("setStringValue:"), nsString(action.title))
	}
}

func layoutSettingsActionRow(cell objc.ID, size CGSize) {
	subviews := objc.Send[objc.ID](cell, sel("subviews"))
	if subviews != 0 && objc.Send[uint](subviews, sel("count")) > 0 {
		checkbox := objc.Send[objc.ID](subviews, sel("objectAtIndex:"), uint(0))
		checkbox.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 8, Y: 6}, Size: CGSize{Width: 18, Height: 18}})
	}
	if title := cell.Send(sel("viewWithTag:"), int64(1)); title != 0 {
		title.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 36, Y: 5}, Size: CGSize{Width: size.Width - 44, Height: 18}})
	}
}

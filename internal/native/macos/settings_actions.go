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

// buildSettingsActions resolves the persisted selection and order into the
// rows the table renders, localizing their titles.
func buildSettingsActions(values SettingsValues) []settingsAction {
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
	return actions
}

// settingsActionsSameShape reports whether two rows lists share the same keys
// in the same order, ignoring their checked state and title.
func settingsActionsSameShape(current, next []settingsAction) bool {
	if len(current) != len(next) {
		return false
	}
	for i := range current {
		if current[i].key != next[i].key {
			return false
		}
	}
	return true
}

// settingsActionChangedRows returns the rows whose rendered content changed.
func settingsActionChangedRows(current, next []settingsAction) []int {
	rows := make([]int, 0, len(current))
	for i := 0; i < len(current) && i < len(next); i++ {
		if current[i].enabled != next[i].enabled || current[i].title != next[i].title {
			rows = append(rows, i)
		}
	}
	return rows
}

func (v *SettingsView) selectedActionKey() string {
	if v.actionsTable == 0 {
		return ""
	}
	row := int(objc.Send[int64](v.actionsTable, sel("selectedRow")))
	if row < 0 || row >= len(v.actions) {
		return ""
	}
	return v.actions[row].key
}

// applyActions refreshes the hover-action rows. It only redraws the rows whose
// checked state or title changed, and preserves the list's scroll position and
// selection when the row order itself changes.
func (v *SettingsView) applyActions(values SettingsValues) {
	actions := buildSettingsActions(values)
	if settingsActionsSameShape(v.actions, actions) {
		changed := settingsActionChangedRows(v.actions, actions)
		v.actions = actions
		if len(changed) > 0 {
			reloadSettingsTableRows(v.actionsTable, changed)
		}
		return
	}
	selectedKey := v.selectedActionKey()
	v.actions = actions
	if v.actionsTable != 0 {
		v.actionsTable.Send(sel("reloadData"))
		sizeSettingsTable(v.actionsTable, settingsContentWidth-40, 30, len(actions))
		row := stringIndexKeys(v.actions, selectedKey)
		if row < 0 && len(actions) > 0 {
			row = 0
		}
		selectSettingsRowNoScroll(v.actionsTable, row)
		v.updateMoveButtons()
	}
}

// stringIndexKeys maps a key to its row in an action list.
func stringIndexKeys(actions []settingsAction, key string) int {
	for i, action := range actions {
		if action.key == key {
			return i
		}
	}
	return -1
}

func (v *SettingsView) selectAction(row int) {
	if v.actionsTable == 0 || row < 0 || row >= len(v.actions) {
		return
	}
	selectSettingsRow(v.actionsTable, row)
	v.updateMoveButtons()
}

func (v *SettingsView) updateMoveButtons() {
	updateSettingsMoveButtons(v.actionsTable, v.actionUp, v.actionDown, len(v.actions))
}

func orderMoveEnabled(selected, total int) (bool, bool) {
	if selected < 0 || selected >= total {
		return false, false
	}
	return selected > 0, selected < total-1
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
	next, ok := moveSelectedRow(v.actionsTable, v.actions, delta)
	if !ok {
		return
	}
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
		{Cmd: sel("tableViewSelectionDidChange:"), Fn: func(_ objc.ID, _ objc.SEL, notification objc.ID) {
			if notification == 0 {
				return
			}
			table := notification.Send(sel("object"))
			if v := settingsActionsByTable[table]; v != nil {
				v.updateMoveButtons()
			}
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

	checkbox := newSettingsCheckbox(settingsActionsTarget, "actionToggle:")
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
		checkbox.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 8, Y: 7}, Size: CGSize{Width: 16, Height: 16}})
	}
	if title := cell.Send(sel("viewWithTag:"), int64(1)); title != 0 {
		title.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 36, Y: 5}, Size: CGSize{Width: size.Width - 44, Height: 18}})
	}
}

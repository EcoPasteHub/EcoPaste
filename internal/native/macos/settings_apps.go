//go:build darwin

package macos

import (
	"sync"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

const (
	settingsAppRowIdentifier = "ecopaste.settings-app-row"
	settingsAppRowHeight     = 44.0
	settingsAppCheckTagBase  = 1000
	settingsAppIconTag       = 201
	settingsAppNameTag       = 202
	settingsAppIDTag         = 203
)

var (
	registerSettingsAppsOnce sync.Once
	settingsAppsTarget       objc.ID
	settingsAppsRowClass     objc.Class
	settingsAppsByTable      = map[objc.ID]*SettingsView{}
	settingsAppControls      = map[objc.ID]*SettingsView{}
)

// buildApps adds the source application filter tab. Each row toggles one
// app in clipboard.filters.excludedAppIds.
func (v *SettingsView) buildApps() {
	registerSettingsAppsOnce.Do(registerSettingsAppsTargets)
	content := v.newTab("apps", settingsTabTitle("apps"))
	v.addSearchEntry(content, i18n.T(i18n.SettingsSourceApps), "clipboard.filters.excludedAppIds",
		i18n.T(i18n.SettingsAppsKeywords), "source apps filter exclude privacy",
		CGRect{Origin: CGPoint{X: 12, Y: 24}, Size: CGSize{Width: tabContentW - 24, Height: 420}})

	v.addSectionLabel(content, 438, i18n.T(i18n.SettingsAppsSection))
	intro := objc.ID(classNSTextField).Send(sel("wrappingLabelWithString:"),
		nsString(i18n.T(i18n.SettingsAppsIntro)))
	intro.Send(sel("setFont:"), systemFont(11))
	intro.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	intro.Send(sel("setPreferredMaxLayoutWidth:"), tabContentW-48)
	intro.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: 408}, Size: CGSize{Width: tabContentW - 48, Height: 28}})
	content.Send(sel("addSubview:"), intro)

	scroll := objc.ID(class("NSScrollView")).Send(sel("new"))
	scroll.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 20, Y: 84}, Size: CGSize{Width: tabContentW - 40, Height: 310}})
	scroll.Send(sel("setHasVerticalScroller:"), true)
	scroll.Send(sel("setAutohidesScrollers:"), true)
	scroll.Send(sel("setDrawsBackground:"), false)
	scroll.Send(sel("setBorderType:"), int64(2))

	table := objc.ID(class("NSTableView")).Send(sel("new"))
	table.Send(sel("setRowHeight:"), settingsAppRowHeight)
	table.Send(sel("setIntercellSpacing:"), CGSize{})
	table.Send(sel("setHeaderView:"), objc.ID(0))
	table.Send(sel("setGridStyleMask:"), int64(0))
	table.Send(sel("setAllowsMultipleSelection:"), false)
	table.Send(sel("setAllowsEmptySelection:"), true)
	table.Send(sel("setSelectionHighlightStyle:"), int64(-1))
	table.Send(sel("setFocusRingType:"), int64(1))
	table.Send(sel("setStyle:"), int64(3))
	table.Send(sel("setBackgroundColor:"), objc.ID(classNSColor).Send(sel("clearColor")))
	table.Send(sel("setDataSource:"), settingsAppsTarget)
	table.Send(sel("setDelegate:"), settingsAppsTarget)
	table.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.SettingsSourceApps)))

	column := objc.ID(class("NSTableColumn")).Send(sel("new"))
	column.Send(sel("setIdentifier:"), nsString("apps"))
	column.Send(sel("setResizingMask:"), int64(1))
	column.Send(sel("setWidth:"), float64(tabContentW-44))
	table.Send(sel("addTableColumn:"), column)
	column.Send(sel("release"))
	table.Send(sel("setColumnAutoresizingStyle:"), int64(1))

	scroll.Send(sel("setDocumentView:"), table)
	table.Send(sel("release"))
	content.Send(sel("addSubview:"), scroll)
	scroll.Send(sel("release"))
	v.appsTable = table
	settingsAppsByTable[table] = v

	empty := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(i18n.T(i18n.SettingsAppsEmpty)))
	empty.Send(sel("setFont:"), systemFont(12))
	empty.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	empty.Send(sel("setAlignment:"), int64(2))
	empty.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: 229}, Size: CGSize{Width: tabContentW - 48, Height: 20}})
	empty.Send(sel("setHidden:"), true)
	content.Send(sel("addSubview:"), empty)
	v.appsEmpty = empty

	add := objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(i18n.T(i18n.SettingsAddApp)), settingsTarget, sel("addAppPressed:"))
	add.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: tabContentW - 134, Y: 34}, Size: CGSize{Width: 110, Height: 30}})
	add.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.SettingsAddAppA11y)))
	content.Send(sel("addSubview:"), add)
	settingsViews[add] = v

	refresh := objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(i18n.T(i18n.SettingsRefresh)), settingsTarget, sel("refreshAppsPressed:"))
	refresh.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: tabContentW - 214, Y: 34}, Size: CGSize{Width: 70, Height: 30}})
	refresh.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.SettingsRefreshA11y)))
	content.Send(sel("addSubview:"), refresh)
	settingsViews[refresh] = v
}

func (v *SettingsView) applyApps(values SettingsValues) {
	v.apps = values.Apps
	shown := make(map[string]struct{}, len(values.Apps))
	for _, app := range values.Apps {
		shown[app.ID] = struct{}{}
	}
	v.extraExcluded = v.extraExcluded[:0]
	for _, id := range values.ExcludedAppIDs {
		if _, ok := shown[id]; !ok {
			v.extraExcluded = append(v.extraExcluded, id)
		}
	}
	if v.appsTable != 0 {
		v.appsTable.Send(sel("reloadData"))
		sizeSettingsTable(v.appsTable, tabContentW-40, settingsAppRowHeight, len(values.Apps))
	}
	if v.appsEmpty != 0 {
		v.appsEmpty.Send(sel("setHidden:"), len(v.apps) > 0)
	}
}

func (v *SettingsView) emitExcludedApps() {
	ids := make([]string, 0, len(v.apps)+len(v.extraExcluded))
	seen := map[string]struct{}{}
	for _, app := range v.apps {
		if app.Excluded {
			ids = append(ids, app.ID)
			seen[app.ID] = struct{}{}
		}
	}
	for _, id := range v.extraExcluded {
		if _, ok := seen[id]; !ok {
			ids = append(ids, id)
			seen[id] = struct{}{}
		}
	}
	v.cx.Emit("change", SettingsChange{Path: "filters.excludedAppIds", Value: ids})
}

func registerSettingsAppsTargets() {
	var err error
	settingsAppsRowClass, err = objc.RegisterClass("EcoPasteSettingsAppRow", class("NSTableCellView"), nil, nil,
		[]objc.MethodDef{{Cmd: sel("setFrameSize:"), Fn: func(self objc.ID, _ objc.SEL, size CGSize) {
			sendSuper[objc.ID](self, settingsAppsRowClass, sel("setFrameSize:"), size)
			layoutSettingsAppRow(self, size)
		}}})
	if err != nil {
		panic(err)
	}
	target, err := objc.RegisterClass("EcoPasteSettingsAppsTarget", class("NSObject"), nil, nil, []objc.MethodDef{
		{Cmd: sel("numberOfRowsInTableView:"), Fn: func(_ objc.ID, _ objc.SEL, table objc.ID) int64 {
			if v := settingsAppsByTable[table]; v != nil {
				return int64(len(v.apps))
			}
			return 0
		}},
		{Cmd: sel("tableView:viewForTableColumn:row:"), Fn: func(_ objc.ID, _ objc.SEL, table, _ objc.ID, row int64) objc.ID {
			v := settingsAppsByTable[table]
			if v == nil || row < 0 || int(row) >= len(v.apps) {
				return 0
			}
			cell := table.Send(sel("makeViewWithIdentifier:owner:"), nsString(settingsAppRowIdentifier), table)
			if cell == 0 {
				cell = newSettingsAppRow()
			}
			fillSettingsAppRow(v, cell, int(row), v.apps[row])
			return cell
		}},
		{Cmd: sel("appToggle:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			v := settingsAppControls[sender]
			if v == nil {
				return
			}
			row := int(objc.Send[int64](sender, sel("tag"))) - settingsAppCheckTagBase
			if row < 0 || row >= len(v.apps) {
				return
			}
			v.apps[row].Excluded = objc.Send[int64](sender, sel("state")) == 1
			v.emitExcludedApps()
		}},
	})
	if err != nil {
		panic(err)
	}
	settingsAppsTarget = objc.ID(target).Send(sel("new"))
}

func newSettingsAppRow() objc.ID {
	cell := objc.ID(settingsAppsRowClass).Send(sel("new"))
	cell.Send(sel("setIdentifier:"), nsString(settingsAppRowIdentifier))

	checkbox := objc.ID(class("NSButton")).Send(sel("new"))
	checkbox.Send(sel("setButtonType:"), int64(3))
	checkbox.Send(sel("setTitle:"), nsString(""))
	checkbox.Send(sel("setTarget:"), settingsAppsTarget)
	checkbox.Send(sel("setAction:"), sel("appToggle:"))
	checkbox.Send(sel("setFocusRingType:"), int64(1))
	cell.Send(sel("addSubview:"), checkbox)
	checkbox.Send(sel("release"))

	icon := objc.ID(class("NSImageView")).Send(sel("new"))
	icon.Send(sel("setTag:"), int64(settingsAppIconTag))
	icon.Send(sel("setImageScaling:"), int64(3))
	icon.Send(sel("setAccessibilityElement:"), false)
	cell.Send(sel("addSubview:"), icon)
	icon.Send(sel("release"))

	name := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	name.Send(sel("setTag:"), int64(settingsAppNameTag))
	name.Send(sel("setFont:"), systemFont(13))
	name.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("labelColor")))
	name.Send(sel("setLineBreakMode:"), int64(4))
	name.Send(sel("setMaximumNumberOfLines:"), int64(1))
	cell.Send(sel("addSubview:"), name)

	detail := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	detail.Send(sel("setTag:"), int64(settingsAppIDTag))
	detail.Send(sel("setFont:"), systemFont(10))
	detail.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	detail.Send(sel("setLineBreakMode:"), int64(4))
	detail.Send(sel("setMaximumNumberOfLines:"), int64(1))
	cell.Send(sel("addSubview:"), detail)

	return cell.Send(sel("autorelease"))
}

func fillSettingsAppRow(v *SettingsView, cell objc.ID, row int, app SettingsApp) {
	checkbox := firstSettingsAppCheckbox(cell)
	if checkbox != 0 {
		checkbox.Send(sel("setTag:"), int64(settingsAppCheckTagBase+row))
		checkbox.Send(sel("setState:"), int64(boolInt(app.Excluded)))
		checkbox.Send(sel("setAccessibilityLabel:"), nsString(i18n.Tf(i18n.SettingsIgnoreApp, app.Name)))
		settingsAppControls[checkbox] = v
	}
	icon := cell.Send(sel("viewWithTag:"), int64(settingsAppIconTag))
	if icon != 0 {
		image := objc.ID(0)
		if app.IconPath != "" {
			image = loadImage(app.IconPath)
		}
		icon.Send(sel("setImage:"), image)
	}
	name := cell.Send(sel("viewWithTag:"), int64(settingsAppNameTag))
	if name != 0 {
		name.Send(sel("setStringValue:"), nsString(app.Name))
	}
	detail := cell.Send(sel("viewWithTag:"), int64(settingsAppIDTag))
	if detail != 0 {
		detail.Send(sel("setStringValue:"), nsString(app.ID))
	}
}

func firstSettingsAppCheckbox(cell objc.ID) objc.ID {
	subviews := objc.Send[objc.ID](cell, sel("subviews"))
	if subviews == 0 || objc.Send[uint](subviews, sel("count")) == 0 {
		return 0
	}
	return objc.Send[objc.ID](subviews, sel("objectAtIndex:"), uint(0))
}

func layoutSettingsAppRow(cell objc.ID, size CGSize) {
	if checkbox := firstSettingsAppCheckbox(cell); checkbox != 0 {
		checkbox.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 8, Y: 12}, Size: CGSize{Width: 18, Height: 18}})
	}
	if icon := cell.Send(sel("viewWithTag:"), int64(settingsAppIconTag)); icon != 0 {
		icon.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 36, Y: 10}, Size: CGSize{Width: 24, Height: 24}})
	}
	if name := cell.Send(sel("viewWithTag:"), int64(settingsAppNameTag)); name != 0 {
		name.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 68, Y: 13}, Size: CGSize{Width: size.Width - 76, Height: 18}})
	}
	if detail := cell.Send(sel("viewWithTag:"), int64(settingsAppIDTag)); detail != 0 {
		detail.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 68, Y: 30}, Size: CGSize{Width: size.Width - 76, Height: 14}})
	}
}

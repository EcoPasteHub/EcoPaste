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

func (v *SettingsView) buildAppsPage() {
	registerSettingsAppsOnce.Do(registerSettingsAppsTargets)
	document := v.newPage("apps")
	top := 18.0
	v.addSectionLabel(document, top, i18n.SettingsAppsSection)
	top += 24
	top = v.addPageParagraph(document, top, 32, i18n.SettingsAppsIntro)

	const listHeight = 300.0
	card := CGRect{
		Origin: CGPoint{X: settingsContentX, Y: top},
		Size:   CGSize{Width: settingsContentWidth, Height: listHeight},
	}
	addSettingsListCard(document, card)

	scroll := objc.ID(class("NSScrollView")).Send(sel("new"))
	scroll.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsContentX + 1, Y: top + 1},
		Size:   CGSize{Width: settingsContentWidth - 2, Height: listHeight - 2},
	})
	scroll.Send(sel("setHasVerticalScroller:"), true)
	scroll.Send(sel("setAutohidesScrollers:"), true)
	scroll.Send(sel("setDrawsBackground:"), false)
	scroll.Send(sel("setBorderType:"), int64(0))

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
	v.bindText(func() {
		table.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.SettingsSourceApps)))
	})

	column := objc.ID(class("NSTableColumn")).Send(sel("new"))
	column.Send(sel("setIdentifier:"), nsString("apps"))
	column.Send(sel("setResizingMask:"), int64(1))
	column.Send(sel("setWidth:"), settingsContentWidth-4)
	table.Send(sel("addTableColumn:"), column)
	column.Send(sel("release"))
	table.Send(sel("setColumnAutoresizingStyle:"), int64(1))

	scroll.Send(sel("setDocumentView:"), table)
	table.Send(sel("release"))
	document.Send(sel("addSubview:"), scroll)
	scroll.Send(sel("release"))
	v.appsTable = table
	settingsAppsByTable[table] = v

	empty := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(i18n.T(i18n.SettingsAppsEmpty)))
	empty.Send(sel("setFont:"), systemFont(12))
	empty.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	empty.Send(sel("setAlignment:"), int64(2))
	empty.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsContentX + 24, Y: top + listHeight/2 - 10},
		Size:   CGSize{Width: settingsContentWidth - 48, Height: 20},
	})
	empty.Send(sel("setHidden:"), true)
	document.Send(sel("addSubview:"), empty)
	v.appsEmpty = empty
	v.bindString(empty, i18n.SettingsAppsEmpty)

	top += listHeight + 16
	add := objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(i18n.T(i18n.SettingsAddApp)), settingsTarget, sel("addAppPressed:"))
	add.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: settingsContentX + settingsContentWidth - 110, Y: top},
		Size: CGSize{Width: 110, Height: 30}})
	add.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.SettingsAddAppA11y)))
	document.Send(sel("addSubview:"), add)
	settingsViews[add] = v
	v.bindText(func() {
		add.Send(sel("setTitle:"), nsString(i18n.T(i18n.SettingsAddApp)))
		add.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.SettingsAddAppA11y)))
	})

	refresh := objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(i18n.T(i18n.SettingsRefresh)), settingsTarget, sel("refreshAppsPressed:"))
	refresh.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: settingsContentX + settingsContentWidth - 192, Y: top},
		Size: CGSize{Width: 74, Height: 30}})
	refresh.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.SettingsRefreshA11y)))
	document.Send(sel("addSubview:"), refresh)
	settingsViews[refresh] = v
	v.bindText(func() {
		refresh.Send(sel("setTitle:"), nsString(i18n.T(i18n.SettingsRefresh)))
		refresh.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.SettingsRefreshA11y)))
	})

	v.finishPage(document, top+30)
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
		sizeSettingsTable(v.appsTable, settingsContentWidth-40, settingsAppRowHeight, len(values.Apps))
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

	checkbox := newSettingsCheckbox(settingsAppsTarget, "appToggle:")
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
		checkbox.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 8, Y: 14}, Size: CGSize{Width: 16, Height: 16}})
	}
	if icon := cell.Send(sel("viewWithTag:"), int64(settingsAppIconTag)); icon != 0 {
		icon.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 36, Y: 10}, Size: CGSize{Width: 24, Height: 24}})
	}
	if name := cell.Send(sel("viewWithTag:"), int64(settingsAppNameTag)); name != 0 {
		name.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 68, Y: 22}, Size: CGSize{Width: size.Width - 76, Height: 18}})
	}
	if detail := cell.Send(sel("viewWithTag:"), int64(settingsAppIDTag)); detail != 0 {
		detail.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 68, Y: 6}, Size: CGSize{Width: size.Width - 76, Height: 14}})
	}
}

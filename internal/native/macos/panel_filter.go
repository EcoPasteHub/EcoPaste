package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

// showFilterMenu opens the toolbar filter menu: category, custom group and
// group editing, in one native menu.
func (p *Panel) showFilterMenu(anchor objc.ID) {
	menu := objc.ID(class("NSMenu")).Send(sel("new"))
	defer menu.Send(sel("release"))
	menu.Send(sel("setAutoenablesItems:"), false)
	for index, label := range panelNavLabels() {
		item := addFilterAction(menu, p, label, "filterMenuPressed:", index)
		if panelNavFilters[index] == p.filterValue {
			item.Send(sel("setState:"), int64(1))
		}
	}
	menu.Send(sel("addItem:"), objc.ID(class("NSMenuItem")).Send(sel("separatorItem")))
	allGroups := addFilterAction(menu, p, i18n.T(i18n.PanelAllGroups), "filterMenuPressed:", filterGroupTag)
	if p.groupID == "" {
		allGroups.Send(sel("setState:"), int64(1))
	}
	for index, group := range p.groups {
		item := addFilterAction(menu, p, group.Name, "filterMenuPressed:", filterGroupTag+1+index)
		if group.ID == p.groupID {
			item.Send(sel("setState:"), int64(1))
		}
	}
	menu.Send(sel("addItem:"), objc.ID(class("NSMenuItem")).Send(sel("separatorItem")))
	addMenuAction(menu, p, i18n.T(i18n.PanelNewGroup), "menuNewGroup:", true)
	addMenuAction(menu, p, i18n.T(i18n.PanelRenameGroup), "menuRenameGroup:", p.groupID != "")
	addMenuAction(menu, p, i18n.T(i18n.PanelDeleteGroup), "menuDeleteGroup:", p.groupID != "")
	menu.Send(sel("popUpMenuPositioningItem:atLocation:inView:"), objc.ID(0),
		CGPoint{X: 0, Y: bounds(anchor).Size.Height}, anchor)
}

const filterGroupTag = 100

// applyFilterMenuItem resolves the tag of a filter menu item into a filter
// or group selection.
func (p *Panel) applyFilterMenuItem(tag int) {
	switch {
	case tag >= 0 && tag < len(panelNavFilters):
		p.cx.Emit("filter", panelNavFilters[tag])
	case tag == filterGroupTag:
		p.cx.Emit("group", "")
	case tag > filterGroupTag && tag-filterGroupTag-1 < len(p.groups):
		p.cx.Emit("group", p.groups[tag-filterGroupTag-1].ID)
	}
}

func addFilterAction(menu objc.ID, p *Panel, title, action string, tag int) objc.ID {
	item := objc.ID(class("NSMenuItem")).Send(sel("alloc")).
		Send(sel("initWithTitle:action:keyEquivalent:"), nsString(title), sel(action), nsString(""))
	item.Send(sel("setTarget:"), searchTarget)
	item.Send(sel("setRepresentedObject:"), p.table)
	item.Send(sel("setTag:"), int64(tag))
	menu.Send(sel("addItem:"), item)
	item.Send(sel("release"))
	return item
}

func addMenuAction(menu objc.ID, p *Panel, title, action string, enabled bool) {
	item := objc.ID(class("NSMenuItem")).Send(sel("alloc")).
		Send(sel("initWithTitle:action:keyEquivalent:"), nsString(title), sel(action), nsString(""))
	item.Send(sel("setTarget:"), searchTarget)
	item.Send(sel("setRepresentedObject:"), p.table)
	item.Send(sel("setEnabled:"), enabled)
	menu.Send(sel("addItem:"), item)
	item.Send(sel("release"))
}

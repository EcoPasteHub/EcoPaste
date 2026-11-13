//go:build darwin

package macos

import (
	"strings"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

func (p *Panel) populateMenu(menu objc.ID, row HistoryRow) {
	menu.Send(sel("removeAllItems"))
	addMenuItem(menu, p.table, i18n.T(i18n.MenuPaste), "menuPaste:", "arrow.down.doc", "")
	addMenuItem(menu, p.table, i18n.T(i18n.MenuCopy), "menuCopy:", "doc.on.doc", "c")
	if row.Kind == "text" {
		plainPaste := addMenuItem(menu, p.table, i18n.T(i18n.MenuPastePlain), "menuPastePlain:", "textformat", "v")
		setMenuModifiers(plainPaste, commandModifier|alternateModifier)
		plainCopy := addMenuItem(menu, p.table, i18n.T(i18n.MenuCopyPlain), "menuCopyPlain:", "doc.plaintext", "c")
		setMenuModifiers(plainCopy, commandModifier|alternateModifier)
	}
	if row.Kind == "files" {
		pathPaste := addMenuItem(menu, p.table, i18n.T(i18n.MenuPastePath), "menuPastePath:", "point.topleft.down.to.point.bottomright.curvepath", "v")
		setMenuModifiers(pathPaste, commandModifier|alternateModifier)
	}
	switch row.SubKind {
	case "url":
		addMenuItem(menu, p.table, i18n.T(i18n.MenuOpenLink), "menuOpen:", "arrow.up.right.square", "")
	case "email":
		addMenuItem(menu, p.table, i18n.T(i18n.MenuSendEmail), "menuOpen:", "envelope", "")
	}
	if row.SubKind == "path" || row.Kind == "files" {
		addMenuItem(menu, p.table, i18n.T(i18n.MenuReveal), "menuReveal:", "folder", "")
	}
	if row.Kind == "image" {
		addMenuItem(menu, p.table, i18n.T(i18n.MenuSaveImage), "menuSaveImage:", "square.and.arrow.down", "")
	}
	note := i18n.T(i18n.MenuAddNote)
	if strings.TrimSpace(row.Note) != "" {
		note = i18n.T(i18n.MenuEditNote)
	}
	addMenuItem(menu, p.table, note, "menuNote:", "note.text", "")
	menu.Send(sel("addItem:"), objc.ID(class("NSMenuItem")).Send(sel("separatorItem")))

	favorite := i18n.T(i18n.MenuFavorite)
	if row.Favorite {
		favorite = i18n.T(i18n.MenuUnfavorite)
	}
	addMenuItem(menu, p.table, favorite, "menuFavorite:", "star", "")
	pin := i18n.T(i18n.MenuPin)
	if row.Pinned {
		pin = i18n.T(i18n.MenuUnpin)
	}
	addMenuItem(menu, p.table, pin, "menuPin:", "pin", "")
	moveItem := objc.ID(class("NSMenuItem")).Send(sel("alloc")).
		Send(sel("initWithTitle:action:keyEquivalent:"), nsString(i18n.T(i18n.MenuMoveToGroup)), objc.SEL(0), nsString(""))
	moveItem.Send(sel("setImage:"), systemSymbol("folder"))
	submenu := objc.ID(class("NSMenu")).Send(sel("new"))
	addMoveGroupItem(submenu, p, i18n.T(i18n.MenuUngrouped), "", row.GroupID == "")
	if len(p.groups) > 0 {
		submenu.Send(sel("addItem:"), objc.ID(class("NSMenuItem")).Send(sel("separatorItem")))
	}
	for _, group := range p.groups {
		addMoveGroupItem(submenu, p, group.Name, group.ID, row.GroupID == group.ID)
	}
	moveItem.Send(sel("setSubmenu:"), submenu)
	submenu.Send(sel("release"))
	menu.Send(sel("addItem:"), moveItem)
	moveItem.Send(sel("release"))
	if row.CanDelete {
		menu.Send(sel("addItem:"), objc.ID(class("NSMenuItem")).Send(sel("separatorItem")))
		addMenuItem(menu, p.table, i18n.T(i18n.CommonDelete), "menuDelete:", "trash", "")
	}
}

func addMoveGroupItem(menu objc.ID, p *Panel, title, groupID string, selected bool) {
	item := objc.ID(class("NSMenuItem")).Send(sel("alloc")).
		Send(sel("initWithTitle:action:keyEquivalent:"), nsString(title), sel("menuMoveGroup:"), nsString(""))
	item.Send(sel("setTarget:"), historyTarget)
	item.Send(sel("setRepresentedObject:"), p.table)
	tag := int64(0)
	if groupID != "" {
		for index, group := range p.groups {
			if group.ID == groupID {
				tag = int64(index + 1)
				break
			}
		}
	}
	item.Send(sel("setTag:"), tag)
	if selected {
		item.Send(sel("setState:"), int64(1))
	}
	menu.Send(sel("addItem:"), item)
	item.Send(sel("release"))
}

func addMenuItem(menu, table objc.ID, title, action, icon, key string) objc.ID {
	item := objc.ID(class("NSMenuItem")).Send(sel("alloc")).Send(sel("initWithTitle:action:keyEquivalent:"),
		nsString(title), sel(action), nsString(key))
	item.Send(sel("setTarget:"), historyTarget)
	item.Send(sel("setRepresentedObject:"), table)
	item.Send(sel("setImage:"), systemSymbol(icon))
	menu.Send(sel("addItem:"), item)
	item.Send(sel("release"))
	return item
}

func setMenuModifiers(item objc.ID, modifiers uint64) {
	item.Send(sel("setKeyEquivalentModifierMask:"), modifiers)
}

func menuPanel(item objc.ID) *Panel {
	table := objc.Send[objc.ID](item, sel("representedObject"))
	return panelsByTable[table]
}

func menuAction(event string) func(objc.ID, objc.SEL, objc.ID) {
	return func(_ objc.ID, _ objc.SEL, sender objc.ID) {
		if p := menuPanel(sender); p != nil {
			p.cx.Emit(event, p.selection)
		}
	}
}

func confirmDelete() bool {
	alert := objc.ID(class("NSAlert")).Send(sel("new"))
	defer alert.Send(sel("release"))
	alert.Send(sel("setMessageText:"), nsString(i18n.T(i18n.DeleteItemTitle)))
	alert.Send(sel("setInformativeText:"), nsString(i18n.T(i18n.DeleteItemDetail)))
	alert.Send(sel("addButtonWithTitle:"), nsString(i18n.T(i18n.CommonDelete)))
	alert.Send(sel("addButtonWithTitle:"), nsString(i18n.T(i18n.CommonCancel)))
	return alert.Send(sel("runModal")) == 1000
}

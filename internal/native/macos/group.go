//go:build darwin

package macos

import (
	"strings"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

// EditGroupName asks for a custom group name.
func EditGroupName(current string) (string, bool) {
	var result string
	var saved bool
	onMain(func() {
		suppressPanelBlur.Store(true)
		defer suppressPanelBlur.Store(false)

		app := objc.ID(class("NSApplication")).Send(sel("sharedApplication"))
		app.Send(sel("activateIgnoringOtherApps:"), true)

		title := i18n.T(i18n.GroupNewTitle)
		if strings.TrimSpace(current) != "" {
			title = i18n.T(i18n.GroupRenameTitle)
		}
		alert := objc.ID(class("NSAlert")).Send(sel("new"))
		defer alert.Send(sel("release"))
		alert.Send(sel("setMessageText:"), nsString(title))
		alert.Send(sel("setInformativeText:"), nsString(i18n.T(i18n.GroupNameTooLong)))
		alert.Send(sel("addButtonWithTitle:"), nsString(i18n.T(i18n.CommonSave)))
		alert.Send(sel("addButtonWithTitle:"), nsString(i18n.T(i18n.CommonCancel)))

		field := objc.ID(classNSTextField).Send(sel("alloc")).
			Send(sel("initWithFrame:"), CGRect{Size: CGSize{Width: 320, Height: 24}})
		field.Send(sel("setStringValue:"), nsString(current))
		field.Send(sel("setPlaceholderString:"), nsString(i18n.T(i18n.GroupNamePlaceholder)))
		field.Send(sel("setFont:"), systemFont(13))
		field.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.GroupNameLabel)))
		alert.Send(sel("setAccessoryView:"), field)

		window := objc.Send[objc.ID](alert, sel("window"))
		window.Send(sel("setLevel:"), int64(1000))
		window.Send(sel("makeKeyAndOrderFront:"), objc.ID(0))
		window.Send(sel("makeFirstResponder:"), field)
		if alert.Send(sel("runModal")) == 1000 {
			result = goString(field.Send(sel("stringValue")))
			saved = true
		}
		field.Send(sel("release"))
	})
	return result, saved
}

// ConfirmDeleteGroup asks before deleting a group. Clipboard items in
// the group remain in history.
func ConfirmDeleteGroup(name string) bool {
	var confirmed bool
	onMain(func() {
		suppressPanelBlur.Store(true)
		defer suppressPanelBlur.Store(false)

		app := objc.ID(class("NSApplication")).Send(sel("sharedApplication"))
		app.Send(sel("activateIgnoringOtherApps:"), true)

		alert := objc.ID(class("NSAlert")).Send(sel("new"))
		defer alert.Send(sel("release"))
		alert.Send(sel("setMessageText:"), nsString(i18n.Tf(i18n.GroupDeleteTitle, name)))
		alert.Send(sel("setInformativeText:"), nsString(i18n.T(i18n.GroupDeleteBody)))
		alert.Send(sel("addButtonWithTitle:"), nsString(i18n.T(i18n.CommonDelete)))
		alert.Send(sel("addButtonWithTitle:"), nsString(i18n.T(i18n.CommonCancel)))
		window := objc.Send[objc.ID](alert, sel("window"))
		window.Send(sel("setLevel:"), int64(1000))
		window.Send(sel("makeKeyAndOrderFront:"), objc.ID(0))
		confirmed = alert.Send(sel("runModal")) == 1000
	})
	return confirmed
}

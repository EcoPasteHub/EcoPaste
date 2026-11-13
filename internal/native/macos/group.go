//go:build darwin

package macos

import (
	"strings"

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

		title := "新建分组"
		if strings.TrimSpace(current) != "" {
			title = "重命名分组"
		}
		alert := objc.ID(class("NSAlert")).Send(sel("new"))
		defer alert.Send(sel("release"))
		alert.Send(sel("setMessageText:"), nsString(title))
		alert.Send(sel("setInformativeText:"), nsString("分组名称最多 32 个字符。"))
		alert.Send(sel("addButtonWithTitle:"), nsString("保存"))
		alert.Send(sel("addButtonWithTitle:"), nsString("取消"))

		field := objc.ID(classNSTextField).Send(sel("alloc")).
			Send(sel("initWithFrame:"), CGRect{Size: CGSize{Width: 320, Height: 24}})
		field.Send(sel("setStringValue:"), nsString(current))
		field.Send(sel("setPlaceholderString:"), nsString("例如：工作"))
		field.Send(sel("setFont:"), systemFont(13))
		field.Send(sel("setAccessibilityLabel:"), nsString("分组名称"))
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
		alert.Send(sel("setMessageText:"), nsString("删除分组“"+name+"”？"))
		alert.Send(sel("setInformativeText:"), nsString("分组中的剪贴板记录会保留，并移到“未分组”。"))
		alert.Send(sel("addButtonWithTitle:"), nsString("删除"))
		alert.Send(sel("addButtonWithTitle:"), nsString("取消"))
		window := objc.Send[objc.ID](alert, sel("window"))
		window.Send(sel("setLevel:"), int64(1000))
		window.Send(sel("makeKeyAndOrderFront:"), objc.ID(0))
		confirmed = alert.Send(sel("runModal")) == 1000
	})
	return confirmed
}

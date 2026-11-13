//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

// ClearHistoryChoice is the result of the clear-history dialog.
type ClearHistoryChoice int

const (
	ClearHistoryCancel ClearHistoryChoice = iota
	ClearHistoryKeepProtected
	ClearHistoryAll
)

// ConfirmClearHistory asks how much history to delete. The protected
// option keeps favorites and pinned items.
func ConfirmClearHistory() ClearHistoryChoice {
	choice := ClearHistoryCancel
	onMain(func() {
		alert := objc.ID(class("NSAlert")).Send(sel("new"))
		defer alert.Send(sel("release"))
		alert.Send(sel("setMessageText:"), nsString("清除剪贴板历史？"))
		alert.Send(sel("setInformativeText:"), nsString("可以保留收藏和置顶的记录。"))
		alert.Send(sel("addButtonWithTitle:"), nsString("取消"))
		alert.Send(sel("addButtonWithTitle:"), nsString("保留收藏与置顶"))
		alert.Send(sel("addButtonWithTitle:"), nsString("清除全部"))
		switch alert.Send(sel("runModal")) {
		case 1001:
			choice = ClearHistoryKeepProtected
		case 1002:
			choice = ClearHistoryAll
		}
	})
	return choice
}

// ShowMessage tells the user something with one acknowledged button.
func ShowMessage(title, text string) {
	onMain(func() {
		alert := objc.ID(class("NSAlert")).Send(sel("new"))
		defer alert.Send(sel("release"))
		alert.Send(sel("setMessageText:"), nsString(title))
		alert.Send(sel("setInformativeText:"), nsString(text))
		alert.Send(sel("addButtonWithTitle:"), nsString("好"))
		alert.Send(sel("runModal"))
	})
}

// Confirm asks before an action that changes local data.
func Confirm(title, text, confirmTitle string) bool {
	confirmed := false
	onMain(func() {
		alert := objc.ID(class("NSAlert")).Send(sel("new"))
		defer alert.Send(sel("release"))
		alert.Send(sel("setMessageText:"), nsString(title))
		alert.Send(sel("setInformativeText:"), nsString(text))
		alert.Send(sel("addButtonWithTitle:"), nsString("取消"))
		alert.Send(sel("addButtonWithTitle:"), nsString(confirmTitle))
		confirmed = alert.Send(sel("runModal")) == 1001
	})
	return confirmed
}

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
	alert := objc.ID(class("NSAlert")).Send(sel("new"))
	defer alert.Send(sel("release"))
	alert.Send(sel("setMessageText:"), nsString("清除剪贴板历史？"))
	alert.Send(sel("setInformativeText:"), nsString("可以保留收藏和置顶的记录。"))
	alert.Send(sel("addButtonWithTitle:"), nsString("取消"))
	alert.Send(sel("addButtonWithTitle:"), nsString("保留收藏与置顶"))
	alert.Send(sel("addButtonWithTitle:"), nsString("清除全部"))
	switch alert.Send(sel("runModal")) {
	case 1001:
		return ClearHistoryKeepProtected
	case 1002:
		return ClearHistoryAll
	default:
		return ClearHistoryCancel
	}
}

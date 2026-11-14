//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

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
		alert.Send(sel("setMessageText:"), nsString(i18n.T(i18n.ClearHistoryTitle)))
		alert.Send(sel("setInformativeText:"), nsString(i18n.T(i18n.ClearHistoryBody)))
		alert.Send(sel("addButtonWithTitle:"), nsString(i18n.T(i18n.CommonCancel)))
		alert.Send(sel("addButtonWithTitle:"), nsString(i18n.T(i18n.ClearHistoryProtected)))
		alert.Send(sel("addButtonWithTitle:"), nsString(i18n.T(i18n.ClearHistoryAll)))
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
		alert.Send(sel("addButtonWithTitle:"), nsString(i18n.T(i18n.CommonOK)))
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
		alert.Send(sel("addButtonWithTitle:"), nsString(i18n.T(i18n.CommonCancel)))
		alert.Send(sel("addButtonWithTitle:"), nsString(confirmTitle))
		confirmed = alert.Send(sel("runModal")) == 1001
	})
	return confirmed
}

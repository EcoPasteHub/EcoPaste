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

// newAlert builds a modal alert with the given message and ordered buttons.
// The caller owns the returned alert and must release it.
func newAlert(title, text string, buttons ...string) objc.ID {
	alert := objc.ID(class("NSAlert")).Send(sel("new"))
	alert.Send(sel("setMessageText:"), nsString(title))
	alert.Send(sel("setInformativeText:"), nsString(text))
	for _, button := range buttons {
		alert.Send(sel("addButtonWithTitle:"), nsString(button))
	}
	return alert
}

// ConfirmClearHistory asks how much history to delete. The protected
// option keeps favorites and pinned items.
func ConfirmClearHistory() ClearHistoryChoice {
	choice := ClearHistoryCancel
	onMain(func() {
		alert := newAlert(i18n.T(i18n.ClearHistoryTitle), i18n.T(i18n.ClearHistoryBody),
			i18n.T(i18n.CommonCancel), i18n.T(i18n.ClearHistoryProtected), i18n.T(i18n.ClearHistoryAll))
		defer alert.Send(sel("release"))
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
		alert := newAlert(title, text, i18n.T(i18n.CommonOK))
		defer alert.Send(sel("release"))
		alert.Send(sel("runModal"))
	})
}

// Confirm asks before an action that changes local data.
func Confirm(title, text, confirmTitle string) bool {
	confirmed := false
	onMain(func() {
		alert := newAlert(title, text, i18n.T(i18n.CommonCancel), confirmTitle)
		defer alert.Send(sel("release"))
		confirmed = alert.Send(sel("runModal")) == 1001
	})
	return confirmed
}

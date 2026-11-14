//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

// BackupExportChoice is how the user wants to write a backup.
type BackupExportChoice int

const (
	BackupExportCancel BackupExportChoice = iota
	BackupExportEncrypted
	BackupExportPlain
)

// BackupImportChoice is how the user wants to apply a backup.
type BackupImportChoice int

const (
	BackupImportCancel BackupImportChoice = iota
	BackupImportMerge
	BackupImportOverwrite
)

// ChooseBackupImport defaults to a non-destructive merge. Overwrite stays
// available but requires a separate confirmation at the call site.
func ChooseBackupImport() BackupImportChoice {
	choice := BackupImportCancel
	onMain(func() {
		alert := objc.ID(class("NSAlert")).Send(sel("new"))
		defer alert.Send(sel("release"))
		alert.Send(sel("setMessageText:"), nsString(i18n.T(i18n.BackupImportTitle)))
		alert.Send(sel("setInformativeText:"),
			nsString(i18n.T(i18n.BackupImportBody)))
		alert.Send(sel("addButtonWithTitle:"), nsString(i18n.T(i18n.BackupImportMerge)))
		alert.Send(sel("addButtonWithTitle:"), nsString(i18n.T(i18n.BackupImportOverwrite)))
		alert.Send(sel("addButtonWithTitle:"), nsString(i18n.T(i18n.CommonCancel)))
		switch alert.Send(sel("runModal")) {
		case 1000:
			choice = BackupImportMerge
		case 1001:
			choice = BackupImportOverwrite
		}
	})
	return choice
}

// ChooseBackupExport asks between an encrypted and a plain archive.
func ChooseBackupExport() BackupExportChoice {
	choice := BackupExportCancel
	onMain(func() {
		alert := objc.ID(class("NSAlert")).Send(sel("new"))
		defer alert.Send(sel("release"))
		alert.Send(sel("setMessageText:"), nsString(i18n.T(i18n.BackupExportTitle)))
		alert.Send(sel("setInformativeText:"), nsString(i18n.T(i18n.BackupExportBody)))
		alert.Send(sel("addButtonWithTitle:"), nsString(i18n.T(i18n.BackupExportEncrypted)))
		alert.Send(sel("addButtonWithTitle:"), nsString(i18n.T(i18n.BackupExportPlain)))
		alert.Send(sel("addButtonWithTitle:"), nsString(i18n.T(i18n.CommonCancel)))
		switch alert.Send(sel("runModal")) {
		case 1000:
			choice = BackupExportEncrypted
		case 1001:
			choice = BackupExportPlain
		}
	})
	return choice
}

// PromptPassword asks for the backup password without echoing it.
func PromptPassword(title, text string) (string, bool) {
	var password string
	confirmed := false
	onMain(func() {
		alert := objc.ID(class("NSAlert")).Send(sel("new"))
		defer alert.Send(sel("release"))
		alert.Send(sel("setMessageText:"), nsString(title))
		alert.Send(sel("setInformativeText:"), nsString(text))
		alert.Send(sel("addButtonWithTitle:"), nsString(i18n.T(i18n.CommonConfirm)))
		alert.Send(sel("addButtonWithTitle:"), nsString(i18n.T(i18n.CommonCancel)))
		field := objc.ID(class("NSSecureTextField")).Send(sel("alloc")).
			Send(sel("initWithFrame:"), CGRect{Size: CGSize{Width: 300, Height: 24}})
		defer field.Send(sel("release"))
		field.Send(sel("setPlaceholderString:"), nsString(i18n.T(i18n.BackupPasswordField)))
		alert.Send(sel("setAccessoryView:"), field)
		window := objc.Send[objc.ID](alert, sel("window"))
		window.Send(sel("setInitialFirstResponder:"), field)
		window.Send(sel("makeFirstResponder:"), field)
		if alert.Send(sel("runModal")) == 1000 {
			confirmed = true
			password = goString(objc.Send[objc.ID](field, sel("stringValue")))
		}
	})
	return password, confirmed
}

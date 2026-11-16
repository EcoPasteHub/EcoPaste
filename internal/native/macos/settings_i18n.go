//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

// bindText registers a translation update to run when the language changes.
func (v *SettingsView) bindText(update func()) {
	v.texts = append(v.texts, update)
}

// bindString updates a control's string value from a key.
func (v *SettingsView) bindString(control objc.ID, key i18n.Key) {
	v.bindText(func() {
		control.Send(sel("setStringValue:"), nsString(i18n.T(key)))
	})
}

// applyLanguage refreshes the static labels and derived rows in place, so the
// settings window keeps its identity, selection and scroll position.
func (v *SettingsView) applyLanguage(language string) {
	if language == "" || language == v.language {
		return
	}
	v.language = language
	for _, update := range v.texts {
		update()
	}
	for id, item := range v.tabItems {
		item.Send(sel("setLabel:"), nsString(settingsTabTitle(id)))
	}
	v.refreshPermissionLabels()
	v.apply(v.lastValues)
}

func settingsOptionText(option settingsOption) string {
	if option.Key != "" {
		return i18n.T(option.Key)
	}
	return option.Title
}

func settingsTextValue(value any) string {
	switch text := value.(type) {
	case i18n.Key:
		return i18n.T(text)
	case string:
		return text
	}
	return ""
}

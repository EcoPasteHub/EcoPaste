//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
)

// applyLanguage retranslates the panel's persistent chrome in place; history
// rows are rebuilt from their raw items by the UI language prop.
func (p *Panel) applyLanguage(language string) {
	if language == "" || language == p.language {
		return
	}
	p.language = language
	p.search.Send(sel("setPlaceholderString:"), nsString(i18n.T(i18n.PanelSearchPlaceholder)))
	p.updatePinLabel()
	p.filterButton.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.PanelFilter)))
	p.settingsButton.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.SettingsWindowTitle)))
	p.clearButton.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.PanelClearHistory)))
	p.shortcutButton.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.PanelShortcutsAction)))
	p.table.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.PanelHistoryLabel)))
	if p.previewImage != 0 {
		p.previewImage.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.PreviewImageLabel)))
	}
	p.refreshShortcutHelp()
	if p.nav != nil {
		p.nav.refreshLanguage()
	}
	p.refreshNewGroupPopoverText()
	p.refreshNotePopoverText()
	p.updateStatus()
	if p.hoveredRow >= 0 {
		p.updateHover(p.hoveredRow)
	}
}

func (p *Panel) updatePinLabel() {
	label := i18n.T(i18n.PanelPinWindow)
	if p.pinned {
		label = i18n.T(i18n.PanelUnpinWindow)
	}
	p.pin.Send(sel("setAccessibilityLabel:"), nsString(label))
}

// refreshLanguage relayouts the category capsules after their labels change.
func (n *panelNav) refreshLanguage() {
	labels := panelNavLabels()
	for index, capsule := range n.capsules {
		if index < len(labels) {
			capsule.Send(sel("setAccessibilityLabel:"), nsString(labels[index]))
		}
		capsule.Send(sel("setNeedsDisplay:"), true)
	}
	n.plus.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.PanelNewGroup)))
	n.layout()
}

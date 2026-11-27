package app

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

// loadSourceApps presents the registry to the settings window with each
// application's effective mode, including the macOS default for password
// managers that is never written to settings.json.
func (m *settingsModel) loadSourceApps(cfg settings.SourceApps) []macos.SettingsApp {
	if m.listApps == nil {
		return nil
	}
	apps, err := m.listApps()
	if err != nil {
		return nil
	}
	out := make([]macos.SettingsApp, 0, len(apps))
	for _, app := range apps {
		iconPath := ""
		if app.IconFile != nil && m.appIconPath != nil {
			iconPath = m.appIconPath(*app.IconFile)
		}
		out = append(out, macos.SettingsApp{
			ID: app.ID, Name: app.Name, IconPath: iconPath,
			Mode: string(cfg.ModeFor(app.ID)),
		})
	}
	return out
}

// applySourceAppMode stores one application's mode. Selecting the default
// removes the rule so only modified applications keep a record.
func (m *settingsModel) applySourceAppMode(value any) error {
	selection, ok := value.(macos.SettingsAppSelection)
	if !ok || selection.ID == "" {
		return i18n.Error(i18n.ErrorInvalidSourceApp)
	}
	mode := settings.SourceAppMode(selection.Mode)
	if !settings.IsSourceAppMode(mode) {
		return i18n.Error(i18n.ErrorInvalidSourceApp)
	}
	rules := m.store.Snapshot().Clipboard.SourceApps.Rules
	next := make([]settings.SourceAppRule, 0, len(rules)+1)
	for _, rule := range rules {
		if rule.ID != selection.ID {
			next = append(next, rule)
		}
	}
	// A rule is only needed when it differs from the platform default. This
	// also lets a password app be explicitly allowed without losing that
	// choice to the default rule.
	if mode != settings.DefaultSourceAppMode(selection.ID) {
		next = append(next, settings.SourceAppRule{ID: selection.ID, Mode: mode})
		if m.ensureApp != nil {
			if err := m.ensureApp(selection.ID, selection.Name); err != nil {
				return err
			}
		}
	}
	return m.setSourceApps(settings.NormalizeSourceAppRules(next))
}

func (m *settingsModel) setSourceApps(rules []settings.SourceAppRule) error {
	return m.update(map[string]any{
		"clipboard": map[string]any{
			"sourceApps": map[string]any{"rules": rules},
		},
	})
}

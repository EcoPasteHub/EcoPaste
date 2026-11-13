package app

import (
	"fmt"
	"slices"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
)

// loadSourceApps presents the registry to the settings window and marks
// which applications are currently excluded from capture.
func (m *settingsModel) loadSourceApps(excludedIDs []string) []macos.SettingsApp {
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
			Excluded: slices.Contains(excludedIDs, app.ID),
		})
	}
	return out
}

// applyExcludedApps saves the ignored list and deletes rows that no
// history entry references, matching the old transfer control.
func (m *settingsModel) applyExcludedApps(value any) error {
	ids, ok := value.([]string)
	if !ok {
		return fmt.Errorf("%s", i18n.T(i18n.ErrorInvalidExcludedApps))
	}
	ids = dedupeIDs(ids)
	previous := m.store.Snapshot().Clipboard.Filters.ExcludedAppIDs
	if err := m.update(map[string]any{
		"clipboard": map[string]any{
			"filters": map[string]any{"excludedAppIds": ids},
		},
	}); err != nil {
		return err
	}
	removed := differenceIDs(previous, ids)
	if len(removed) == 0 || m.deleteApps == nil {
		return nil
	}
	if err := m.deleteApps(removed); err != nil {
		return fmt.Errorf(i18n.T(i18n.ErrorCleanApps), err)
	}
	return nil
}

// addSourceApp registers a bundle chosen in the native open panel and
// adds it to the ignored list so the user sees the effect immediately.
func (m *settingsModel) addSourceApp(path string) error {
	if m.addApp == nil {
		return fmt.Errorf("%s", i18n.T(i18n.ErrorAddAppUnavailable))
	}
	app, err := m.addApp(path)
	if err != nil {
		return err
	}
	ids := append([]string(nil), m.store.Snapshot().Clipboard.Filters.ExcludedAppIDs...)
	if !slices.Contains(ids, app.ID) {
		ids = append(ids, app.ID)
	}
	return m.update(map[string]any{
		"clipboard": map[string]any{
			"filters": map[string]any{"excludedAppIds": ids},
		},
	})
}

func dedupeIDs(ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id != "" && !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out
}

func differenceIDs(left, right []string) []string {
	var out []string
	for _, id := range left {
		if !slices.Contains(right, id) {
			out = append(out, id)
		}
	}
	return out
}

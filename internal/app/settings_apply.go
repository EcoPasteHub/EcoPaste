package app

import (
	"fmt"
	"strings"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
)

// settingsCommands are actions settings controls trigger directly rather
// than by writing a value into settings.json.
var settingsCommands = map[string]func(*settingsModel, any) error{
	"settings.reset": func(m *settingsModel, _ any) error {
		return m.resetSettings()
	},
	"sourceApps.mode": func(m *settingsModel, value any) error {
		return m.applySourceAppMode(value)
	},
	"shortcuts.openClipboard": func(m *settingsModel, value any) error {
		return m.applyShortcut("shortcuts.openClipboard", value)
	},
	"shortcuts.openPreference": func(m *settingsModel, value any) error {
		return m.applyShortcut("shortcuts.openPreference", value)
	},
	"capture.updateOrder": func(m *settingsModel, value any) error {
		return m.applyCaptureOrder(value)
	},
	"backup.export": func(m *settingsModel, _ any) error {
		if m.exportBackup == nil {
			return i18n.Error(i18n.ErrorBackupUnavailable)
		}
		m.exportBackup()
		return nil
	},
	"backup.import": func(m *settingsModel, _ any) error {
		if m.importBackup == nil {
			return i18n.Error(i18n.ErrorBackupUnavailable)
		}
		m.importBackup()
		return nil
	},
	"update.check": func(m *settingsModel, _ any) error {
		if m.checkUpdates == nil {
			return i18n.Error(i18n.UpdateCheckFailedBody)
		}
		m.checkUpdates()
		return nil
	},
	"logs.open": func(m *settingsModel, _ any) error {
		if m.openLogs == nil {
			return i18n.Error(i18n.ErrorOpenFile)
		}
		return m.openLogs()
	},
	"general.dataDir": func(m *settingsModel, value any) error {
		parent, _ := value.(string)
		if m.changeDataDir == nil {
			return i18n.Error(i18n.ErrorDataDirChange)
		}
		if err := m.changeDataDir(parent); err != nil {
			return err
		}
		if m.restart != nil {
			m.restart()
		}
		return nil
	},
	"general.defaultDataDir": func(m *settingsModel, _ any) error {
		if m.restoreDataDir == nil {
			return i18n.Error(i18n.ErrorDataDirRestore)
		}
		if err := m.restoreDataDir(); err != nil {
			return err
		}
		if m.restart != nil {
			m.restart()
		}
		return nil
	},
	"history.retention": func(m *settingsModel, value any) error {
		text, _ := value.(string)
		retention, err := decodeRetention(text)
		if err != nil {
			return err
		}
		if err := m.update(map[string]any{
			"clipboard": map[string]any{
				"history": map[string]any{"retention": retention},
			},
		}); err != nil {
			return err
		}
		m.notifyChanged()
		return nil
	},
}

func (m *settingsModel) Apply(path string, value any) error {
	if command, ok := settingsCommands[path]; ok {
		return command(m, value)
	}
	return m.applySettingPath(path, value)
}

func (m *settingsModel) applySettingPath(path string, value any) error {
	keys, ok := settingsPaths[path]
	if !ok {
		return fmt.Errorf(i18n.T(i18n.ErrorUnknownSetting), path)
	}
	if err := m.applyBeforeSave(path, value); err != nil {
		return err
	}
	patch := map[string]any{}
	node := patch
	for _, key := range keys[:len(keys)-1] {
		next := map[string]any{}
		node[key] = next
		node = next
	}
	node[keys[len(keys)-1]] = value
	if err := m.update(patch); err != nil {
		return err
	}
	if strings.HasPrefix(path, "logging.") && m.onLoggingChanged != nil {
		if err := m.onLoggingChanged(m.store.Snapshot().Logging); err != nil {
			return err
		}
	}
	if path == "appearance.language" && m.setLanguage != nil {
		language, _ := value.(string)
		if err := m.setLanguage(language); err != nil {
			return err
		}
	}
	m.notifyChanged()
	return nil
}

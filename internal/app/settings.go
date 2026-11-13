package app

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

type settingsModel struct {
	store                *settings.Store
	setDockIcon          func(bool) error
	setAutoStart         func(bool) error
	setAppearance        func(string) error
	listApps             func() ([]database.App, error)
	listGroups           func() ([]database.Group, error)
	addApp               func(string) (database.App, error)
	deleteApps           func([]string) error
	appIconPath          func(string) string
	onShortcutsChanged   func()
	onShortcutsSuspended func(bool)
	onDisplayChanged     func()
	onActionsChanged     func()
	dataDir              func() string
	changeDataDir        func(string) error
	restoreDataDir       func() error
	canRestoreDataDir    func() bool
	restart              func()
}

func newSettingsModel(store *settings.Store) *settingsModel {
	return &settingsModel{
		store: store, setDockIcon: macos.SetDockIconVisible,
		setAutoStart: macos.SetAutostart, setAppearance: macos.SetAppearance,
	}
}

func (m *settingsModel) Load() macos.SettingsValues {
	snapshot := m.store.Snapshot()
	cfg := snapshot.Clipboard
	excluded := cfg.Filters.ExcludedAppIDs
	return macos.SettingsValues{
		DockIcon:               snapshot.General.DockIcon,
		AutoStart:              snapshot.General.AutoStart,
		Theme:                  string(snapshot.Appearance.Theme),
		DataDir:                m.currentDataDir(),
		CanRestoreDataDir:      m.canRestoreDataDir != nil && m.canRestoreDataDir(),
		CopyPlain:              cfg.Content.CopyPlain,
		CopyThenHideWindow:     cfg.Content.CopyThenHideWindow,
		CopySound:              cfg.Feedback.CopySound,
		PastePlain:             cfg.Content.PastePlain,
		UpdateOnReuse:          cfg.Content.UpdateOnReuse,
		Sort:                   string(cfg.Content.Sort),
		WindowPosition:         string(cfg.Window.Position),
		CaptureText:            cfg.Capture.Text,
		CaptureHTML:            cfg.Capture.HTML,
		CaptureRTF:             cfg.Capture.RTF,
		CaptureImage:           cfg.Capture.Image,
		CaptureFiles:           cfg.Capture.Files,
		MaxTextMB:              cfg.Capture.MaxTextMB,
		MaxImageMB:             cfg.Capture.MaxImageMB,
		TextMaxLines:           uint32(clampInt(int(cfg.Display.TextMaxLines), 1, 5)),
		ImageMaxHeight:         uint32(clampInt(int(cfg.Display.ImageMaxHeight), 20, 100)),
		FileMaxCount:           uint32(clampInt(int(cfg.Display.FileMaxCount), 1, 5)),
		ScrollToTopOnOpen:      cfg.Window.ScrollToTopOnOpen,
		RangeOnOpen:            string(cfg.Window.SelectRangeOnOpen),
		CategoryOnOpen:         string(cfg.Window.SelectCategoryOnOpen),
		GroupOnOpen:            cfg.Window.SelectGroupOnOpen,
		Groups:                 m.loadGroups(),
		Retention:              encodeRetention(cfg.History.Retention),
		MaxCount:               cfg.History.MaxCount,
		CleanupIntervalHours:   cfg.History.CleanupIntervalHours,
		CollectSecrets:         cfg.Sensitive.CollectSecrets,
		RedactSecrets:          cfg.Sensitive.RedactSecrets,
		ExcludedAppIDs:         excluded,
		Apps:                   m.loadSourceApps(excluded),
		ItemActions:            actionValues(cfg.Content.ItemActions),
		ItemActionOrder:        actionValues(resolvedActionOrder(cfg.Content.ItemActionOrder)),
		OpenClipboardShortcut:  snapshot.Shortcuts.OpenClipboard,
		OpenPreferenceShortcut: snapshot.Shortcuts.OpenPreference,
	}
}

func (m *settingsModel) Apply(path string, value any) error {
	switch path {
	case "settings.reset":
		return m.resetSettings()
	case "apps.refresh":
		return nil
	case "apps.add":
		text, _ := value.(string)
		return m.addSourceApp(text)
	case "filters.excludedAppIds":
		return m.applyExcludedApps(value)
	case "shortcuts.openClipboard", "shortcuts.openPreference":
		return m.applyShortcut(path, value)
	case "actions.update":
		return m.applyActions(value)
	}
	if path == "general.dataDir" {
		parent, _ := value.(string)
		if m.changeDataDir == nil {
			return fmt.Errorf("当前环境无法更改数据目录")
		}
		if err := m.changeDataDir(parent); err != nil {
			return err
		}
		if m.restart != nil {
			m.restart()
		}
		return nil
	}
	if path == "general.defaultDataDir" {
		if m.restoreDataDir == nil {
			return fmt.Errorf("当前环境无法恢复默认数据目录")
		}
		if err := m.restoreDataDir(); err != nil {
			return err
		}
		if m.restart != nil {
			m.restart()
		}
		return nil
	}
	if path == "history.retention" {
		text, _ := value.(string)
		retention, err := decodeRetention(text)
		if err != nil {
			return err
		}
		return m.update(map[string]any{
			"clipboard": map[string]any{
				"history": map[string]any{"retention": retention},
			},
		})
	}
	keys, ok := settingsPaths[path]
	if !ok {
		return fmt.Errorf("未知设置项 %q", path)
	}
	if path == "general.autoStart" && m.setAutoStart != nil {
		enabled, _ := value.(bool)
		if err := m.setAutoStart(enabled); err != nil {
			return err
		}
	}
	if path == "general.dockIcon" && m.setDockIcon != nil {
		visible, _ := value.(bool)
		if err := m.setDockIcon(visible); err != nil {
			return err
		}
	}
	if path == "appearance.theme" && m.setAppearance != nil {
		theme, _ := value.(string)
		if err := m.setAppearance(theme); err != nil {
			return err
		}
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
	if strings.HasPrefix(path, "display.") && m.onDisplayChanged != nil {
		m.onDisplayChanged()
	}
	return nil
}

// applyActions stores the hover action selection and order. Unknown
// actions are rejected so a hand-edited file cannot hide a whole menu.
func (m *settingsModel) applyActions(value any) error {
	update, ok := value.(macos.SettingsItemActions)
	if !ok {
		return fmt.Errorf("无效的动作配置")
	}
	known := map[settings.ItemAction]struct{}{}
	for _, action := range resolvedActionOrder(nil) {
		known[action] = struct{}{}
	}
	order := make([]settings.ItemAction, 0, len(update.Order))
	seen := map[settings.ItemAction]struct{}{}
	for _, key := range update.Order {
		action := settings.ItemAction(key)
		if _, ok := known[action]; !ok {
			return fmt.Errorf("未知动作 %q", key)
		}
		if _, ok := seen[action]; ok {
			continue
		}
		seen[action] = struct{}{}
		order = append(order, action)
	}
	for _, action := range resolvedActionOrder(nil) {
		if _, ok := seen[action]; !ok {
			seen[action] = struct{}{}
			order = append(order, action)
		}
	}
	enabled := make([]settings.ItemAction, 0, len(update.Enabled))
	for _, key := range update.Enabled {
		action := settings.ItemAction(key)
		if _, ok := known[action]; !ok {
			return fmt.Errorf("未知动作 %q", key)
		}
		if _, ok := seen[action]; ok {
			enabled = append(enabled, action)
		}
	}
	if err := m.update(map[string]any{"clipboard": map[string]any{"content": map[string]any{
		"itemActions":     enabled,
		"itemActionOrder": order,
	}}}); err != nil {
		return err
	}
	if m.onActionsChanged != nil {
		m.onActionsChanged()
	}
	return nil
}

func actionValues(actions []settings.ItemAction) []string {
	values := make([]string, 0, len(actions))
	for _, action := range actions {
		values = append(values, string(action))
	}
	return values
}

func resolvedActionOrder(order []settings.ItemAction) []settings.ItemAction {
	full := settings.Default().Clipboard.Content.ItemActionOrder
	if len(order) == 0 {
		return full
	}
	known := map[settings.ItemAction]struct{}{}
	for _, action := range full {
		known[action] = struct{}{}
	}
	resolved := make([]settings.ItemAction, 0, len(full))
	seen := map[settings.ItemAction]struct{}{}
	for _, action := range order {
		if _, ok := known[action]; !ok {
			continue
		}
		if _, ok := seen[action]; ok {
			continue
		}
		seen[action] = struct{}{}
		resolved = append(resolved, action)
	}
	for _, action := range full {
		if _, ok := seen[action]; !ok {
			resolved = append(resolved, action)
		}
	}
	return resolved
}

// resetSettings restores the published defaults and re-applies the
// system side effects so the running app matches the new file.
func (m *settingsModel) resetSettings() error {
	next, err := m.store.Reset()
	if err != nil {
		return fmt.Errorf("恢复默认设置: %w", err)
	}
	if m.setDockIcon != nil {
		if err := m.setDockIcon(next.General.DockIcon); err != nil {
			return err
		}
	}
	if m.setAutoStart != nil {
		if err := m.setAutoStart(next.General.AutoStart); err != nil {
			return err
		}
	}
	if m.setAppearance != nil {
		if err := m.setAppearance(string(next.Appearance.Theme)); err != nil {
			return err
		}
	}
	if m.onShortcutsChanged != nil {
		m.onShortcutsChanged()
	}
	if m.onDisplayChanged != nil {
		m.onDisplayChanged()
	}
	return nil
}

func (m *settingsModel) update(patch map[string]any) error {
	_, err := m.store.Update(patch)
	return err
}

// SuspendShortcuts releases the global bindings while the recorder is
// capturing keys, then restores them when recording ends.
func (m *settingsModel) SuspendShortcuts(suspend bool) {
	if m.onShortcutsSuspended != nil {
		m.onShortcutsSuspended(suspend)
	}
}

func (m *settingsModel) currentDataDir() string {
	if m.dataDir == nil {
		return ""
	}
	return m.dataDir()
}

func (m *settingsModel) loadGroups() []macos.SettingsGroup {
	if m.listGroups == nil {
		return nil
	}
	groups, err := m.listGroups()
	if err != nil {
		return nil
	}
	out := make([]macos.SettingsGroup, 0, len(groups))
	for _, group := range groups {
		out = append(out, macos.SettingsGroup{ID: group.ID, Name: group.Name})
	}
	return out
}

var settingsPaths = map[string][]string{
	"general.dockIcon":             {"general", "dockIcon"},
	"general.autoStart":            {"general", "autoStart"},
	"appearance.theme":             {"appearance", "theme"},
	"copyPlain":                    {"clipboard", "content", "copyPlain"},
	"copyThenHideWindow":           {"clipboard", "content", "copyThenHideWindow"},
	"feedback.copySound":           {"clipboard", "feedback", "copySound"},
	"pastePlain":                   {"clipboard", "content", "pastePlain"},
	"updateOnReuse":                {"clipboard", "content", "updateOnReuse"},
	"sort":                         {"clipboard", "content", "sort"},
	"window.position":              {"clipboard", "window", "position"},
	"capture.text":                 {"clipboard", "capture", "text"},
	"capture.html":                 {"clipboard", "capture", "html"},
	"capture.rtf":                  {"clipboard", "capture", "rtf"},
	"capture.image":                {"clipboard", "capture", "image"},
	"capture.files":                {"clipboard", "capture", "files"},
	"capture.maxTextMb":            {"clipboard", "capture", "maxTextMb"},
	"capture.maxImageMb":           {"clipboard", "capture", "maxImageMb"},
	"display.textMaxLines":         {"clipboard", "display", "textMaxLines"},
	"display.imageMaxHeight":       {"clipboard", "display", "imageMaxHeight"},
	"display.fileMaxCount":         {"clipboard", "display", "fileMaxCount"},
	"window.scrollToTopOnOpen":     {"clipboard", "window", "scrollToTopOnOpen"},
	"window.selectRangeOnOpen":     {"clipboard", "window", "selectRangeOnOpen"},
	"window.selectCategoryOnOpen":  {"clipboard", "window", "selectCategoryOnOpen"},
	"window.selectGroupOnOpen":     {"clipboard", "window", "selectGroupOnOpen"},
	"history.maxCount":             {"clipboard", "history", "maxCount"},
	"history.cleanupIntervalHours": {"clipboard", "history", "cleanupIntervalHours"},
	"sensitive.collectSecrets":     {"clipboard", "sensitive", "collectSecrets"},
	"sensitive.redactSecrets":      {"clipboard", "sensitive", "redactSecrets"},
}

func encodeRetention(r settings.Retention) string {
	if r.Unit == settings.RetentionForever || r.Value == 0 {
		return "forever"
	}
	unit := "d"
	switch r.Unit {
	case settings.RetentionHours:
		unit = "h"
	case settings.RetentionWeeks:
		unit = "w"
	case settings.RetentionMonths:
		unit = "mo"
	}
	return strconv.FormatUint(uint64(r.Value), 10) + unit
}

func decodeRetention(text string) (settings.Retention, error) {
	if text == "forever" {
		return settings.Retention{Unit: settings.RetentionForever}, nil
	}
	if strings.HasSuffix(text, "mo") {
		value, err := strconv.ParseUint(strings.TrimSuffix(text, "mo"), 10, 32)
		if err != nil {
			return settings.Retention{}, fmt.Errorf("无法识别保留时长 %q", text)
		}
		return settings.Retention{Value: uint32(value), Unit: settings.RetentionMonths}, nil
	}
	if len(text) < 2 {
		return settings.Retention{}, fmt.Errorf("无法识别保留时长 %q", text)
	}
	value, err := strconv.ParseUint(text[:len(text)-1], 10, 32)
	if err != nil {
		return settings.Retention{}, fmt.Errorf("无法识别保留时长 %q", text)
	}
	unit := settings.RetentionDays
	switch text[len(text)-1:] {
	case "h":
		unit = settings.RetentionHours
	case "d":
		unit = settings.RetentionDays
	case "w":
		unit = settings.RetentionWeeks
	default:
		return settings.Retention{}, fmt.Errorf("无法识别保留时长单位 %q", text)
	}
	return settings.Retention{Value: uint32(value), Unit: unit}, nil
}

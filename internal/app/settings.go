package app

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

type settingsModel struct {
	store                *settings.Store
	setTrayIcon          func(bool) error
	setAutoStart         func(bool) error
	setAppearance        func(string) error
	setLanguage          func(string) error
	listApps             func() ([]database.App, error)
	listGroups           func() ([]database.Group, error)
	addApp               func(string) (database.App, error)
	deleteApps           func([]string) error
	appIconPath          func(string) string
	onShortcutsChanged   func(string) error
	onShortcutsSuspended func(bool)
	onChanged            func()
	onHistoryChanged     func()
	dataDir              func() string
	changeDataDir        func(string) error
	restoreDataDir       func() error
	canRestoreDataDir    func() bool
	restart              func()
	exportBackup         func()
	importBackup         func()
	checkUpdates         func()
	openLogs             func() error
	onLoggingChanged     func(settings.Logging) error
}

func newSettingsModel(store *settings.Store) *settingsModel {
	return &settingsModel{
		store:        store,
		setAutoStart: macos.SetAutostart, setAppearance: macos.SetAppearance,
	}
}

func (m *settingsModel) Load() macos.SettingsValues {
	snapshot := m.store.Snapshot()
	cfg := snapshot.Clipboard
	excluded := cfg.Filters.ExcludedAppIDs
	groups := m.loadGroups()
	return macos.SettingsValues{
		TrayIcon:               snapshot.General.TrayIcon,
		AutoStart:              snapshot.General.AutoStart,
		Theme:                  string(snapshot.Appearance.Theme),
		Language:               string(snapshot.Appearance.Language),
		DataDir:                m.currentDataDir(),
		CanRestoreDataDir:      m.canRestoreDataDir != nil && m.canRestoreDataDir(),
		CopyPlain:              cfg.Content.CopyPlain,
		CopyThenHideWindow:     cfg.Content.CopyThenHideWindow,
		CopySound:              cfg.Feedback.CopySound,
		PastePlain:             cfg.Content.PastePlain,
		UpdateOnReuse:          cfg.Content.UpdateOnReuse,
		AutoPaste:              string(cfg.Content.AutoPaste),
		MiddleClick:            string(cfg.Content.MiddleClick),
		PasteFilesAsPath:       cfg.Content.PasteFilesAsPath,
		AutoFavorite:           cfg.Content.AutoFavorite,
		ShowOriginalPreview:    cfg.Content.ShowOriginalPreview,
		DeleteConfirm:          cfg.Content.DeleteConfirm,
		DeleteFavoriteConfirm:  cfg.Content.DeleteFavoriteConfirm,
		DeletePinnedConfirm:    cfg.Content.DeletePinnedConfirm,
		DeleteFavoriteItems:    cfg.Content.DeleteFavoriteItems,
		DeletePinnedItems:      cfg.Content.DeletePinnedItems,
		DeleteFavoriteOnlyFav:  cfg.Content.DeleteFavoriteItemsOnlyInFavGroup,
		Sort:                   string(cfg.Content.Sort),
		WindowPosition:         string(cfg.Window.Position),
		CaptureText:            cfg.Capture.Text,
		CaptureHTML:            cfg.Capture.HTML,
		CaptureRTF:             cfg.Capture.RTF,
		CaptureImage:           cfg.Capture.Image,
		CaptureFiles:           cfg.Capture.Files,
		CaptureOrder:           captureOrderValues(cfg.Capture.Order),
		MaxTextMB:              cfg.Capture.MaxTextMB,
		MaxImageMB:             cfg.Capture.MaxImageMB,
		TextMaxLines:           uint32(clampInt(int(cfg.Display.TextMaxLines), 1, 5)),
		ImageMaxHeight:         uint32(clampInt(int(cfg.Display.ImageMaxHeight), 20, 100)),
		FileMaxCount:           uint32(clampInt(int(cfg.Display.FileMaxCount), 1, 5)),
		ScrollToTopOnOpen:      cfg.Window.ScrollToTopOnOpen,
		RangeOnOpen:            string(cfg.Window.SelectRangeOnOpen),
		CategoryOnOpen:         string(cfg.Window.SelectCategoryOnOpen),
		GroupOnOpen:            openGroupValue(cfg.Window.SelectGroupOnOpen, groups),
		SearchFocus:            cfg.Search.DefaultFocus,
		SearchClear:            cfg.Search.ClearOnHide,
		Groups:                 groups,
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
		SpacePreview:           cfg.Preview.SpaceEnabled,
		LogLevel:               string(snapshot.Logging.Level),
		LogRetentionDays:       snapshot.Logging.RetentionDays,
	}
}

// applyBeforeSave applies the settings that must reach the system before the
// file is written, so a failed native call does not leave the two out of sync.
func (m *settingsModel) applyBeforeSave(path string, value any) error {
	switch path {
	case "general.autoStart":
		if m.setAutoStart == nil {
			return nil
		}
		enabled, _ := value.(bool)
		return m.setAutoStart(enabled)
	case "general.trayIcon":
		if m.setTrayIcon == nil {
			return nil
		}
		visible, _ := value.(bool)
		return m.setTrayIcon(visible)
	case "appearance.theme":
		if m.setAppearance == nil {
			return nil
		}
		theme, _ := value.(string)
		return m.setAppearance(theme)
	}
	return nil
}

// applyActions stores the hover action selection and order. Unknown
// actions are rejected so a hand-edited file cannot hide a whole menu.
func (m *settingsModel) applyActions(value any) error {
	update, ok := value.(macos.SettingsItemActions)
	if !ok {
		return i18n.Error(i18n.ErrorInvalidActions)
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
			return fmt.Errorf(i18n.T(i18n.ErrorUnknownAction), key)
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
			return fmt.Errorf(i18n.T(i18n.ErrorUnknownAction), key)
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
	m.notifyChanged()
	return nil
}

func actionValues(actions []settings.ItemAction) []string {
	values := make([]string, 0, len(actions))
	for _, action := range actions {
		values = append(values, string(action))
	}
	return values
}

// applyCaptureOrder persists a complete capture priority list. Duplicate
// kinds are collapsed and missing kinds are appended so capture cannot
// silently lose a representation after a malformed update.
func (m *settingsModel) applyCaptureOrder(value any) error {
	values, ok := value.([]string)
	if !ok {
		return i18n.Error(i18n.ErrorInvalidCaptureOrder)
	}
	order := make([]settings.CaptureKind, 0, len(values))
	for _, text := range values {
		kind := settings.CaptureKind(text)
		if !settings.IsCaptureKind(kind) {
			return fmt.Errorf(i18n.T(i18n.ErrorUnknownCaptureKind), text)
		}
		order = append(order, kind)
	}
	order = settings.NormalizeCaptureOrder(order)
	return m.update(map[string]any{"clipboard": map[string]any{
		"capture": map[string]any{"order": order},
	}})
}

func captureOrderValues(order []settings.CaptureKind) []string {
	normalized := settings.NormalizeCaptureOrder(order)
	values := make([]string, 0, len(normalized))
	for _, kind := range normalized {
		values = append(values, string(kind))
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
		return fmt.Errorf(i18n.T(i18n.ErrorResetSettings), err)
	}
	if m.setTrayIcon != nil {
		if err := m.setTrayIcon(next.General.TrayIcon); err != nil {
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
	if m.setLanguage != nil {
		if err := m.setLanguage(string(next.Appearance.Language)); err != nil {
			return err
		}
	}
	if m.onShortcutsChanged != nil {
		if err := m.onShortcutsChanged(""); err != nil {
			return err
		}
	}
	if m.onLoggingChanged != nil {
		if err := m.onLoggingChanged(next.Logging); err != nil {
			return err
		}
	}
	m.notifyChanged()
	m.triggerCleanup()
	return nil
}

func (m *settingsModel) update(patch map[string]any) error {
	_, err := m.store.Update(patch)
	return err
}

// notifyChanged refreshes list-only settings (sort, redaction, delete rules,
// preview and search behavior) in the already-built panel.
func (m *settingsModel) notifyChanged() {
	if m.onChanged != nil {
		m.onChanged()
	}
}

// triggerCleanup applies a new retention or count limit right away instead of
// waiting for the next scheduled cleanup pass, which may be disabled.
func (m *settingsModel) triggerCleanup() {
	if m.onHistoryChanged != nil {
		m.onHistoryChanged()
	}
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

// openGroupValue falls back to "preserve" when the stored group was deleted,
// so the popup never shows a stale "group:<id>" item.
func openGroupValue(value string, groups []macos.SettingsGroup) string {
	if !strings.HasPrefix(value, "group:") {
		return value
	}
	id := strings.TrimPrefix(value, "group:")
	for _, group := range groups {
		if group.ID == id {
			return value
		}
	}
	return "preserve"
}

var settingsPaths = map[string][]string{
	"logging.level":                 {"logging", "level"},
	"logging.retentionDays":         {"logging", "retentionDays"},
	"general.trayIcon":              {"general", "trayIcon"},
	"general.autoStart":             {"general", "autoStart"},
	"appearance.theme":              {"appearance", "theme"},
	"appearance.language":           {"appearance", "language"},
	"copyPlain":                     {"clipboard", "content", "copyPlain"},
	"copyThenHideWindow":            {"clipboard", "content", "copyThenHideWindow"},
	"feedback.copySound":            {"clipboard", "feedback", "copySound"},
	"pastePlain":                    {"clipboard", "content", "pastePlain"},
	"content.autoPaste":             {"clipboard", "content", "autoPaste"},
	"content.middleClick":           {"clipboard", "content", "middleClick"},
	"content.pasteFilesAsPath":      {"clipboard", "content", "pasteFilesAsPath"},
	"content.autoFavorite":          {"clipboard", "content", "autoFavorite"},
	"content.showOriginalPreview":   {"clipboard", "content", "showOriginalPreview"},
	"content.deleteConfirm":         {"clipboard", "content", "deleteConfirm"},
	"content.deleteFavoriteConfirm": {"clipboard", "content", "deleteFavoriteConfirm"},
	"content.deletePinnedConfirm":   {"clipboard", "content", "deletePinnedConfirm"},
	"content.deleteFavoriteItems":   {"clipboard", "content", "deleteFavoriteItems"},
	"content.deletePinnedItems":     {"clipboard", "content", "deletePinnedItems"},
	"content.deleteFavoriteOnlyFav": {"clipboard", "content", "deleteFavoriteItemsOnlyInFavoriteGroup"},
	"updateOnReuse":                 {"clipboard", "content", "updateOnReuse"},
	"sort":                          {"clipboard", "content", "sort"},
	"window.position":               {"clipboard", "window", "position"},
	"capture.text":                  {"clipboard", "capture", "text"},
	"capture.html":                  {"clipboard", "capture", "html"},
	"capture.rtf":                   {"clipboard", "capture", "rtf"},
	"capture.image":                 {"clipboard", "capture", "image"},
	"capture.files":                 {"clipboard", "capture", "files"},
	"capture.maxTextMb":             {"clipboard", "capture", "maxTextMb"},
	"capture.maxImageMb":            {"clipboard", "capture", "maxImageMb"},
	"display.textMaxLines":          {"clipboard", "display", "textMaxLines"},
	"display.imageMaxHeight":        {"clipboard", "display", "imageMaxHeight"},
	"display.fileMaxCount":          {"clipboard", "display", "fileMaxCount"},
	"window.scrollToTopOnOpen":      {"clipboard", "window", "scrollToTopOnOpen"},
	"window.selectRangeOnOpen":      {"clipboard", "window", "selectRangeOnOpen"},
	"window.selectCategoryOnOpen":   {"clipboard", "window", "selectCategoryOnOpen"},
	"window.selectGroupOnOpen":      {"clipboard", "window", "selectGroupOnOpen"},
	"search.defaultFocus":           {"clipboard", "search", "defaultFocus"},
	"search.clearOnHide":            {"clipboard", "search", "clearOnHide"},
	"history.maxCount":              {"clipboard", "history", "maxCount"},
	"history.cleanupIntervalHours":  {"clipboard", "history", "cleanupIntervalHours"},
	"sensitive.collectSecrets":      {"clipboard", "sensitive", "collectSecrets"},
	"sensitive.redactSecrets":       {"clipboard", "sensitive", "redactSecrets"},
	"preview.spaceEnabled":          {"clipboard", "preview", "spaceEnabled"},
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
			return settings.Retention{}, fmt.Errorf(i18n.T(i18n.ErrorInvalidRetention), text)
		}
		return settings.Retention{Value: uint32(value), Unit: settings.RetentionMonths}, nil
	}
	if len(text) < 2 {
		return settings.Retention{}, fmt.Errorf(i18n.T(i18n.ErrorInvalidRetention), text)
	}
	value, err := strconv.ParseUint(text[:len(text)-1], 10, 32)
	if err != nil {
		return settings.Retention{}, fmt.Errorf(i18n.T(i18n.ErrorInvalidRetention), text)
	}
	var unit settings.RetentionUnit
	switch text[len(text)-1:] {
	case "h":
		unit = settings.RetentionHours
	case "d":
		unit = settings.RetentionDays
	case "w":
		unit = settings.RetentionWeeks
	default:
		return settings.Retention{}, fmt.Errorf(i18n.T(i18n.ErrorInvalidRetentionUnit), text)
	}
	return settings.Retention{Value: uint32(value), Unit: unit}, nil
}

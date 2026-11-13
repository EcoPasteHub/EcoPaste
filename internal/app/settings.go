package app

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

type settingsModel struct {
	store         *settings.Store
	setDockIcon   func(bool) error
	setAutoStart  func(bool) error
	setAppearance func(string) error
}

func newSettingsModel(store *settings.Store) *settingsModel {
	return &settingsModel{
		store: store, setDockIcon: macos.SetDockIconVisible,
		setAutoStart: macos.SetAutostart, setAppearance: macos.SetAppearance,
	}
}

func (m *settingsModel) Load() macos.SettingsValues {
	cfg := m.store.Snapshot().Clipboard
	return macos.SettingsValues{
		DockIcon:             m.store.Snapshot().General.DockIcon,
		AutoStart:            m.store.Snapshot().General.AutoStart,
		Theme:                string(m.store.Snapshot().Appearance.Theme),
		CopyPlain:            cfg.Content.CopyPlain,
		CopyThenHideWindow:   cfg.Content.CopyThenHideWindow,
		PastePlain:           cfg.Content.PastePlain,
		UpdateOnReuse:        cfg.Content.UpdateOnReuse,
		Sort:                 string(cfg.Content.Sort),
		WindowPosition:       string(cfg.Window.Position),
		CaptureText:          cfg.Capture.Text,
		CaptureHTML:          cfg.Capture.HTML,
		CaptureRTF:           cfg.Capture.RTF,
		CaptureImage:         cfg.Capture.Image,
		CaptureFiles:         cfg.Capture.Files,
		MaxTextMB:            cfg.Capture.MaxTextMB,
		MaxImageMB:           cfg.Capture.MaxImageMB,
		Retention:            encodeRetention(cfg.History.Retention),
		MaxCount:             cfg.History.MaxCount,
		CleanupIntervalHours: cfg.History.CleanupIntervalHours,
		CollectSecrets:       cfg.Sensitive.CollectSecrets,
		RedactSecrets:        cfg.Sensitive.RedactSecrets,
	}
}

func (m *settingsModel) Apply(path string, value any) error {
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
	return m.update(patch)
}

func (m *settingsModel) update(patch map[string]any) error {
	_, err := m.store.Update(patch)
	return err
}

var settingsPaths = map[string][]string{
	"general.dockIcon":             {"general", "dockIcon"},
	"general.autoStart":            {"general", "autoStart"},
	"appearance.theme":             {"appearance", "theme"},
	"copyPlain":                    {"clipboard", "content", "copyPlain"},
	"copyThenHideWindow":           {"clipboard", "content", "copyThenHideWindow"},
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

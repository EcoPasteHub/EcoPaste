package app

import (
	"path/filepath"
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

func TestSettingsModelAppliesPaths(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	model := newSettingsModel(store)
	if err := model.Apply("copyPlain", true); err != nil {
		t.Fatal(err)
	}
	if err := model.Apply("capture.maxTextMb", uint32(8)); err != nil {
		t.Fatal(err)
	}
	if err := model.Apply("history.retention", "7d"); err != nil {
		t.Fatal(err)
	}
	if err := model.Apply("sensitive.redactSecrets", false); err != nil {
		t.Fatal(err)
	}
	if err := model.Apply("window.position", "remember"); err != nil {
		t.Fatal(err)
	}

	values := model.Load()
	if !values.CopyPlain || values.MaxTextMB != 8 || values.Retention != "7d" || values.RedactSecrets ||
		values.WindowPosition != "remember" {
		t.Errorf("values = %+v", values)
	}
	cfg := store.Snapshot().Clipboard
	if cfg.History.Retention.Value != 7 || cfg.History.Retention.Unit != settings.RetentionDays {
		t.Errorf("retention = %+v", cfg.History.Retention)
	}
	if cfg.Window.Position != settings.PositionRemember {
		t.Errorf("window position = %q", cfg.Window.Position)
	}
}

func TestRetentionEncoding(t *testing.T) {
	cases := map[settings.Retention]string{
		{Unit: settings.RetentionForever}:          "forever",
		{Value: 2, Unit: settings.RetentionHours}:  "2h",
		{Value: 3, Unit: settings.RetentionDays}:   "3d",
		{Value: 2, Unit: settings.RetentionWeeks}:  "2w",
		{Value: 1, Unit: settings.RetentionMonths}: "1mo",
	}
	for retention, want := range cases {
		if got := encodeRetention(retention); got != want {
			t.Errorf("encodeRetention(%+v) = %q, want %q", retention, got, want)
		}
		decoded, err := decodeRetention(want)
		if err != nil {
			t.Fatal(err)
		}
		if decoded != retention {
			t.Errorf("decodeRetention(%q) = %+v, want %+v", want, decoded, retention)
		}
	}
}

func TestSettingsModelRejectsUnknownPath(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	if err := newSettingsModel(store).Apply("missing", true); err == nil {
		t.Error("unknown path was accepted")
	}
}

func TestSettingsModelAppliesDockIcon(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	model := newSettingsModel(store)
	var applied []bool
	model.setDockIcon = func(visible bool) error {
		applied = append(applied, visible)
		return nil
	}
	if err := model.Apply("general.dockIcon", true); err != nil {
		t.Fatal(err)
	}
	if !store.Snapshot().General.DockIcon {
		t.Error("dock icon setting was not persisted")
	}
	if len(applied) != 1 || !applied[0] {
		t.Errorf("dock icon applications = %v", applied)
	}
	if !model.Load().DockIcon {
		t.Error("dock icon setting was not loaded")
	}
}

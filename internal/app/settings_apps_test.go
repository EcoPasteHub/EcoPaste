package app

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

func TestSettingsModelAppliesExcludedAppsAndDeletesRemoved(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	model := newSettingsModel(store)
	model.listApps = func() ([]database.App, error) {
		icon := "icon.png"
		return []database.App{
			{ID: "com.example.a", Name: "A", IconFile: &icon},
			{ID: "com.example.b", Name: "B"},
		}, nil
	}
	model.appIconPath = func(name string) string { return "/icons/" + name }
	var deleted []string
	model.deleteApps = func(ids []string) error {
		deleted = append(deleted, ids...)
		return nil
	}
	removedID := "com.example.a"
	keptID := "com.example.b"
	if err := model.Apply("filters.excludedAppIds",
		[]string{"com.apple.keychainaccess", "com.apple.Passwords", removedID, keptID}); err != nil {
		t.Fatal(err)
	}
	if err := model.Apply("filters.excludedAppIds",
		[]string{"com.apple.keychainaccess", "com.apple.Passwords", keptID}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(deleted, []string{removedID}) {
		t.Errorf("deleted = %v", deleted)
	}
	if got := store.Snapshot().Clipboard.Filters.ExcludedAppIDs; !slices.Contains(got, keptID) ||
		slices.Contains(got, removedID) {
		t.Errorf("excluded = %v", got)
	}

	values := model.Load()
	if len(values.Apps) != 2 || values.Apps[0].IconPath != "/icons/icon.png" || !values.Apps[1].Excluded {
		t.Errorf("apps = %+v", values.Apps)
	}
}

func TestSettingsModelAddSourceAppExcludesIt(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	model := newSettingsModel(store)
	model.addApp = func(path string) (database.App, error) {
		if path != "/Applications/Example.app" {
			t.Errorf("path = %q", path)
		}
		return database.App{ID: "com.example.added", Name: "Added"}, nil
	}
	if err := model.Apply("apps.add", "/Applications/Example.app"); err != nil {
		t.Fatal(err)
	}
	excluded := store.Snapshot().Clipboard.Filters.ExcludedAppIDs
	if !slices.Contains(excluded, "com.example.added") {
		t.Errorf("excluded = %v", excluded)
	}
}

package app

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/EcoPasteHub/EcoPaste/internal/storage"
)

func testAppRegistry(t *testing.T) (*appRegistry, *database.DB, *storage.AppIconStore) {
	t.Helper()
	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	store := storage.NewAppIconStore(dir)
	icons := newAppIconCache(store)
	icons.load = func(string) []byte { return testPNG(t) }
	registry := newAppRegistry(db, store, icons)
	registry.running = func() []macos.RunningApp { return nil }
	return registry, db, store
}

func TestAppRegistryMergesRunningApps(t *testing.T) {
	registry, db, _ := testAppRegistry(t)
	ctx := context.Background()
	if err := db.EnsureApp(ctx, "com.example.zebra", "Zebra", nil); err != nil {
		t.Fatal(err)
	}
	registry.running = func() []macos.RunningApp {
		return []macos.RunningApp{
			{BundleID: "com.example.apple", Name: "apple"},
			{BundleID: "com.example.zebra", Name: "Zebra"},
		}
	}

	apps, err := registry.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 2 || apps[0].ID != "com.example.apple" || apps[1].ID != "com.example.zebra" {
		t.Fatalf("apps = %+v", apps)
	}
	for _, app := range apps {
		if app.IconFile == nil {
			t.Errorf("%s has no icon", app.ID)
		}
	}
}

func TestAppRegistryAddRefreshesWithoutDuplicates(t *testing.T) {
	registry, db, store := testAppRegistry(t)
	ctx := context.Background()
	registry.resolve = func(path string) (macos.RunningApp, error) {
		return macos.RunningApp{BundleID: "com.example.editor", Name: path, Path: path}, nil
	}
	registry.iconFromPath = func(string) []byte { return testPNG(t) }

	first, err := registry.Add(ctx, "First")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.Add(ctx, "Renamed")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID || second.Name != "Renamed" {
		t.Errorf("first = %+v, second = %+v", first, second)
	}
	apps, err := db.ListApps(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 1 {
		t.Fatalf("apps = %+v", apps)
	}
	if second.IconFile == nil || !store.Exists(*second.IconFile) {
		t.Errorf("icon = %v", second.IconFile)
	}
}

func TestAppRegistryDeletePrunesUnusedIcon(t *testing.T) {
	registry, db, store := testAppRegistry(t)
	ctx := context.Background()
	keptIcon, err := store.StorePNG(testPNGBytes(t, 16, 16, 20))
	if err != nil {
		t.Fatal(err)
	}
	removedIcon, err := store.StorePNG(testPNGBytes(t, 16, 16, 80))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.EnsureApp(ctx, "com.example.kept", "Kept", &keptIcon); err != nil {
		t.Fatal(err)
	}
	if err := db.EnsureApp(ctx, "com.example.removed", "Removed", &removedIcon); err != nil {
		t.Fatal(err)
	}
	source := "com.example.kept"
	item := &database.Item{Kind: database.KindText, Content: "hello", Platform: "macos", SourceAppID: &source}
	if _, _, err := db.Upsert(ctx, item); err != nil {
		t.Fatal(err)
	}

	deleted, err := registry.DeleteUnreferenced(ctx, []string{"com.example.kept", "com.example.removed"})
	if err != nil {
		t.Fatal(err)
	}
	if len(deleted) != 1 || deleted[0] != "com.example.removed" {
		t.Errorf("deleted = %v", deleted)
	}
	if !store.Exists(keptIcon) {
		t.Error("referenced icon was removed")
	}
	if store.Exists(removedIcon) {
		t.Error("unreferenced icon still exists")
	}
}

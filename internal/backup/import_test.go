package backup

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/storage"
)

func TestInspectBackupModes(t *testing.T) {
	src, root := testSource(t)
	plain := filepath.Join(root, "plain.ecopastebak")
	if _, err := Export(context.Background(), src, plain, ExportOptions{Mode: ModePlain}); err != nil {
		t.Fatal(err)
	}
	if mode, err := Inspect(plain); err != nil || mode != ModePlain {
		t.Errorf("Inspect(plain) = %q, %v", mode, err)
	}
	encrypted := filepath.Join(root, "encrypted.ecopastebak")
	if _, err := Export(context.Background(), src, encrypted, ExportOptions{
		Mode: ModeEncrypted, Password: "password",
	}); err != nil {
		t.Fatal(err)
	}
	if mode, err := Inspect(encrypted); err != nil || mode != ModeEncrypted {
		t.Errorf("Inspect(encrypted) = %q, %v", mode, err)
	}
}

func TestStageAndApplyImport(t *testing.T) {
	src, sourceRoot := testSource(t)
	archive := filepath.Join(sourceRoot, "history.ecopastebak")
	if _, err := Export(context.Background(), src, archive, ExportOptions{Mode: ModePlain}); err != nil {
		t.Fatal(err)
	}
	paths := testImportPaths(t)
	plan, err := StageImport(archive, "", paths.State)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Items != 3 || plan.Texts != 1 || plan.Images != 1 || plan.Files != 1 {
		t.Errorf("plan = %+v", plan)
	}
	result, applied, err := ApplyPendingImport(paths)
	if err != nil {
		t.Fatal(err)
	}
	if !applied || result.Items != 3 {
		t.Errorf("applied = %t, result = %+v", applied, result)
	}
	db, err := database.Open(filepath.Join(paths.DB, "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	counts, err := db.Counts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if counts.Items != 3 {
		t.Errorf("imported counts = %+v", counts)
	}
	settings, err := os.ReadFile(filepath.Join(paths.Config, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(settings) != `{"general":{}}` {
		t.Errorf("settings = %q", settings)
	}
	if _, err := os.Stat(filepath.Join(paths.Resources, "clipboard-images", "origin", "a.png")); err != nil {
		t.Error(err)
	}
	if _, err := os.Stat(filepath.Join(paths.Resources, "old.png")); err == nil {
		t.Error("old resources were kept")
	}
	for _, path := range []string{
		filepath.Join(paths.State, pendingImportDir),
		filepath.Join(paths.State, "import-rollback"),
	} {
		if _, err := os.Stat(path); err == nil {
			t.Errorf("%s was not cleaned up", path)
		}
	}
}

func TestApplyImportRestoresOnFailure(t *testing.T) {
	paths := testImportPaths(t)
	pending := filepath.Join(paths.State, pendingImportDir)
	if err := os.MkdirAll(filepath.Join(pending, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pending, "manifest.json"), []byte(`{"formatVersion":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pending, "config", "settings.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, applied, err := ApplyPendingImport(paths); err == nil || !applied {
		t.Fatalf("applied = %t, err = %v", applied, err)
	}
	if _, err := os.Stat(filepath.Join(paths.DB, "clipboard.db")); err != nil {
		t.Errorf("current database was removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(paths.Config, "settings.json")); err != nil {
		t.Errorf("current settings were removed: %v", err)
	}
}

func testImportPaths(t *testing.T) *storage.Paths {
	t.Helper()
	root := t.TempDir()
	paths := &storage.Paths{
		Root:      root,
		State:     filepath.Join(root, "state"),
		DB:        filepath.Join(root, "db"),
		Resources: filepath.Join(root, "resources"),
		Config:    filepath.Join(root, "config"),
	}
	for _, dir := range []string{paths.State, paths.DB, paths.Resources, paths.Config} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	db, err := database.Open(filepath.Join(paths.DB, "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := db.Upsert(context.Background(), &database.Item{
		Kind: database.KindText, Content: "old", Platform: "macos",
	}); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if err := os.WriteFile(filepath.Join(paths.Resources, "old.png"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(paths.Config, "settings.json"), []byte(`{"general":{"dockIcon":true}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	return paths
}

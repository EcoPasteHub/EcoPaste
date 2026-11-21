package backup

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"fmt"
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
	for _, mode := range []Mode{ModePlain, ModeEncrypted} {
		t.Run(string(mode), func(t *testing.T) { testStageAndApplyImport(t, mode) })
	}
}

func testStageAndApplyImport(t *testing.T, mode Mode) {
	ctx := context.Background()
	src, sourceRoot := testSource(t)
	archive := filepath.Join(sourceRoot, "history.ecopastebak")
	password := ""
	if mode == ModeEncrypted {
		password = "test password"
	}
	if _, err := Export(ctx, src, archive, ExportOptions{Mode: mode, Password: password}); err != nil {
		t.Fatal(err)
	}
	paths := testImportPaths(t)
	plan, err := StageImport(ctx, archive, password, paths.State)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Mode != mode || plan.Items != 3 || plan.Texts != 1 || plan.Images != 1 || plan.Files != 1 {
		t.Errorf("plan = %+v", plan)
	}
	result, applied, err := ApplyPendingImport(ctx, paths)
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
		filepath.Join(paths.State, storage.PendingBackupDir),
		filepath.Join(paths.State, storage.BackupRollbackDir),
	} {
		if _, err := os.Stat(path); err == nil {
			t.Errorf("%s was not cleaned up", path)
		}
	}
}

func TestMergeImport(t *testing.T) {
	src, sourceRoot := testSource(t)
	archive := filepath.Join(sourceRoot, "history.ecopastebak")
	if err := os.WriteFile(filepath.Join(src.Resources, "b.png"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Export(context.Background(), src, archive, ExportOptions{Mode: ModePlain}); err != nil {
		t.Fatal(err)
	}

	paths := testImportPaths(t)
	db, err := database.Open(filepath.Join(paths.DB, "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	existing := &database.Item{Kind: database.KindText, Content: "hello"}
	if _, _, err := db.Upsert(context.Background(), existing); err != nil {
		t.Fatal(err)
	}
	note := "local note"
	if err := db.SetNote(context.Background(), existing.ID, &note); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(paths.Resources, "clipboard-images", "origin"), 0o755); err != nil {
		t.Fatal(err)
	}
	localImage := filepath.Join(paths.Resources, "clipboard-images", "origin", "a.png")
	if err := os.WriteFile(localImage, []byte("local"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Merge(context.Background(), archive, "", db, paths.Resources)
	if err != nil {
		t.Fatal(err)
	}
	if result.Items != 2 || result.Skipped != 1 || result.Resources != 1 {
		t.Errorf("result = %+v", result)
	}
	if general, ok := result.Settings["general"].(map[string]any); !ok || len(general) != 0 {
		t.Errorf("settings patch = %#v", result.Settings)
	}
	counts, err := db.Counts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if counts.Items != 4 {
		t.Errorf("counts = %+v", counts)
	}
	merged, err := db.FindByHash(context.Background(), database.Hash(database.KindText, "hello"))
	if err != nil {
		t.Fatal(err)
	}
	if merged == nil || merged.Note == nil || *merged.Note != note {
		t.Errorf("duplicate metadata = %+v", merged)
	}
	image, err := db.FindByHash(context.Background(), database.Hash(database.KindImage, "a.png"))
	if err != nil {
		t.Fatal(err)
	}
	if image == nil || image.Content != "a.png" || image.UseCount != 1 {
		t.Errorf("imported image = %+v", image)
	}
	data, err := os.ReadFile(localImage)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "local" {
		t.Errorf("existing resource was overwritten: %q", data)
	}
	data, err = os.ReadFile(filepath.Join(paths.Resources, "b.png"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new" {
		t.Errorf("missing resource = %q", data)
	}
}

func TestApplyImportRestoresOnFailure(t *testing.T) {
	paths := testImportPaths(t)
	pending := filepath.Join(paths.State, storage.PendingBackupDir)
	if err := os.MkdirAll(filepath.Join(pending, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pending, "manifest.json"), fmt.Appendf(nil, `{"formatVersion":%d}`, formatVersion), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pending, "config", "settings.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, applied, err := ApplyPendingImport(context.Background(), paths); err == nil || !applied {
		t.Fatalf("applied = %t, err = %v", applied, err)
	}
	if _, err := os.Stat(filepath.Join(paths.DB, "clipboard.db")); err != nil {
		t.Errorf("current database was removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(paths.Config, "settings.json")); err != nil {
		t.Errorf("current settings were removed: %v", err)
	}
}

func TestImportRejectsUnsupportedBackupBeforeChangingLocalData(t *testing.T) {
	ctx := context.Background()
	paths := testImportPaths(t)
	current, err := database.Open(filepath.Join(paths.DB, "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer current.Close()
	root := t.TempDir()
	oldDBPath := filepath.Join(root, "clipboard.db")
	oldDB, err := sql.Open("sqlite", oldDBPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = oldDB.Exec(`CREATE TABLE clipboard_items (id TEXT PRIMARY KEY)`)
	oldDB.Close()
	if err != nil {
		t.Fatal(err)
	}
	dbData, err := os.ReadFile(oldDBPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []uint16{1, formatVersion} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			var data bytes.Buffer
			archive := zip.NewWriter(&data)
			for name, contents := range map[string][]byte{
				"manifest.json":        fmt.Appendf(nil, `{"formatVersion":%d}`, version),
				"db/clipboard.db":      dbData,
				"config/settings.json": []byte(`{}`),
				"resources/new.txt":    []byte("new"),
			} {
				entry, err := archive.Create(name)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := entry.Write(contents); err != nil {
					t.Fatal(err)
				}
			}
			if err := archive.Close(); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "backup.ecopastebak")
			if err := os.WriteFile(path, data.Bytes(), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Inspect(path); (err == nil) != (version == formatVersion) {
				t.Fatalf("Inspect format %d: %v", version, err)
			}
			if _, err := StageImport(ctx, path, "", paths.State); err == nil {
				t.Fatal("staged incompatible backup")
			}
			if _, err := Merge(ctx, path, "", current, paths.Resources); err == nil {
				t.Fatal("merged incompatible backup")
			}
			for _, path := range []string{filepath.Join(paths.State, storage.PendingBackupDir), filepath.Join(paths.Resources, "new.txt")} {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Fatalf("import changed %q: %v", path, err)
				}
			}
		})
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
		Kind: database.KindText, Content: "existing",
	}); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if err := os.WriteFile(filepath.Join(paths.Resources, "old.png"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(paths.Config, "settings.json"), []byte(`{"general":{"trayIcon":false}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	return paths
}

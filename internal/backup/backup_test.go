package backup

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
)

func testSource(t *testing.T) (Source, string) {
	t.Helper()
	root := t.TempDir()
	db, err := database.Open(filepath.Join(root, "db", "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	for _, item := range []*database.Item{
		{Kind: database.KindText, Content: "hello", Platform: "macos"},
		{Kind: database.KindImage, Content: "a.png", Platform: "macos"},
		{Kind: database.KindFiles, Content: "/tmp/a.txt", Platform: "macos"},
	} {
		if _, _, err := db.Upsert(context.Background(), item); err != nil {
			t.Fatal(err)
		}
	}
	resources := filepath.Join(root, "resources")
	if err := os.MkdirAll(filepath.Join(resources, "clipboard-images", "origin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(resources, "clipboard-images", "origin", "a.png"), []byte("image"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(resources, ".DS_Store"), []byte("skip"), 0o644); err != nil {
		t.Fatal(err)
	}
	settings := filepath.Join(root, "config", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settings), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settings, []byte(`{"general":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	return Source{
		Database: db, Resources: resources, Settings: settings,
		AppName: "EcoPaste", AppVersion: "0.1.0",
	}, root
}

func TestExportPlainBackup(t *testing.T) {
	src, root := testSource(t)
	target := filepath.Join(root, "history")
	result, err := Export(context.Background(), src, target, ExportOptions{Mode: ModePlain})
	if err != nil {
		t.Fatal(err)
	}
	if result.Path != target+"."+Extension {
		t.Errorf("path = %q", result.Path)
	}
	if result.Items != 3 || result.Texts != 1 || result.Images != 1 || result.Files != 1 {
		t.Errorf("result = %+v", result)
	}
	if result.ResourceBytes != int64(len("image")+len("skip")) {
		t.Errorf("resource bytes = %d", result.ResourceBytes)
	}
	entries := zipEntries(t, result.Path)
	for _, name := range []string{
		"manifest.json", "db/clipboard.db", "config/settings.json",
		"resources/clipboard-images/origin/a.png",
	} {
		if _, ok := entries[name]; !ok {
			t.Errorf("missing %s in %v", name, keys(entries))
		}
	}
	if _, ok := entries["resources/.DS_Store"]; ok {
		t.Error("backup included .DS_Store")
	}
	var manifest backupManifest
	if err := json.Unmarshal(entries["manifest.json"], &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.FormatVersion != formatVersion || manifest.Platform != "macos" ||
		manifest.Encryption != "none" || manifest.ItemCount != 3 ||
		manifest.AppName != "EcoPaste" || manifest.AppVersion != "0.1.0" {
		t.Errorf("manifest = %+v", manifest)
	}
}

func TestExportEncryptedBackupRoundTrips(t *testing.T) {
	src, root := testSource(t)
	target := filepath.Join(root, "secret.ecopastebak")
	const password = "correct horse battery staple"
	if _, err := Export(context.Background(), src, target, ExportOptions{
		Mode: ModeEncrypted, Password: password,
	}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte(magic)) {
		t.Fatal("encrypted backup has no magic header")
	}
	if !bytes.Contains(data, []byte(`"salt":[`)) {
		t.Error("salt is not a JSON array")
	}
	payload, err := readPayload(target, password)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(payload, []byte("PK")) {
		t.Error("decrypted payload is not a zip")
	}
	if _, err := readPayload(target, "wrong password"); err == nil ||
		!strings.Contains(err.Error(), "密码") {
		t.Errorf("wrong password error = %v", err)
	}
}

func TestNormalizePath(t *testing.T) {
	if got, err := NormalizePath("/tmp/a"); err != nil || got != "/tmp/a."+Extension {
		t.Errorf("NormalizePath without extension = %q, %v", got, err)
	}
	if got, err := NormalizePath("/tmp/a.ECOPASTEBAK"); err != nil || got != "/tmp/a.ECOPASTEBAK" {
		t.Errorf("NormalizePath with extension = %q, %v", got, err)
	}
	if _, err := NormalizePath("/tmp/a.zip"); err == nil {
		t.Error("NormalizePath accepted another extension")
	}
}

func TestBackupErrorsFollowLanguage(t *testing.T) {
	previous := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(string(previous)) })
	i18n.SetLanguage("en-US")

	if _, err := NormalizePath(" "); err == nil || err.Error() != "Choose a location for the backup file" {
		t.Fatalf("empty path error = %v", err)
	}
	if _, err := NormalizePath("backup.zip"); err == nil ||
		err.Error() != "Backup file extension must be .ecopastebak" {
		t.Fatalf("extension error = %v", err)
	}
}

func zipEntries(t *testing.T, path string) map[string][]byte {
	t.Helper()
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	entries := map[string][]byte{}
	for _, file := range reader.File {
		src, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(src)
		src.Close()
		if err != nil {
			t.Fatal(err)
		}
		entries[file.Name] = data
	}
	return entries
}

func keys(entries map[string][]byte) []string {
	result := make([]string, 0, len(entries))
	for name := range entries {
		result = append(result, name)
	}
	return result
}

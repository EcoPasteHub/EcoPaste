package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveCreatesDefaultTree(t *testing.T) {
	base := t.TempDir()
	paths, err := Resolve(base, Dev)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(base, Dev)
	if paths.Root != want || paths.Bootstrap != want {
		t.Fatalf("root=%q bootstrap=%q, want %q", paths.Root, paths.Bootstrap, want)
	}
	for _, dir := range []string{paths.DB, paths.Resources, paths.Config, paths.State, paths.Logs} {
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			t.Errorf("%s is not a directory: %v", dir, err)
		}
	}
	if _, err := os.Stat(filepath.Join(want, manifestName)); err != nil {
		t.Errorf("manifest missing: %v", err)
	}
}

func TestSetRootPointsManifestAtCustomDir(t *testing.T) {
	base := t.TempDir()
	custom := filepath.Join(t.TempDir(), "EcoPasteData", Prod)
	if err := SetRoot(base, Prod, custom); err != nil {
		t.Fatal(err)
	}
	paths, err := Resolve(base, Prod)
	if err != nil {
		t.Fatal(err)
	}
	if paths.Root != custom {
		t.Fatalf("root = %q, want %q", paths.Root, custom)
	}
}

func TestResolveFallsBackFromBrokenManifest(t *testing.T) {
	base := t.TempDir()
	bootstrap := filepath.Join(base, Prod)
	if err := os.MkdirAll(bootstrap, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bootstrap, manifestName), []byte("{broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	paths, err := Resolve(base, Prod)
	if err != nil {
		t.Fatal(err)
	}
	if paths.Root != bootstrap {
		t.Fatalf("root = %q, want fallback %q", paths.Root, bootstrap)
	}
	var m manifest
	data, err := os.ReadFile(filepath.Join(bootstrap, manifestName))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	if m.DataDir != bootstrap {
		t.Fatalf("manifest dataDir = %q, want %q", m.DataDir, bootstrap)
	}
}

func TestResolveRejectsForeignIdentity(t *testing.T) {
	base := t.TempDir()
	foreign := t.TempDir()
	data, err := json.Marshal(identity{Version: manifestVersion, Environment: "other"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(foreign, identityName), data, 0o644); err != nil {
		t.Fatal(err)
	}
	manifestData, err := json.Marshal(manifest{Version: manifestVersion, Environment: Prod, DataDir: foreign})
	if err != nil {
		t.Fatal(err)
	}
	bootstrap := filepath.Join(base, Prod)
	if err := os.MkdirAll(bootstrap, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bootstrap, manifestName), manifestData, 0o644); err != nil {
		t.Fatal(err)
	}
	paths, err := Resolve(base, Prod)
	if err != nil {
		t.Fatal(err)
	}
	if paths.Root != bootstrap {
		t.Fatalf("root = %q, want fallback %q", paths.Root, bootstrap)
	}
}

func TestValidateTargetRejectsNonEmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateTarget(dir, Prod); err == nil {
		t.Fatal("expected an error")
	}
}

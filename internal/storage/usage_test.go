package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMeasureUsageSumsParts(t *testing.T) {
	paths, err := Resolve(t.TempDir(), Dev)
	if err != nil {
		t.Fatal(err)
	}
	writeSized := func(path string, size int) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeSized(filepath.Join(paths.DB, "clipboard.db"), 100)
	writeSized(filepath.Join(paths.Resources, "clipboard-images", "origin", "aa", "a.png"), 40)
	writeSized(filepath.Join(paths.Config, "settings.json"), 10)

	usage, err := MeasureUsage(paths)
	if err != nil {
		t.Fatal(err)
	}
	if usage.Database != 100 || usage.Resource != 40 || usage.Config != 10 ||
		usage.Total < usage.Database+usage.Resource+usage.Config {
		t.Fatalf("usage = %+v", usage)
	}
}

func TestMeasureUsageSkipsMissingDirs(t *testing.T) {
	paths, err := Resolve(t.TempDir(), Dev)
	if err != nil {
		t.Fatal(err)
	}
	usage, err := MeasureUsage(paths)
	if err != nil {
		t.Fatal(err)
	}
	if usage.Database != 0 || usage.Resource != 0 || usage.Config != 0 {
		t.Fatalf("usage = %+v", usage)
	}
}

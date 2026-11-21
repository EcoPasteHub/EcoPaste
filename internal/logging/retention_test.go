package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func ageLogFile(t *testing.T, dir, name string, age time.Duration) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("test log"), 0o600); err != nil {
		t.Fatal(err)
	}
	stamp := time.Now().Add(-age)
	if err := os.Chtimes(path, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestOpenAppliesBuiltInLogRetention(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"ecopaste-old.log", "ecopaste-old.log.gz", "ecopaste-crash-old.log"} {
		ageLogFile(t, dir, name, 15*24*time.Hour)
	}
	for _, name := range []string{"ecopaste-recent.log.gz", "ecopaste-crash-recent.log"} {
		ageLogFile(t, dir, name, 13*24*time.Hour)
	}
	ageLogFile(t, dir, logFileName, 30*24*time.Hour)
	ageLogFile(t, dir, "unrelated.txt", 30*24*time.Hour)
	logger, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { logger.Close() })
	for _, name := range []string{"ecopaste-old.log", "ecopaste-old.log.gz", "ecopaste-crash-old.log"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Errorf("expired log %s remains: %v", name, err)
		}
	}
	for _, name := range []string{"ecopaste-recent.log.gz", "ecopaste-crash-recent.log", logFileName, "unrelated.txt"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("retained file %s was removed: %v", name, err)
		}
	}
	path := ageLogFile(t, dir, "ecopaste-expired-after-start.log.gz", 15*24*time.Hour)
	if err := logger.rotating.cleanup(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("maintenance left an expired log: %v", err)
	}
}

func TestLogCleanupKeepsNewestTwentyArchives(t *testing.T) {
	dir := t.TempDir()
	for index := range 25 {
		ageLogFile(t, dir, fmt.Sprintf("ecopaste-archive-%02d.log.gz", index), time.Duration(index)*time.Minute)
	}
	writer := &rotatingWriter{dir: dir}
	if err := writer.configure(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { writer.Close() })
	if err := writer.cleanup(); err != nil {
		t.Fatal(err)
	}
	for index := range 25 {
		_, err := os.Stat(filepath.Join(dir, fmt.Sprintf("ecopaste-archive-%02d.log.gz", index)))
		if index < 20 && err != nil || index >= 20 && !os.IsNotExist(err) {
			t.Errorf("archive %d: %v", index, err)
		}
	}
}

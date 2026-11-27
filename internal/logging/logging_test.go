package logging

import (
	"context"
	"log"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenWritesRedactedLogFile(t *testing.T) {
	dir := t.TempDir()
	logger, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	log.Printf("request token=%s", "sk-"+strings.Repeat("a", 32))
	if err := logger.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, logFileName))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Contains(text, strings.Repeat("a", 32)) {
		t.Errorf("token was written to the log: %q", text)
	}
	if !strings.Contains(text, "token=***") {
		t.Errorf("log line = %q", text)
	}
	if !strings.Contains(text, `"caller":"logging_test.go:`) {
		t.Errorf("standard logger caller was not recorded: %q", text)
	}
}

func TestRotatingWriterStartsNewFileAtSizeLimit(t *testing.T) {
	dir := t.TempDir()
	writer := &rotatingWriter{dir: dir}
	if err := writer.configure(); err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	for _, size := range []int{4 << 20, 2 << 20} {
		if n, err := writer.Write(make([]byte, size)); err != nil || n != size {
			t.Fatalf("write %d bytes returned %d, %v", size, n, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	foundRotation := false
	for _, entry := range entries {
		if isRotatedLog(entry.Name()) {
			foundRotation = true
			break
		}
	}
	if !foundRotation {
		t.Fatal("no rotated log backup was created")
	}
	info, err := os.Stat(filepath.Join(dir, logFileName))
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Size(); got != 2<<20 {
		t.Errorf("active log size = %d, want %d", got, 2<<20)
	}
}

func TestLoggerFiltersLevels(t *testing.T) {
	dir := t.TempDir()
	logger, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	slog.Info("filtered info")
	if err := logger.SetLevel(LevelInfo); err != nil {
		t.Fatal(err)
	}
	slog.Info("visible info", "component", "test.module")
	if err := logger.Close(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, logFileName))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Contains(text, "filtered info") {
		t.Errorf("below-threshold message was written: %q", text)
	}
	if !strings.Contains(text, `"time":`) || !strings.Contains(text, `"level":"INFO"`) ||
		!strings.Contains(text, `"source":{`) || !strings.Contains(text, `"component":"test.module"`) {
		t.Errorf("structured record fields missing: %q", text)
	}
}

func TestLoggerAllLevelIncludesLevelsBelowDebug(t *testing.T) {
	dir := t.TempDir()
	logger, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := logger.SetLevel(LevelAll); err != nil {
		t.Fatal(err)
	}
	slog.Log(context.Background(), slog.Level(math.MinInt+1), "visible custom level")
	if err := logger.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, logFileName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "visible custom level") {
		t.Fatalf("all level filtered a record below debug: %q", data)
	}
}

func TestOpenRestrictsLogDirectoryPermissions(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "logs")
	logger, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o700 {
		t.Errorf("log directory permissions = %04o, want 0700", got)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	crashPath := ""
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), crashFilePrefix) && strings.HasSuffix(entry.Name(), ".log") {
			crashPath = filepath.Join(dir, entry.Name())
			break
		}
	}
	if crashPath == "" {
		t.Fatal("crash output file was not prepared")
	}
	crashInfo, err := os.Stat(crashPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := crashInfo.Mode().Perm(); got != 0o600 {
		t.Errorf("crash output permissions = %04o, want 0600", got)
	}
}

func TestRedact(t *testing.T) {
	secret := strings.Repeat("x", 24)
	cases := []struct {
		input string
		want  string
	}{
		{"password = " + secret, "password = ***"},
		{`{"authorization":"secret-value"}`, `{"authorization":"***"}`},
		{"ordinary message", "ordinary message"},
		{"client_secret: " + secret, "client_secret: ***"},
		{"-----BEGIN " + "PRIVATE KEY-----\nMIIB\n-----END " + "PRIVATE KEY-----", "[private key]"},
	}
	for _, tc := range cases {
		if got := Redact(tc.input); got != tc.want {
			t.Errorf("Redact(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

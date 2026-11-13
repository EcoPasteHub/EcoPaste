package logging

import (
	"log"
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
}

func TestRedact(t *testing.T) {
	secret := strings.Repeat("x", 24)
	cases := []struct {
		input string
		want  string
	}{
		{"password = " + secret, "password = ***"},
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

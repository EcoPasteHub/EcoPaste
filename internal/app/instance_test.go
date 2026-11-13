//go:build darwin

package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLockInstanceRejectsSecond(t *testing.T) {
	dir := t.TempDir()
	first, running, err := lockInstance(dir)
	if err != nil {
		t.Fatal(err)
	}
	if running {
		t.Fatal("first lock reported a running instance")
	}
	defer first.Close()

	second, running, err := lockInstance(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !running || second != nil {
		t.Fatalf("second lock = %v, running = %t", second, running)
	}

	pid, err := instancePID(dir)
	if err != nil {
		t.Fatal(err)
	}
	if pid != os.Getpid() {
		t.Errorf("lock PID = %d, want %d", pid, os.Getpid())
	}
}

func TestLockInstanceReplacesStalePID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, instanceLockName)
	if err := os.WriteFile(path, []byte("99999999999"), 0o644); err != nil {
		t.Fatal(err)
	}
	lock, running, err := lockInstance(dir)
	if err != nil {
		t.Fatal(err)
	}
	if running {
		t.Fatal("stale lock reported a running instance")
	}
	defer lock.Close()
	pid, err := instancePID(dir)
	if err != nil {
		t.Fatal(err)
	}
	if pid != os.Getpid() {
		t.Errorf("lock PID = %d, want %d", pid, os.Getpid())
	}
}

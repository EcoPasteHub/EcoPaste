package app

import "testing"

func TestFirstBackupPath(t *testing.T) {
	path, ok := firstBackupPath([]string{"/tmp/notes.txt", "/tmp/history.ECOPASTEBAK"})
	if !ok || path != "/tmp/history.ECOPASTEBAK" {
		t.Errorf("firstBackupPath = %q, %t", path, ok)
	}
	if _, ok := firstBackupPath([]string{"/tmp/notes.txt"}); ok {
		t.Error("accepted a file that is not a backup")
	}
}

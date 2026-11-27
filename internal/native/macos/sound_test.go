//go:build darwin

package macos

import "testing"

func TestCopySoundIsAvailable(t *testing.T) {
	if copySound() == 0 {
		t.Fatalf("system sound %q is missing", copySoundName)
	}
}

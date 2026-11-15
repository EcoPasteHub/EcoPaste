//go:build darwin

package macos

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	runOnMain = func(fn func()) { fn() }
	os.Exit(m.Run())
}

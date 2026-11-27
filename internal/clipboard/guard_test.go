package clipboard

import (
	"testing"
)

func TestGuardMatchesOnlyItsPasteboardChangeCount(t *testing.T) {
	guard := NewGuard()
	if err := guard.Write(func() (int, bool, error) { return 4, true, nil }); err != nil {
		t.Fatal(err)
	}

	var matchedSelfWrite, matchedNextChange bool
	if err := guard.Synchronize(func(isSelfWrite func(int) bool) error {
		matchedSelfWrite = isSelfWrite(4)
		matchedNextChange = isSelfWrite(5)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !matchedSelfWrite {
		t.Fatal("EcoPaste write counter was not recognized")
	}
	if matchedNextChange {
		t.Fatal("a later external change was treated as an EcoPaste write")
	}
}

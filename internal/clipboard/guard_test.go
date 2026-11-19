package clipboard

import (
	"testing"
	"time"
)

func TestGuardMarkDropsExpiredEntries(t *testing.T) {
	g := NewGuard(2 * time.Second)
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	g.now = func() time.Time { return now }

	g.Mark("first")
	now = now.Add(3 * time.Second)
	g.Mark("second")

	g.mu.Lock()
	_, hasFirst := g.entries["first"]
	_, hasSecond := g.entries["second"]
	g.mu.Unlock()
	if hasFirst {
		t.Error("expired mark was not pruned")
	}
	if !hasSecond {
		t.Error("fresh mark missing")
	}
}

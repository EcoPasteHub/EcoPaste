package clipboard

import (
	"sync"
	"time"
)

// Guard suppresses the watcher's own pasteboard writes so pasting an
// item does not create a new history row.
type Guard struct {
	mu      sync.Mutex
	entries map[string]time.Time
	ttl     time.Duration
	now     func() time.Time
}

// NewGuard returns a guard whose marks live for ttl.
func NewGuard(ttl time.Duration) *Guard {
	return &Guard{entries: map[string]time.Time{}, ttl: ttl, now: time.Now}
}

// Mark records a hash about to be written to the pasteboard.
func (g *Guard) Mark(hash string) {
	if hash == "" {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	now := g.now()
	// Drop marks whose write never reached the watcher, so the map holds only
	// the writes from the last ttl instead of growing for the whole session.
	for key, expiry := range g.entries {
		if !now.Before(expiry) {
			delete(g.entries, key)
		}
	}
	g.entries[hash] = now.Add(g.ttl)
}

// Consume reports whether hash was written by EcoPaste recently, removing
// the mark so one write suppresses one capture.
func (g *Guard) Consume(hash string) bool {
	if hash == "" {
		return false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	expiry, ok := g.entries[hash]
	if !ok {
		return false
	}
	delete(g.entries, hash)
	return g.now().Before(expiry)
}

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
	g.entries[hash] = g.now().Add(g.ttl)
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

func (g *Guard) sweep() {
	now := g.now()
	g.mu.Lock()
	defer g.mu.Unlock()
	for hash, expiry := range g.entries {
		if now.After(expiry) {
			delete(g.entries, hash)
		}
	}
}

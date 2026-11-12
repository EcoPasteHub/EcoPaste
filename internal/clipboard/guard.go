package clipboard

import (
	"sync"
	"time"
)

const suppressTTL = 2 * time.Second

// WritebackGuard suppresses the watcher event caused by our own write.
type WritebackGuard struct {
	mu      sync.Mutex
	pending []pendingHash
	now     func() time.Time
}

type pendingHash struct {
	hash string
	at   time.Time
}

func NewWritebackGuard() *WritebackGuard {
	return &WritebackGuard{now: time.Now}
}

// Suppress registers a content hash written by EcoPaste.
func (g *WritebackGuard) Suppress(hash string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.prune()
	g.pending = append(g.pending, pendingHash{hash: hash, at: g.now()})
}

// Consume reports whether hash matches a live writeback and removes it.
func (g *WritebackGuard) Consume(hash string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.prune()
	for index, pending := range g.pending {
		if pending.hash == hash {
			g.pending = append(g.pending[:index], g.pending[index+1:]...)
			return true
		}
	}
	return false
}

func (g *WritebackGuard) prune() {
	cutoff := g.now().Add(-suppressTTL)
	live := g.pending[:0]
	for _, pending := range g.pending {
		if pending.at.After(cutoff) {
			live = append(live, pending)
		}
	}
	g.pending = live
}

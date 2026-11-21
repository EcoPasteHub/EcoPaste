package clipboard

import "sync"

// Guard serializes watcher snapshots with EcoPaste pasteboard writes.
type Guard struct {
	mu                 sync.Mutex
	lastSelfWriteCount int
	hasLastSelfWrite   bool
}

// NewGuard creates a guard for pasteboard writes.
func NewGuard() *Guard { return &Guard{} }

// Write performs a pasteboard write while excluding watcher reads and records
// the exact change count produced by a successful write.
func (g *Guard) Write(write func() (int, bool, error)) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	changeCount, changed, err := write()
	if changed {
		g.lastSelfWriteCount = changeCount
		g.hasLastSelfWrite = true
	}
	return err
}

// Synchronize keeps an entire watcher snapshot on one side of an EcoPaste
// write, preventing the watcher from reading a pasteboard mid-replacement.
func (g *Guard) Synchronize(read func(isSelfWrite func(changeCount int) bool) error) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return read(g.isSelfWriteLocked)
}

func (g *Guard) isSelfWriteLocked(changeCount int) bool {
	if !g.hasLastSelfWrite {
		return false
	}
	if changeCount == g.lastSelfWriteCount {
		return true
	}
	if changeCount > g.lastSelfWriteCount {
		g.hasLastSelfWrite = false
	}
	return false
}

package app

import (
	"context"
	"log"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/clipboard"
	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
	"github.com/EcoPasteHub/EcoPaste/internal/storage"
	"github.com/nsgo-dev/nsgo/core"
)

type clipboardRuntime struct {
	db            *database.DB
	service       *clipboard.Service
	images        *storage.ImageStore
	appIcons      *storage.AppIconStore
	fileIcons     *storage.FileIconStore
	fileIconCache *fileIconCache
	paths         *storage.Paths
	apps          *appRegistry
	watcher       *clipboard.Watcher
	changed       chan struct{}
}

func (r *clipboardRuntime) Signal() {
	select {
	case r.changed <- struct{}{}:
	default:
	}
}

// readClipboard captures what is on the pasteboard now, even while the
// automatic watcher is paused.
func (r *clipboardRuntime) readClipboard(ctx context.Context) error {
	_, err := r.watcher.Poll(ctx, true)
	return err
}

func startClipboard(app *core.Scope, paths *storage.Paths, store *settings.Store, paused *atomic.Bool) (*clipboardRuntime, error) {
	db, err := database.Open(filepath.Join(paths.DB, "clipboard.db"))
	if err != nil {
		return nil, err
	}
	images := storage.NewImageStore(paths.Resources)
	appIcons := storage.NewAppIconStore(paths.Resources)
	pruneIcons("app icons", db.AppIconFiles, appIcons.RemoveUnreferenced)
	fileIcons := storage.NewFileIconStore(paths.Resources)
	pruneIcons("file icons", db.FileIconFiles, fileIcons.RemoveUnreferenced)
	iconCache := newAppIconCache(appIcons)
	registry := newAppRegistry(db, appIcons, iconCache)
	fileIconCache := newFileIconCache(db, fileIcons)
	changed := make(chan struct{}, 1)
	service := &clipboard.Service{
		DB:       db,
		Images:   images,
		Settings: func() settings.Clipboard { return store.Snapshot().Clipboard },
		Guard:    clipboard.NewGuard(2 * time.Second),
		Writer:   pasteboardWriter{},
	}
	watcher := &clipboard.Watcher{
		Source: pasteboardSource{icons: iconCache},
		Paused: paused.Load,
		Capture: func(ctx context.Context, payload clipboard.Payload, source clipboard.SourceApp) {
			_, captured, err := service.Capture(ctx, payload, source)
			if err != nil {
				log.Printf("capture clipboard: %v", err)
			} else if captured {
				if store.Snapshot().Clipboard.Feedback.CopySound {
					macos.PlayCopySound()
				}
				select {
				case changed <- struct{}{}:
				default:
				}
			}
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		watcher.Run(ctx)
		close(done)
	}()
	runtime := &clipboardRuntime{
		db: db, service: service, images: images, appIcons: appIcons,
		fileIcons: fileIcons, fileIconCache: fileIconCache, paths: paths,
		apps: registry, watcher: watcher, changed: changed,
	}
	startCleanup(ctx, runtime, store)
	app.OnQuit(func(context.Context) {
		cancel()
		select {
		case <-done:
		case <-time.After(150 * time.Millisecond):
		}
		db.Close()
	})
	return runtime, nil
}

// pruneIcons removes cached icons no history item references anymore.
func pruneIcons(label string, referenced func(context.Context) (map[string]struct{}, error), purge func(map[string]struct{}) (int, int64, error)) {
	keep, err := referenced(context.Background())
	if err != nil {
		log.Printf("list %s: %v", label, err)
		return
	}
	if _, _, err := purge(keep); err != nil {
		log.Printf("clean %s: %v", label, err)
	}
}

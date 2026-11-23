package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/EcoPasteHub/EcoPaste/internal/storage"
)

// appRegistry merges persisted source apps with the applications that are
// running now, and owns icon extraction for apps added from disk.
type appRegistry struct {
	db           *database.DB
	icons        *appIconCache
	iconStore    *storage.AppIconStore
	running      func() []macos.RunningApp
	resolve      func(string) (macos.RunningApp, error)
	pathForID    func(string) string
	iconFromPath func(string) []byte
}

func newAppRegistry(db *database.DB, store *storage.AppIconStore, icons *appIconCache) *appRegistry {
	return &appRegistry{
		db: db, icons: icons, iconStore: store,
		running: macos.RunningApps, resolve: macos.ResolveApp,
		pathForID: macos.AppPathForBundleID, iconFromPath: macos.AppIconForPath,
	}
}

// List returns persisted apps, running apps and every required app that has
// not been stored. Running and required apps stay transient until copied or
// configured; requiring an id keeps default and configured apps in the list
// even when they are not running.
func (r *appRegistry) List(ctx context.Context, requiredIDs []string) ([]database.App, error) {
	known, err := r.db.ListApps(ctx)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]int, len(known))
	for i := range known {
		byID[known[i].ID] = i
		if known[i].IconFile == nil {
			known[i].IconFile = r.icons.File(known[i].ID)
		}
	}
	now := time.Now().UTC()
	for _, running := range r.running() {
		if running.BundleID == "" {
			continue
		}
		if index, ok := byID[running.BundleID]; ok {
			if known[index].IconFile == nil {
				known[index].IconFile = r.icons.File(running.BundleID)
			}
			continue
		}
		name := running.Name
		if name == "" {
			name = running.BundleID
		}
		known = append(known, database.App{
			ID: running.BundleID, Name: name, IconFile: r.icons.File(running.BundleID),
			CreatedAt: now, UpdatedAt: now,
		})
		byID[running.BundleID] = len(known) - 1
	}
	for _, id := range requiredIDs {
		if id == "" {
			continue
		}
		path := ""
		if r.pathForID != nil {
			path = r.pathForID(id)
		}
		if index, ok := byID[id]; ok {
			if known[index].IconFile == nil && path != "" {
				known[index].IconFile = r.icons.FileAtPath(id, path)
			}
			continue
		}
		name := id
		var iconFile *string
		if path != "" {
			if meta, err := r.resolve(path); err == nil && meta.Name != "" {
				name = meta.Name
			}
			iconFile = r.icons.FileAtPath(id, path)
		}
		known = append(known, database.App{
			ID: id, Name: name, IconFile: iconFile,
			CreatedAt: now, UpdatedAt: now,
		})
		byID[id] = len(known) - 1
	}
	sort.SliceStable(known, func(i, j int) bool {
		left := strings.ToLower(known[i].Name)
		right := strings.ToLower(known[j].Name)
		if left == right {
			return known[i].ID < known[j].ID
		}
		return left < right
	})
	return known, nil
}

// Ensure stores an application discovered from the running list without
// requiring its bundle path, so a configured rule keeps a row after it quits.
func (r *appRegistry) Ensure(ctx context.Context, id, name string) error {
	if id == "" {
		return nil
	}
	if name == "" {
		name = id
	}
	icon := r.icons.File(id)
	if icon == nil && r.pathForID != nil {
		if path := r.pathForID(id); path != "" {
			icon = r.icons.FileAtPath(id, path)
		}
	}
	if err := r.db.EnsureApp(ctx, id, name, icon); err != nil {
		return fmt.Errorf(i18n.T(i18n.ErrorSaveApp), err)
	}
	return nil
}

// Add registers an app bundle and returns the stored row. Re-adding the
// same bundle refreshes its name and icon without duplicating the id.
func (r *appRegistry) Add(ctx context.Context, path string) (database.App, error) {
	meta, err := r.resolve(path)
	if err != nil {
		return database.App{}, err
	}
	iconFile := r.storeIcon(path)
	if err := r.db.EnsureApp(ctx, meta.BundleID, meta.Name, iconFile); err != nil {
		return database.App{}, fmt.Errorf(i18n.T(i18n.ErrorSaveApp), err)
	}
	app, err := r.db.FindApp(ctx, meta.BundleID)
	if err != nil {
		return database.App{}, fmt.Errorf(i18n.T(i18n.ErrorReadApp), err)
	}
	return *app, nil
}

func (r *appRegistry) storeIcon(path string) *string {
	if r.iconFromPath == nil || r.iconStore == nil {
		return nil
	}
	return storeIconPNG(r.iconStore, r.iconFromPath(path), storage.AppIconMaxEdge)
}

// DeleteUnreferenced removes ignored apps that no history row uses and
// cleans up icons that became unused.
func (r *appRegistry) DeleteUnreferenced(ctx context.Context, ids []string) ([]string, error) {
	deleted, err := r.db.DeleteUnreferencedApps(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, id := range deleted {
		r.icons.Forget(id)
	}
	if len(deleted) > 0 {
		if err := r.pruneIcons(ctx); err != nil {
			return deleted, err
		}
	}
	return deleted, nil
}

func (r *appRegistry) pruneIcons(ctx context.Context) error {
	if r.iconStore == nil {
		return nil
	}
	referenced, err := r.db.AppIconFiles(ctx)
	if err != nil {
		return err
	}
	_, _, err = r.iconStore.RemoveUnreferenced(referenced)
	return err
}

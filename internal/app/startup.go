package app

import (
	"log"
	"os"
	"path/filepath"

	"github.com/EcoPasteHub/EcoPaste/internal/backup"
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/EcoPasteHub/EcoPaste/internal/logging"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
	"github.com/EcoPasteHub/EcoPaste/internal/storage"
	"github.com/nsgo-dev/nsgo/core"
)

// loadStorage resolves the data directory, takes the single-instance lock,
// opens logs and settings, and applies any staged backup import.
func loadStorage(app *core.Scope) (*storage.Paths, *settings.Store, *storage.WindowState) {
	paths, err := storage.Default()
	if err != nil {
		log.Printf("resolve storage: %v", err)
		return nil, nil, nil
	}
	lock, running, err := lockInstance(paths.State)
	switch {
	case err != nil:
		log.Printf("lock instance: %v", err)
	case running:
		if err := signalInstance(paths.State); err != nil {
			log.Printf("signal instance: %v", err)
		}
		os.Exit(0)
	default:
		app.Cleanup(func() { lock.Close() })
	}
	if logger, err := logging.Open(paths.Logs); err != nil {
		log.Printf("open logs: %v", err)
	} else {
		app.Cleanup(func() { logger.Close() })
	}
	if result, applied, err := backup.ApplyPendingImport(paths); err != nil {
		log.Printf("apply pending import: %v", err)
	} else if applied {
		log.Printf("imported backup: %d items", result.Items)
	}
	log.Printf("data root %s", paths.Root)

	var windowState *storage.WindowState
	if state, err := storage.OpenWindowState(paths.State); err != nil {
		log.Printf("open window state: %v", err)
	} else {
		windowState = state
	}
	store, err := settings.Open(filepath.Join(paths.Config, settings.FileName))
	if err != nil {
		log.Printf("open settings: %v", err)
		return paths, nil, windowState
	}
	return paths, store, windowState
}

func syncStoredSettings(app *core.Scope, store *settings.Store) {
	snapshot := store.Snapshot()
	applyAppLanguage(app, string(snapshot.Appearance.Language))
	if err := macos.SetAppearance(string(snapshot.Appearance.Theme)); err != nil {
		log.Printf("sync appearance: %v", err)
	}
	if err := macos.SyncAutostart(snapshot.General.AutoStart); err != nil {
		log.Printf("sync autostart: %v", err)
	}
}

// applyAppLanguage resolves the stored language for EcoPaste's own text and
// keeps AppKit's locale consistent. Auto clears the app-level override so
// native system text follows the user's languages instead of the previous
// explicit choice.
func applyAppLanguage(app *core.Scope, requested string) i18n.Language {
	resolved := i18n.ResolveLanguage(requested, macos.PreferredLanguage())
	i18n.Publish(resolved)
	if requested == "" || requested == string(i18n.LanguageAuto) {
		app.SetLocale("")
	} else {
		app.SetLocale(string(resolved))
	}
	return resolved
}

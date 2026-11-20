package app

import (
	"log/slog"
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
func loadStorage(app *core.Scope) (*storage.Paths, *settings.Store, *storage.WindowState, *logging.Logger) {
	paths, err := storage.Default()
	if err != nil {
		slog.Error("resolve storage", "module", "app.startup", "error", err)
		return nil, nil, nil, nil
	}
	lock, running, err := lockInstance(paths.State)
	switch {
	case err != nil:
		slog.Error("lock instance", "module", "app.startup", "error", err)
	case running:
		if err := signalInstance(paths.State); err != nil {
			slog.Error("signal existing instance", "module", "app.startup", "error", err)
		}
		os.Exit(0)
	default:
		app.Cleanup(func() { lock.Close() })
	}
	var logger *logging.Logger
	if logger, err = logging.Open(paths.Logs); err != nil {
		slog.Error("open log files", "module", "app.startup", "error", err)
	} else {
		app.Cleanup(func() { logger.Close() })
	}
	if result, applied, err := backup.ApplyPendingImport(paths); err != nil {
		slog.Error("apply pending backup import", "module", "app.startup", "error", err)
	} else if applied {
		slog.Info("imported backup", "module", "app.startup", "items", result.Items)
	}
	slog.Info("application storage ready", "module", "app.startup")

	var windowState *storage.WindowState
	if state, err := storage.OpenWindowState(paths.State); err != nil {
		slog.Error("open window state", "module", "app.startup", "error", err)
	} else {
		windowState = state
	}
	store, err := settings.Open(filepath.Join(paths.Config, settings.FileName))
	if err != nil {
		slog.Error("open settings", "module", "app.startup", "error", err)
		return paths, nil, windowState, logger
	}
	if logger != nil {
		config := store.Snapshot().Logging
		if err := logger.Apply(logging.Level(config.Level), config.RetentionDays); err != nil {
			slog.Error("apply logging settings", "module", "app.startup", "error", err)
		}
	}
	return paths, store, windowState, logger
}

func syncStoredSettings(app *core.Scope, store *settings.Store) {
	snapshot := store.Snapshot()
	applyAppLanguage(app, string(snapshot.Appearance.Language))
	if err := macos.SetAppearance(string(snapshot.Appearance.Theme)); err != nil {
		slog.Error("sync appearance", "module", "app.startup", "error", err)
	}
	if err := macos.SyncAutostart(snapshot.General.AutoStart); err != nil {
		slog.Error("sync autostart", "module", "app.startup", "error", err)
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

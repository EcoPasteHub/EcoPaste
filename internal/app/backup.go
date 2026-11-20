package app

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/backup"
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/EcoPasteHub/EcoPaste/internal/logging"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
)

// exportBackupWithUI runs the export chooser, password prompt and save
// panel shared by the tray and settings window.
func exportBackupWithUI(app *core.Scope, runtime *clipboardRuntime) {
	if runtime == nil {
		return
	}
	mode := backup.ModePlain
	password := ""
	switch macos.ChooseBackupExport() {
	case macos.BackupExportCancel:
		return
	case macos.BackupExportEncrypted:
		var confirmed bool
		password, confirmed = macos.PromptPassword(i18n.T(i18n.BackupPasswordTitle),
			i18n.T(i18n.BackupPasswordBody))
		if !confirmed {
			return
		}
		if password == "" {
			macos.ShowMessage(i18n.T(i18n.BackupExportFailed), i18n.T(i18n.BackupPasswordRequired))
			return
		}
		mode = backup.ModeEncrypted
	case macos.BackupExportPlain:
		if !macos.Confirm(i18n.T(i18n.BackupUnencryptedTitle),
			i18n.T(i18n.BackupUnencryptedBody), i18n.T(i18n.BackupUnencryptedConfirm)) {
			return
		}
	}
	path, ok := macos.SavePanel(
		fmt.Sprintf("EcoPaste-%s.ecopastebak", time.Now().Format("2006-01-02")),
		i18n.T(i18n.BackupExportPanelTitle))
	if !ok {
		return
	}
	runBackupExport(app, runtime, path, mode, password)
}

// importBackupWithUI asks for an archive and applies the shared import
// flow.
func importBackupWithUI(app *core.Scope, runtime *clipboardRuntime, store *settings.Store, reloadHotkeys func() error) {
	if runtime == nil {
		return
	}
	path, ok := macos.OpenBackupPanel(i18n.T(i18n.BackupImportPanelTitle))
	if !ok {
		return
	}
	runBackupImport(app, runtime, path, store, reloadHotkeys)
}

func runBackupExport(app *core.Scope, runtime *clipboardRuntime, path string, mode backup.Mode, password string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		result, err := backup.Export(ctx, backup.Source{
			Database:   runtime.db,
			Resources:  runtime.paths.Resources,
			Settings:   filepath.Join(runtime.paths.Config, settings.FileName),
			AppName:    "EcoPaste",
			AppVersion: macos.AppVersion(),
		}, path, backup.ExportOptions{Mode: mode, Password: password})
		if err != nil {
			slog.Error("export backup", "module", "app.backup", "error", err)
		} else {
			slog.Info("exported backup", "module", "app.backup", "items", result.Items, "bytes", result.TotalBytes)
		}
		app.Post(func() {
			if err != nil {
				macos.ShowMessage(i18n.T(i18n.BackupExportFailed), err.Error())
				return
			}
			macos.ShowMessage(i18n.T(i18n.BackupExportedTitle),
				i18n.Tf(i18n.BackupExportedBody, result.Path, result.Items, formatBytes(result.TotalBytes)))
		})
	}()
}

// runBackupImport asks how to apply a backup. Merge updates the running app;
// overwrite stages a replacement for the next launch.
func runBackupImport(app *core.Scope, runtime *clipboardRuntime, path string, store *settings.Store, reloadHotkeys func() error) {
	if runtime == nil {
		return
	}
	mode, err := backup.Inspect(path)
	if err != nil {
		slog.Error("inspect backup", "module", "app.backup", "error", err)
		macos.ShowMessage(i18n.T(i18n.BackupImportFailed), err.Error())
		return
	}
	password := ""
	if mode == backup.ModeEncrypted {
		var confirmed bool
		password, confirmed = macos.PromptPassword(i18n.T(i18n.BackupPasswordTitle), i18n.T(i18n.BackupPasswordBody))
		if !confirmed {
			return
		}
	}
	switch macos.ChooseBackupImport() {
	case macos.BackupImportCancel:
		return
	case macos.BackupImportMerge:
		runBackupMerge(app, runtime, path, password, store, reloadHotkeys)
		return
	}
	if !macos.Confirm(i18n.T(i18n.BackupOverwriteTitle),
		i18n.T(i18n.BackupOverwriteBody), i18n.T(i18n.BackupOverwriteConfirm)) {
		return
	}
	if _, err := backup.StageImport(path, password, runtime.paths.State); err != nil {
		slog.Error("stage backup import", "module", "app.backup", "error", err)
		macos.ShowMessage(i18n.T(i18n.BackupImportFailed), err.Error())
		return
	}
	desktop.Restart(app)
}

func runBackupMerge(app *core.Scope, runtime *clipboardRuntime, path, password string, store *settings.Store, reloadHotkeys func() error) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		result, err := backup.Merge(ctx, path, password, runtime.db, runtime.paths.Resources)
		if err != nil {
			slog.Error("merge backup", "module", "app.backup", "error", err)
			app.Post(func() {
				macos.ShowMessage(i18n.T(i18n.BackupMergeFailed), err.Error())
			})
			return
		}
		var (
			next        settings.Settings
			settingsErr error
		)
		if store != nil && len(result.Settings) > 0 {
			next, settingsErr = store.Update(result.Settings)
			if settingsErr == nil {
				settingsErr = logging.Configure(logging.Level(next.Logging.Level))
			}
			if settingsErr != nil {
				slog.Error("apply imported settings", "module", "app.backup", "error", settingsErr)
			}
		}
		slog.Info("merged backup", "module", "app.backup", "items", result.Items, "skipped", result.Skipped,
			"resources", result.Resources)
		app.Post(func() {
			switch {
			case settingsErr != nil:
				macos.ShowMessage(i18n.T(i18n.BackupMergedNoSettings), settingsErr.Error())
			case store == nil:
				macos.ShowMessage(i18n.T(i18n.BackupMergedTitle),
					mergeSummary(result)+"\n"+i18n.T(i18n.BackupMergedSettingsNote))
			default:
				if err := macos.SetAppearance(string(next.Appearance.Theme)); err != nil {
					slog.Error("sync imported appearance", "module", "app.backup", "error", err)
				}
				if err := macos.SyncAutostart(next.General.AutoStart); err != nil {
					slog.Error("sync imported autostart setting", "module", "app.backup", "error", err)
				}
				summary := mergeSummary(result)
				if reloadHotkeys != nil {
					if err := reloadHotkeys(); err != nil {
						summary += "\n" + i18n.T(i18n.BackupShortcutFailed) + err.Error()
					}
				}
				macos.ShowMessage(i18n.T(i18n.BackupMergedDoneTitle), summary)
			}
			runtime.Signal()
		})
	}()
}

func mergeSummary(result backup.MergeResult) string {
	return i18n.Tf(i18n.BackupMergeSummary, result.Items, result.Skipped, result.Resources)
}

// firstBackupPath picks the first .ecopastebak file the system opened.
func firstBackupPath(paths []string) (string, bool) {
	for _, path := range paths {
		if strings.EqualFold(filepath.Ext(path), ".ecopastebak") {
			return path, true
		}
	}
	return "", false
}

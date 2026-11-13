package app

import (
	"context"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/backup"
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
)

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
		}
		app.Post(func() {
			if settingsErr != nil {
				macos.ShowMessage(i18n.T(i18n.BackupMergedNoSettings), settingsErr.Error())
			} else if store == nil {
				macos.ShowMessage(i18n.T(i18n.BackupMergedTitle),
					mergeSummary(result)+"\n"+i18n.T(i18n.BackupMergedSettingsNote))
				runtime.Signal()
				return
			}
			if settingsErr == nil && store != nil {
				if err := macos.SetAppearance(string(next.Appearance.Theme)); err != nil {
					log.Printf("sync appearance: %v", err)
				}
				if err := macos.SetDockIconVisible(next.General.DockIcon); err != nil {
					log.Printf("sync dock icon: %v", err)
				}
				if err := macos.SyncAutostart(next.General.AutoStart); err != nil {
					log.Printf("sync autostart: %v", err)
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

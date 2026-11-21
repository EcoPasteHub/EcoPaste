package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
	"github.com/EcoPasteHub/EcoPaste/internal/update"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
)

type trayMenu struct {
	app           *core.Scope
	runtime       *clipboardRuntime
	store         *settings.Store
	showPanel     func()
	openSettings  func()
	shortcuts     func() (string, string)
	reloadHotkeys func() error
}

func (t trayMenu) items() []desktop.MenuItem {
	hotKey, preferenceHotKey := t.shortcuts()
	pauseLabel := i18n.T(i18n.TrayPause)
	if t.runtime != nil && t.runtime.watcher.Paused() {
		pauseLabel = i18n.T(i18n.TrayResume)
	}
	return []desktop.MenuItem{
		{Label: i18n.T(i18n.TrayShow), Shortcut: hotKey, OnSelect: t.showPanel},
		{Label: pauseLabel, Disabled: t.runtime == nil, OnSelect: t.togglePause},
		{Label: i18n.T(i18n.TrayReadClipboard), OnSelect: t.readClipboard},
		{Label: i18n.T(i18n.TraySettings), Shortcut: preferenceHotKey, OnSelect: t.openSettings},
		{Separator: true},
		{Label: i18n.T(i18n.TrayExportBackup), OnSelect: func() {
			exportBackupWithUI(t.app, t.runtime)
		}},
		{Label: i18n.T(i18n.TrayImportBackup), OnSelect: func() {
			importBackupWithUI(t.app, t.runtime, t.store, t.reloadHotkeys)
		}},
		{Label: i18n.T(i18n.TrayClearHistory), OnSelect: t.clearHistory},
		{Separator: true},
		{Label: i18n.T(i18n.TrayStorageUsage), OnSelect: t.showStorageUsage},
		{Label: i18n.T(i18n.TrayCleanCache), OnSelect: t.cleanCache},
		{Label: i18n.T(i18n.TrayOpenDataDir), OnSelect: func() {
			if t.runtime == nil {
				return
			}
			if err := macos.OpenPath(t.runtime.paths.Root); err != nil {
				macos.ShowMessage(i18n.T(i18n.OpenDataDirFailed), err.Error())
			}
		}},
		{Label: i18n.T(i18n.TrayOpenLogs), OnSelect: func() {
			if t.runtime == nil {
				return
			}
			if err := macos.OpenPath(t.runtime.paths.Logs); err != nil {
				macos.ShowMessage(i18n.T(i18n.OpenLogsFailed), err.Error())
			}
		}},
		{Separator: true},
		{Label: i18n.T(i18n.TraySource), OnSelect: func() {
			if err := macos.OpenURL(macos.SourceURL); err != nil {
				slog.Error("open source page", "module", "app.tray", "error", err)
			}
		}},
		{Label: i18n.T(i18n.TrayCheckUpdates), OnSelect: func() { update.CheckNow(t.app) }},
		{Label: i18n.Tf(i18n.TrayVersion, macos.AppVersion()), Disabled: true},
		{Label: i18n.T(i18n.TrayRestart), OnSelect: func() { desktop.Restart(t.app) }},
		{Separator: true},
		{Label: i18n.T(i18n.TrayQuit), Shortcut: "mod+q", OnSelect: t.app.Quit},
	}
}

func (t trayMenu) togglePause() {
	if t.runtime != nil {
		watcher := t.runtime.watcher
		watcher.SetPaused(!watcher.Paused())
	}
}

func (t trayMenu) readClipboard() {
	if t.runtime == nil {
		return
	}
	parent := t.runtime.ctx
	go func() {
		ctx, cancel := context.WithTimeout(parent, 30*time.Second)
		defer cancel()
		if err := t.runtime.readClipboard(ctx); err != nil {
			slog.Error("read clipboard", "module", "app.tray", "error", err)
			return
		}
	}()
}

func (t trayMenu) clearHistory() {
	choice := macos.ConfirmClearHistory()
	if choice == macos.ClearHistoryCancel || t.runtime == nil {
		return
	}
	if err := t.runtime.clearHistory(context.Background(), choice == macos.ClearHistoryAll); err != nil {
		slog.Error("clear history", "module", "app.tray", "error", err)
	}
}

func (t trayMenu) showStorageUsage() {
	if t.runtime == nil {
		return
	}
	usage, err := t.runtime.storageUsage()
	if err != nil {
		macos.ShowMessage(i18n.T(i18n.StorageUsageFailed), err.Error())
		return
	}
	macos.ShowMessage(i18n.T(i18n.StorageUsageTitle),
		i18n.Tf(i18n.StorageUsageBody,
			formatBytes(usage.Database), formatBytes(usage.Resource),
			formatBytes(usage.Config), formatBytes(usage.Total)))
}

func (t trayMenu) cleanCache() {
	if t.runtime == nil || !macos.Confirm(i18n.T(i18n.CleanCacheTitle),
		i18n.T(i18n.CleanCacheBody), i18n.T(i18n.CleanCacheConfirm)) {
		return
	}
	files, freed, err := t.runtime.cleanCache(context.Background())
	if err != nil {
		macos.ShowMessage(i18n.T(i18n.CleanCacheFailed), err.Error())
		return
	}
	if files == 0 {
		macos.ShowMessage(i18n.T(i18n.CleanCacheEmptyTitle), i18n.T(i18n.CleanCacheEmptyBody))
		return
	}
	macos.ShowMessage(i18n.T(i18n.CleanCacheDoneTitle),
		i18n.Tf(i18n.CleanCacheDoneBody, files, formatBytes(freed)))
}

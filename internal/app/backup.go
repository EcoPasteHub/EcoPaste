package app

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/backup"
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
				macos.ShowMessage("导出备份失败", err.Error())
				return
			}
			macos.ShowMessage("备份已导出",
				fmt.Sprintf("%s\n%d 条记录 · %s", result.Path, result.Items, formatBytes(result.TotalBytes)))
		})
	}()
}

// runBackupImport asks how to apply a backup. Merge updates the running app;
// overwrite stages a replacement for the next launch.
func runBackupImport(app *core.Scope, runtime *clipboardRuntime, path string, store *settings.Store, reloadHotkeys func()) {
	if runtime == nil {
		return
	}
	mode, err := backup.Inspect(path)
	if err != nil {
		macos.ShowMessage("无法导入备份", err.Error())
		return
	}
	password := ""
	if mode == backup.ModeEncrypted {
		var confirmed bool
		password, confirmed = macos.PromptPassword("备份密码", "请输入备份文件的密码。")
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
	if !macos.Confirm("覆盖当前数据？",
		"覆盖导入会替换当前历史、设置和资源，然后重启 EcoPaste。", "导入并重启") {
		return
	}
	if _, err := backup.StageImport(path, password, runtime.paths.State); err != nil {
		macos.ShowMessage("无法导入备份", err.Error())
		return
	}
	desktop.Restart(app)
}

func runBackupMerge(app *core.Scope, runtime *clipboardRuntime, path, password string, store *settings.Store, reloadHotkeys func()) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		result, err := backup.Merge(ctx, path, password, runtime.db, runtime.paths.Resources)
		if err != nil {
			app.Post(func() {
				macos.ShowMessage("无法合并备份", err.Error())
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
				macos.ShowMessage("备份已合并，设置未应用", settingsErr.Error())
			} else if store == nil {
				macos.ShowMessage("备份已合并", mergeSummary(result)+"\n当前设置未更新。")
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
				if reloadHotkeys != nil {
					reloadHotkeys()
				}
				macos.ShowMessage("备份合并完成", mergeSummary(result))
			}
			runtime.Signal()
		})
	}()
}

func mergeSummary(result backup.MergeResult) string {
	return fmt.Sprintf("已合并 %d 条记录，跳过 %d 条重复记录，补充 %d 个资源文件。",
		result.Items, result.Skipped, result.Resources)
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

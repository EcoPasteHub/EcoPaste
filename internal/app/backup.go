package app

import (
	"context"
	"fmt"
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

// runBackupImport stages path for the next launch, asking for the password
// and overwrite confirmation first.
func runBackupImport(app *core.Scope, runtime *clipboardRuntime, path string) {
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
	if !macos.Confirm("覆盖当前数据？",
		"导入会覆盖当前历史、设置和资源，然后重启 EcoPaste。", "导入并重启") {
		return
	}
	if _, err := backup.StageImport(path, password, runtime.paths.State); err != nil {
		macos.ShowMessage("无法导入备份", err.Error())
		return
	}
	desktop.Restart(app)
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

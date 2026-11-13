package app

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/backup"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
	"github.com/nsgo-dev/nsgo/core"
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

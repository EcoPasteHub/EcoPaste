package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/storage"
)

const customDataDirName = "EcoPasteData"

// changeDataDir copies the live data into parent/EcoPasteData/<mode> and
// repoints the bootstrap manifest. The process keeps using the old root
// until it restarts.
func (r *clipboardRuntime) changeDataDir(ctx context.Context, parent string) (*storage.Paths, error) {
	if strings.TrimSpace(parent) == "" {
		return nil, errors.New("未选择数据目录")
	}
	target := filepath.Join(parent, customDataDirName, storage.Mode())
	if filepath.Clean(target) == filepath.Clean(r.paths.Root) {
		return nil, errors.New("当前已使用该目录")
	}
	if _, err := os.Stat(target); err == nil {
		return nil, fmt.Errorf("目标目录已存在：%s", target)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("检查目标目录: %w", err)
	}
	if err := storage.ValidateTarget(target, storage.Mode()); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(target, "db"), 0o755); err != nil {
		return nil, fmt.Errorf("创建目标目录: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			os.RemoveAll(target)
		}
	}()
	if err := r.copyLiveData(ctx, target); err != nil {
		return nil, err
	}
	if err := storage.CommitRoot(r.paths.Base, storage.Mode(), target); err != nil {
		return nil, err
	}
	committed = true
	paths, err := storage.Resolve(r.paths.Base, storage.Mode())
	if err != nil {
		return nil, err
	}
	return paths, nil
}

// restoreDefaultDataDir copies the live data back to the bootstrap
// directory. The previous default directory stays beside it as a
// timestamped backup.
func (r *clipboardRuntime) restoreDefaultDataDir(ctx context.Context) (*storage.Paths, error) {
	bootstrap := r.paths.Bootstrap
	current, err := storage.Resolve(r.paths.Base, storage.Mode())
	if err != nil {
		return nil, err
	}
	if filepath.Clean(bootstrap) == filepath.Clean(current.Root) {
		return nil, errors.New("当前已使用默认数据目录")
	}
	staging, err := os.MkdirTemp(r.paths.Base, ".restore-default-")
	if err != nil {
		return nil, fmt.Errorf("创建临时目录: %w", err)
	}
	var backup string
	bootstrapMoved, swapped, committed := false, false, false
	defer func() {
		if committed {
			return
		}
		if swapped {
			os.RemoveAll(bootstrap)
		}
		if bootstrapMoved {
			os.Rename(backup, bootstrap)
		}
		os.RemoveAll(staging)
	}()
	if err := r.copyLiveData(ctx, staging); err != nil {
		return nil, err
	}
	backup, err = backupDataDirName(r.paths.Base, storage.Mode())
	if err != nil {
		return nil, err
	}
	if err := os.Rename(bootstrap, backup); err != nil {
		return nil, fmt.Errorf("备份默认数据目录: %w", err)
	}
	bootstrapMoved = true
	if err := os.Rename(staging, bootstrap); err != nil {
		return nil, fmt.Errorf("恢复默认数据目录: %w", err)
	}
	swapped = true
	if err := storage.CommitRoot(r.paths.Base, storage.Mode(), bootstrap); err != nil {
		return nil, err
	}
	committed = true
	paths, err := storage.Resolve(r.paths.Base, storage.Mode())
	if err != nil {
		return nil, err
	}
	return paths, nil
}

func (r *clipboardRuntime) copyLiveData(ctx context.Context, target string) error {
	if err := os.MkdirAll(filepath.Join(target, "db"), 0o755); err != nil {
		return fmt.Errorf("创建目标目录: %w", err)
	}
	for _, part := range []struct{ src, dst string }{
		{r.paths.Resources, filepath.Join(target, "resources")},
		{r.paths.Config, filepath.Join(target, "config")},
		{r.paths.State, filepath.Join(target, "state")},
	} {
		if err := copyTree(part.src, part.dst, skipRuntimeState); err != nil {
			return err
		}
	}
	if err := r.db.Snapshot(ctx, filepath.Join(target, "db", "clipboard.db")); err != nil {
		return err
	}
	return nil
}

func backupDataDirName(base, environment string) (string, error) {
	stamp := time.Now().Format("20060102-150405")
	for suffix := 0; ; suffix++ {
		name := filepath.Join(base, fmt.Sprintf("%s-backup-%s", environment, stamp))
		if suffix > 0 {
			name = fmt.Sprintf("%s-%d", name, suffix)
		}
		if _, err := os.Stat(name); errors.Is(err, os.ErrNotExist) {
			return name, nil
		} else if err != nil {
			return "", fmt.Errorf("检查备份目录: %w", err)
		}
	}
}

func skipRuntimeState(name string) bool {
	switch name {
	case "instance.lock", "import-pending", "import-rollback":
		return true
	}
	return strings.HasPrefix(name, ".import-")
}

func copyTree(src, dst string, skip func(string) bool) error {
	info, err := os.Stat(src)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("读取数据目录 %s: %w", src, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("数据路径 %s 不是目录", src)
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("读取数据目录 %s: %w", src, err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return fmt.Errorf("创建数据目录 %s: %w", dst, err)
	}
	for _, entry := range entries {
		if skip != nil && skip(entry.Name()) {
			continue
		}
		from := filepath.Join(src, entry.Name())
		to := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyTree(from, to, nil); err != nil {
				return err
			}
			continue
		}
		if !entry.Type().IsRegular() {
			continue
		}
		data, err := os.ReadFile(from)
		if err != nil {
			return fmt.Errorf("读取数据文件 %s: %w", from, err)
		}
		if err := os.WriteFile(to, data, 0o644); err != nil {
			return fmt.Errorf("写入数据文件 %s: %w", to, err)
		}
	}
	return nil
}

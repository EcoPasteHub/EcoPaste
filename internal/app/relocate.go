package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
		return nil, errors.New("目标目录已存在，请选择其他位置")
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
	for _, part := range []struct{ src, dst string }{
		{r.paths.Resources, filepath.Join(target, "resources")},
		{r.paths.Config, filepath.Join(target, "config")},
		{r.paths.State, filepath.Join(target, "state")},
	} {
		if err := copyTree(part.src, part.dst, skipRuntimeState); err != nil {
			return nil, err
		}
	}
	if err := r.db.Snapshot(ctx, filepath.Join(target, "db", "clipboard.db")); err != nil {
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

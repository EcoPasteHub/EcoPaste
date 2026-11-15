package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/EcoPasteHub/EcoPaste/internal/storage"
)

const customDataDirName = "EcoPasteData"

// changeDataDir copies the live data into parent/EcoPasteData/<mode> and
// repoints the bootstrap manifest. The process keeps using the old root
// until it restarts.
func (r *clipboardRuntime) changeDataDir(ctx context.Context, parent string) (*storage.Paths, error) {
	if strings.TrimSpace(parent) == "" {
		return nil, errors.New(i18n.T(i18n.ErrorDataDirNotSelected))
	}
	target := filepath.Join(parent, customDataDirName, storage.Mode())
	if filepath.Clean(target) == filepath.Clean(r.paths.Root) {
		return nil, errors.New(i18n.T(i18n.ErrorDataDirInUse))
	}
	if _, err := os.Stat(target); err == nil {
		return nil, fmt.Errorf(i18n.T(i18n.ErrorDataDirExists), target)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf(i18n.T(i18n.ErrorCheckTargetDir), err)
	}
	if err := storage.ValidateTarget(target, storage.Mode()); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(target, "db"), 0o755); err != nil {
		return nil, fmt.Errorf(i18n.T(i18n.ErrorCreateTargetDir), err)
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
		return nil, errors.New(i18n.T(i18n.ErrorDefaultDataDirInUse))
	}
	staging, err := os.MkdirTemp(r.paths.Base, ".restore-default-")
	if err != nil {
		return nil, fmt.Errorf(i18n.T(i18n.ErrorCreateTempDir), err)
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
		return nil, fmt.Errorf(i18n.T(i18n.ErrorBackupDefaultDir), err)
	}
	bootstrapMoved = true
	if err := os.Rename(staging, bootstrap); err != nil {
		return nil, fmt.Errorf(i18n.T(i18n.ErrorRestoreDefaultDir), err)
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
		return fmt.Errorf(i18n.T(i18n.ErrorCreateTargetDir), err)
	}
	for _, part := range []struct{ src, dst string }{
		{r.paths.Resources, filepath.Join(target, "resources")},
		{r.paths.Config, filepath.Join(target, "config")},
		{r.paths.State, filepath.Join(target, "state")},
	} {
		if err := copyTree(ctx, part.src, part.dst, skipRuntimeState); err != nil {
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
			return "", fmt.Errorf(i18n.T(i18n.ErrorCheckBackupDir), err)
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

func copyTree(ctx context.Context, src, dst string, skip func(string) bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	info, err := os.Stat(src)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf(i18n.T(i18n.ErrorReadDataDir), src, err)
	}
	if !info.IsDir() {
		return fmt.Errorf(i18n.T(i18n.ErrorDataPathNotDir), src)
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf(i18n.T(i18n.ErrorReadDataDir), src, err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return fmt.Errorf(i18n.T(i18n.ErrorCreateDataDir), dst, err)
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if skip != nil && skip(entry.Name()) {
			continue
		}
		from := filepath.Join(src, entry.Name())
		to := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyTree(ctx, from, to, nil); err != nil {
				return err
			}
			continue
		}
		if !entry.Type().IsRegular() {
			continue
		}
		if err := copyFileContents(ctx, from, to); err != nil {
			return err
		}
	}
	return nil
}

// copyFileContents streams one file so moving a data root does not load
// whole image and database files into memory.
func copyFileContents(ctx context.Context, src, dst string) error {
	input, err := os.Open(src)
	if err != nil {
		return fmt.Errorf(i18n.T(i18n.ErrorReadDataFile), src, err)
	}
	defer input.Close()
	output, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf(i18n.T(i18n.ErrorWriteDataFile), dst, err)
	}
	if _, err := io.Copy(output, contextReader{ctx: ctx, reader: input}); err != nil {
		output.Close()
		return fmt.Errorf(i18n.T(i18n.ErrorWriteDataFile), dst, err)
	}
	if err := output.Close(); err != nil {
		return fmt.Errorf(i18n.T(i18n.ErrorWriteDataFile), dst, err)
	}
	return nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(buffer []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(buffer)
}

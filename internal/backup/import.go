package backup

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/renameio/v2"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/EcoPasteHub/EcoPaste/internal/storage"
)

// ImportPlan summarizes a backup before it is applied.
type ImportPlan struct {
	Mode          Mode
	Items         int64
	Texts         int64
	Images        int64
	Files         int64
	ResourceBytes int64
	AppVersion    string
	ExportedAt    string
}

// ImportResult reports a completed overwrite import.
type ImportResult struct {
	Items int64
}

// MergeResult reports a live merge import. Settings is the backup's JSON
// patch and is applied by the app layer after the history merge succeeds.
type MergeResult struct {
	Items     int64
	Skipped   int64
	Resources int
	Settings  map[string]any
}

// Inspect identifies the backup format without decrypting its contents.
func Inspect(path string) (Mode, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf(i18n.T(i18n.BackupReadFile), err)
	}
	defer file.Close()
	head := make([]byte, len(magic))
	n, err := io.ReadFull(file, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf(i18n.T(i18n.BackupReadFile), err)
	}
	if n >= 2 && head[0] == 'P' && head[1] == 'K' {
		info, err := file.Stat()
		if err != nil {
			return "", fmt.Errorf(i18n.T(i18n.BackupReadFile), err)
		}
		reader, err := zip.NewReader(file, info.Size())
		if err != nil {
			return "", fmt.Errorf(i18n.T(i18n.BackupReadContents), err)
		}
		if _, err := readZIPManifest(reader); err != nil {
			return "", err
		}
		return ModePlain, nil
	}
	if n < len(magic) || string(head) != magic {
		return "", errors.New(i18n.T(i18n.BackupInvalidFile))
	}
	if _, err := readEncryptedHeader(file); err != nil {
		return "", err
	}
	return ModeEncrypted, nil
}

// StageImport validates path and leaves an extracted copy under the
// state directory for the next launch to swap in.
func StageImport(ctx context.Context, path, password, stateDir string) (ImportPlan, error) {
	var plan ImportPlan
	temp, manifest, err := extractImport(path, password, stateDir, ".import-*")
	if err != nil {
		return plan, err
	}
	defer os.RemoveAll(temp)
	if err := validateSnapshot(ctx, temp); err != nil {
		return plan, err
	}

	pending := filepath.Join(stateDir, storage.PendingBackupDir)
	if err := os.RemoveAll(pending); err != nil {
		return plan, fmt.Errorf(i18n.T(i18n.BackupCleanPending), err)
	}
	if err := os.Rename(temp, pending); err != nil {
		return plan, fmt.Errorf(i18n.T(i18n.BackupStagePending), err)
	}
	plan = ImportPlan{
		Mode: ModePlain, Items: manifest.ItemCount, Texts: manifest.TextCount,
		Images: manifest.ImageCount, Files: manifest.FilesCount,
		ResourceBytes: manifest.ResourceBytes, AppVersion: manifest.AppVersion,
		ExportedAt: manifest.ExportedAt.Format("2006-01-02 15:04"),
	}
	if manifest.Encryption == "password" {
		plan.Mode = ModeEncrypted
	}
	return plan, nil
}

// Merge imports a backup into the live database without replacing
// existing history. Missing resources are copied and duplicate items keep
// their local metadata.
func Merge(ctx context.Context, path, password string, target *database.DB, resourcesRoot string) (MergeResult, error) {
	var result MergeResult
	if target == nil {
		return result, errors.New(i18n.T(i18n.BackupMissingCurrentDB))
	}
	root, _, err := extractImport(path, password, "", ".import-*")
	if err != nil {
		return result, err
	}
	defer os.RemoveAll(root)

	settingsPatch, err := readSettingsPatch(filepath.Join(root, "config", "settings.json"))
	if err != nil {
		return result, err
	}
	backupDB, err := openSnapshot(ctx, root)
	if err != nil {
		return result, err
	}
	defer backupDB.Close()
	resources, err := mergeResources(ctx, filepath.Join(root, "resources"), resourcesRoot)
	if err != nil {
		return result, err
	}
	merged, err := target.MergeBackup(ctx, backupDB)
	if err != nil {
		return result, err
	}
	result.Items = merged.Items
	result.Skipped = merged.Skipped
	result.Resources = resources
	result.Settings = settingsPatch
	return result, nil
}

func openSnapshot(ctx context.Context, root string) (*database.DB, error) {
	db, err := database.OpenSnapshot(ctx, filepath.Join(root, "db", "clipboard.db"))
	if err != nil {
		return nil, fmt.Errorf(i18n.T(i18n.BackupOpenDatabase), err)
	}
	return db, nil
}

func validateSnapshot(ctx context.Context, root string) error {
	db, err := openSnapshot(ctx, root)
	if err != nil {
		return err
	}
	return db.Close()
}

func extractImport(path, password, parent, pattern string) (string, backupManifest, error) {
	var manifest backupManifest
	payload, err := readPayload(path, password)
	if err != nil {
		return "", manifest, err
	}
	reader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		return "", manifest, fmt.Errorf(i18n.T(i18n.BackupReadContents), err)
	}
	manifest, err = readZIPManifest(reader)
	if err != nil {
		return "", manifest, err
	}
	if parent != "" {
		if err := os.MkdirAll(parent, 0o755); err != nil {
			return "", manifest, fmt.Errorf(i18n.T(i18n.BackupCreateImportDir), err)
		}
	}
	temp, err := os.MkdirTemp(parent, pattern)
	if err != nil {
		return "", manifest, fmt.Errorf(i18n.T(i18n.BackupCreateImportTemp), err)
	}
	extracted := false
	defer func() {
		if !extracted {
			os.RemoveAll(temp)
		}
	}()
	if err := extractPayload(reader, temp); err != nil {
		return "", manifest, err
	}
	if _, err := os.Stat(filepath.Join(temp, "db", "clipboard.db")); err != nil {
		return "", manifest, errors.New(i18n.T(i18n.BackupMissingHistory))
	}
	if _, err := os.Stat(filepath.Join(temp, "config", "settings.json")); err != nil {
		return "", manifest, errors.New(i18n.T(i18n.BackupMissingSettings))
	}
	extracted = true
	return temp, manifest, nil
}

func readZIPManifest(reader *zip.Reader) (backupManifest, error) {
	var manifest backupManifest
	file, err := reader.Open("manifest.json")
	if errors.Is(err, os.ErrNotExist) {
		return manifest, errors.New(i18n.T(i18n.BackupMissingManifest))
	}
	if err != nil {
		return manifest, fmt.Errorf(i18n.T(i18n.BackupReadManifest), err)
	}
	defer file.Close()
	const maxManifestSize = 64 * 1024
	data, err := io.ReadAll(io.LimitReader(file, maxManifestSize+1))
	if err != nil {
		return manifest, fmt.Errorf(i18n.T(i18n.BackupReadManifest), err)
	}
	if len(data) > maxManifestSize {
		return manifest, errors.New(i18n.T(i18n.BackupInvalidFile))
	}
	return parseManifest(data)
}

func parseManifest(data []byte) (backupManifest, error) {
	var manifest backupManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return manifest, fmt.Errorf(i18n.T(i18n.BackupParseManifest), err)
	}
	if manifest.FormatVersion != formatVersion {
		return manifest, fmt.Errorf(i18n.T(i18n.BackupUnsupportedVersion), manifest.FormatVersion)
	}
	return manifest, nil
}

func extractPayload(reader *zip.Reader, root string) error {
	for _, entry := range reader.File {
		name, err := safeArchiveName(entry.Name)
		if err != nil {
			return err
		}
		switch {
		case name == "manifest.json":
		case name == "db/clipboard.db":
		case name == "config/settings.json":
		case strings.HasPrefix(name, "resources/"):
		default:
			continue
		}
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf(i18n.T(i18n.BackupCreateImportDir), err)
		}
		if err := extractEntry(entry, path); err != nil {
			return err
		}
	}
	return nil
}

func readSettingsPatch(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf(i18n.T(i18n.BackupReadSettings), err)
	}
	var patch map[string]any
	if err := json.Unmarshal(data, &patch); err != nil {
		return nil, fmt.Errorf(i18n.T(i18n.BackupParseSettings), err)
	}
	if patch == nil {
		return nil, errors.New(i18n.T(i18n.BackupSettingsObject))
	}
	return patch, nil
}

func mergeResources(ctx context.Context, src, dst string) (int, error) {
	if _, err := os.Stat(src); errors.Is(err, os.ErrNotExist) {
		return 0, nil
	} else if err != nil {
		return 0, fmt.Errorf(i18n.T(i18n.BackupReadResources), err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return 0, fmt.Errorf(i18n.T(i18n.BackupCreateResources), err)
	}
	copied := 0
	err := filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		relative, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		target := filepath.Join(dst, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !entry.Type().IsRegular() || skipName(entry.Name()) {
			return nil
		}
		if _, err := os.Stat(target); err == nil {
			return nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := copyMissingFile(path, target); err != nil {
			return err
		}
		copied++
		return nil
	})
	if err != nil {
		return copied, fmt.Errorf(i18n.T(i18n.BackupMergeResources), err)
	}
	return copied, nil
}

func copyMissingFile(src, dst string) error {
	input, err := os.Open(src)
	if err != nil {
		return err
	}
	defer input.Close()
	temp, err := renameio.TempFile(filepath.Dir(dst), dst)
	if err != nil {
		return err
	}
	defer temp.Cleanup()
	if _, err := io.Copy(temp, input); err != nil {
		return err
	}
	if err := temp.Chmod(0o644); err != nil {
		return err
	}
	return temp.CloseAtomicallyReplace()
}

func extractEntry(entry *zip.File, path string) error {
	src, err := entry.Open()
	if err != nil {
		return fmt.Errorf(i18n.T(i18n.BackupReadEntry), entry.Name, err)
	}
	defer src.Close()
	dst, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf(i18n.T(i18n.BackupWriteEntry), entry.Name, err)
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf(i18n.T(i18n.BackupWriteEntry), entry.Name, err)
	}
	return nil
}

func safeArchiveName(name string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(name))
	if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
		return "", errors.New(i18n.T(i18n.BackupContentEscape))
	}
	return filepath.ToSlash(clean), nil
}

// ApplyPendingImport swaps a staged backup into the data root. It runs
// before the database opens, and restores the previous files on failure.
func ApplyPendingImport(ctx context.Context, paths *storage.Paths) (ImportResult, bool, error) {
	var result ImportResult
	pending := filepath.Join(paths.State, storage.PendingBackupDir)
	if _, err := os.Stat(pending); errors.Is(err, os.ErrNotExist) {
		return result, false, nil
	}
	manifestData, err := os.ReadFile(filepath.Join(pending, "manifest.json"))
	if err != nil {
		os.RemoveAll(pending)
		return result, true, fmt.Errorf(i18n.T(i18n.BackupReadPendingManifest), err)
	}
	manifest, err := parseManifest(manifestData)
	if err != nil {
		os.RemoveAll(pending)
		return result, true, err
	}
	if _, err := os.Stat(filepath.Join(pending, "db", "clipboard.db")); err != nil {
		os.RemoveAll(pending)
		return result, true, errors.New(i18n.T(i18n.BackupPendingMissingDB))
	}
	if err := validateSnapshot(ctx, pending); err != nil {
		os.RemoveAll(pending)
		return result, true, err
	}
	settingsSrc := filepath.Join(pending, "config", "settings.json")
	if _, err := os.Stat(settingsSrc); err != nil {
		os.RemoveAll(pending)
		return result, true, errors.New(i18n.T(i18n.BackupPendingMissingConfig))
	}

	rollback := filepath.Join(paths.State, storage.BackupRollbackDir)
	if err := os.RemoveAll(rollback); err != nil {
		return result, true, fmt.Errorf(i18n.T(i18n.BackupCleanRollback), err)
	}
	if err := os.MkdirAll(rollback, 0o755); err != nil {
		return result, true, fmt.Errorf(i18n.T(i18n.BackupCreateRollback), err)
	}
	swap := func(src, dst string) error {
		if _, err := os.Stat(src); errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.Rename(src, dst)
	}
	settingsDst := filepath.Join(paths.Config, "settings.json")
	fail := func(cause error) (ImportResult, bool, error) {
		rollbackImport(paths, rollback)
		os.RemoveAll(pending)
		os.RemoveAll(rollback)
		return result, true, cause
	}
	if err := swap(paths.DB, filepath.Join(rollback, "db")); err != nil {
		return fail(fmt.Errorf(i18n.T(i18n.BackupCurrentDatabase), err))
	}
	if err := swap(paths.Resources, filepath.Join(rollback, "resources")); err != nil {
		return fail(fmt.Errorf(i18n.T(i18n.BackupCurrentResources), err))
	}
	if err := swap(settingsDst, filepath.Join(rollback, "settings.json")); err != nil {
		return fail(fmt.Errorf(i18n.T(i18n.BackupCurrentSettings), err))
	}
	if err := swap(filepath.Join(pending, "db"), paths.DB); err != nil {
		return fail(fmt.Errorf(i18n.T(i18n.BackupWriteDatabase), err))
	}
	if err := swap(filepath.Join(pending, "resources"), paths.Resources); err != nil {
		return fail(fmt.Errorf(i18n.T(i18n.BackupWriteResources), err))
	}
	if err := swap(settingsSrc, settingsDst); err != nil {
		return fail(fmt.Errorf(i18n.T(i18n.BackupWriteSettings), err))
	}
	if err := os.RemoveAll(pending); err != nil {
		return fail(fmt.Errorf(i18n.T(i18n.BackupCleanImport), err))
	}
	os.RemoveAll(rollback)
	result.Items = manifest.ItemCount
	return result, true, nil
}

func rollbackImport(paths *storage.Paths, rollback string) {
	settingsDst := filepath.Join(paths.Config, "settings.json")
	restore := func(src, dst string) {
		if _, err := os.Stat(src); err != nil {
			return
		}
		os.RemoveAll(dst)
		os.MkdirAll(filepath.Dir(dst), 0o755)
		os.Rename(src, dst)
	}
	restore(filepath.Join(rollback, "db"), paths.DB)
	restore(filepath.Join(rollback, "resources"), paths.Resources)
	restore(filepath.Join(rollback, "settings.json"), settingsDst)
}

// Package backup reads and writes EcoPaste's .ecopastebak archives.
package backup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
)

const (
	Extension = "ecopastebak"

	formatVersion = 1
)

// Mode is the container format: a plain ZIP or the password container.
type Mode string

const (
	ModePlain     Mode = "plain"
	ModeEncrypted Mode = "encrypted"
)

// Source is what an export archives.
type Source struct {
	Database   *database.DB
	Resources  string
	Settings   string
	AppName    string
	AppVersion string
}

// ExportOptions selects the container mode and password.
type ExportOptions struct {
	Mode     Mode
	Password string
}

// ExportResult reports what the backup contains.
type ExportResult struct {
	Path          string
	TotalBytes    int64
	Items         int64
	Texts         int64
	Images        int64
	Files         int64
	ResourceBytes int64
	ExportedAt    time.Time
	Mode          Mode
}

// Export writes a consistent snapshot, resources and settings to target.
func Export(ctx context.Context, src Source, target string, opts ExportOptions) (ExportResult, error) {
	var result ExportResult
	if src.Database == nil {
		return result, errors.New(i18n.T(i18n.BackupMissingDatabase))
	}
	if opts.Mode != ModePlain && opts.Mode != ModeEncrypted {
		return result, fmt.Errorf(i18n.T(i18n.BackupUnknownMode), opts.Mode)
	}
	if opts.Mode == ModeEncrypted && opts.Password == "" {
		return result, errors.New(i18n.T(i18n.BackupPasswordMissing))
	}
	path, err := NormalizePath(target)
	if err != nil {
		return result, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return result, fmt.Errorf(i18n.T(i18n.BackupCreateDirectory), err)
	}
	counts, err := src.Database.Counts(ctx)
	if err != nil {
		return result, err
	}
	resourceBytes, err := dirSize(src.Resources)
	if err != nil {
		return result, fmt.Errorf(i18n.T(i18n.BackupMeasureResources), err)
	}
	exportedAt := time.Now().UTC()
	manifest := backupManifest{
		FormatVersion: formatVersion,
		AppName:       src.AppName,
		AppVersion:    src.AppVersion,
		ExportedAt:    exportedAt,
		Platform:      "macos",
		Encryption:    "none",
		ItemCount:     counts.Items,
		TextCount:     counts.Text,
		ImageCount:    counts.Images,
		FilesCount:    counts.Files,
		ResourceBytes: resourceBytes,
	}
	if manifest.AppName == "" {
		manifest.AppName = "EcoPaste"
	}
	if manifest.AppVersion == "" {
		manifest.AppVersion = "0.0.0"
	}
	if opts.Mode == ModeEncrypted {
		manifest.Encryption = "password"
	}

	work, err := os.MkdirTemp("", "ecopaste-backup-")
	if err != nil {
		return result, fmt.Errorf(i18n.T(i18n.BackupCreateTemp), err)
	}
	defer os.RemoveAll(work)

	snapshot := filepath.Join(work, "clipboard.db")
	if err := src.Database.Snapshot(ctx, snapshot); err != nil {
		return result, err
	}
	payload := filepath.Join(work, "payload.zip")
	if err := writePayload(payload, payloadSource{
		Database: snapshot, Resources: src.Resources,
		Settings: src.Settings, Manifest: manifest,
	}); err != nil {
		return result, err
	}
	if err := writeContainer(path, payload, opts); err != nil {
		return result, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return result, fmt.Errorf(i18n.T(i18n.BackupReadFile), err)
	}
	result = ExportResult{
		Path: path, TotalBytes: info.Size(),
		Items: counts.Items, Texts: counts.Text, Images: counts.Images, Files: counts.Files,
		ResourceBytes: resourceBytes, ExportedAt: exportedAt, Mode: opts.Mode,
	}
	return result, nil
}

// NormalizePath appends the backup extension, as the old app did, and
// rejects a different one.
func NormalizePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New(i18n.T(i18n.BackupChoosePath))
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case "":
		return path + "." + Extension, nil
	case "." + Extension:
		return path, nil
	default:
		return "", fmt.Errorf(i18n.T(i18n.BackupExtension), Extension)
	}
}

func writeContainer(target, payload string, opts ExportOptions) error {
	parent := filepath.Dir(target)
	temp, err := os.CreateTemp(parent, ".ecopastebak-*")
	if err != nil {
		return fmt.Errorf(i18n.T(i18n.BackupCreateFile), err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)

	switch opts.Mode {
	case ModePlain:
		err = copyFile(temp, payload)
	case ModeEncrypted:
		err = writeEncrypted(temp, payload, opts.Password)
	default:
		err = fmt.Errorf(i18n.T(i18n.BackupUnknownMode), opts.Mode)
	}
	if err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return fmt.Errorf(i18n.T(i18n.BackupSyncFile), err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf(i18n.T(i18n.BackupCloseFile), err)
	}
	if err := os.Rename(tempName, target); err != nil {
		return fmt.Errorf(i18n.T(i18n.BackupSaveFile), err)
	}
	return nil
}

func copyFile(dst *os.File, path string) error {
	src, err := os.Open(path)
	if err != nil {
		return fmt.Errorf(i18n.T(i18n.BackupOpenPayload), err)
	}
	defer src.Close()
	if _, err := dst.ReadFrom(src); err != nil {
		return fmt.Errorf(i18n.T(i18n.BackupWritePayload), err)
	}
	return nil
}

func dirSize(path string) (int64, error) {
	if path == "" {
		return 0, nil
	}
	var total int64
	err := filepath.WalkDir(path, func(entry string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	if os.IsNotExist(err) {
		return 0, nil
	}
	return total, err
}

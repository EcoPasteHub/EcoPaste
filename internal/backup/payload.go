package backup

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
)

type backupManifest struct {
	FormatVersion uint16    `json:"formatVersion"`
	AppName       string    `json:"appName"`
	AppVersion    string    `json:"appVersion"`
	ExportedAt    time.Time `json:"exportedAt"`
	Platform      string    `json:"platform"`
	Encryption    string    `json:"encryption"`
	ItemCount     int64     `json:"itemCount"`
	TextCount     int64     `json:"textCount"`
	ImageCount    int64     `json:"imageCount"`
	FilesCount    int64     `json:"filesCount"`
	ResourceBytes int64     `json:"resourceBytes"`
}

type payloadSource struct {
	Database  string
	Resources string
	Settings  string
	Manifest  backupManifest
}

func writePayload(path string, src payloadSource) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf(i18n.T(i18n.BackupCreatePayload), err)
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	if err := writeManifest(writer, src.Manifest); err != nil {
		writer.Close()
		return err
	}
	if err := addFile(writer, src.Database, "db/clipboard.db", true); err != nil {
		writer.Close()
		return err
	}
	if err := addDir(writer, src.Resources, "resources"); err != nil {
		writer.Close()
		return err
	}
	if err := addFile(writer, src.Settings, "config/settings.json", false); err != nil {
		writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf(i18n.T(i18n.BackupFinishPayload), err)
	}
	return nil
}

func writeManifest(writer *zip.Writer, manifest backupManifest) error {
	entry, err := writer.CreateHeader(&zip.FileHeader{
		Name: "manifest.json", Method: zip.Deflate, Modified: time.Now(),
	})
	if err != nil {
		return fmt.Errorf(i18n.T(i18n.BackupWriteManifest), err)
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf(i18n.T(i18n.BackupEncodeManifest), err)
	}
	if _, err := entry.Write(data); err != nil {
		return fmt.Errorf(i18n.T(i18n.BackupWriteManifest), err)
	}
	return nil
}

func addFile(writer *zip.Writer, path, name string, required bool) error {
	if path == "" {
		if required {
			return fmt.Errorf(i18n.T(i18n.BackupMissingEntry), name)
		}
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) && !required {
			return nil
		}
		return fmt.Errorf(i18n.T(i18n.BackupReadSource), name, err)
	}
	if info.IsDir() {
		return fmt.Errorf(i18n.T(i18n.BackupSourceDirectory), name)
	}
	if skipName(info.Name()) {
		return nil
	}
	entry, err := writer.CreateHeader(&zip.FileHeader{
		Name: name, Method: zip.Deflate, Modified: info.ModTime(),
	})
	if err != nil {
		return fmt.Errorf(i18n.T(i18n.BackupWriteEntry), name, err)
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf(i18n.T(i18n.BackupOpenSource), name, err)
	}
	defer file.Close()
	if _, err := io.Copy(entry, file); err != nil {
		return fmt.Errorf(i18n.T(i18n.BackupWriteEntry), name, err)
	}
	return nil
}

func addDir(writer *zip.Writer, root, prefix string) error {
	if root == "" {
		return nil
	}
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil
	}
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		if skipName(d.Name()) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return fmt.Errorf(i18n.T(i18n.BackupReadResources), err)
	}
	sort.Strings(files)
	for _, path := range files {
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf(i18n.T(i18n.BackupResourcePath), err)
		}
		name, err := archiveName(prefix, relative)
		if err != nil {
			return err
		}
		if err := addFile(writer, path, name, true); err != nil {
			return err
		}
	}
	return nil
}

func archiveName(prefix, relative string) (string, error) {
	clean := filepath.Clean(relative)
	if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
		return "", errors.New(i18n.T(i18n.BackupPathEscape))
	}
	return prefix + "/" + filepath.ToSlash(clean), nil
}

func skipName(name string) bool {
	return name == ".DS_Store" ||
		name == ".tmp" ||
		strings.HasSuffix(name, ".tmp") ||
		strings.HasSuffix(name, ".temp") ||
		strings.HasPrefix(name, ".tmp")
}

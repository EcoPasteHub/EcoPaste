package backup

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/EcoPasteHub/EcoPaste/internal/storage"
)

const pendingImportDir = "import-pending"

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

// Inspect reports the container mode without decrypting the payload.
func Inspect(path string) (Mode, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("读取备份文件: %w", err)
	}
	defer file.Close()
	head := make([]byte, len(magic))
	n, err := io.ReadFull(file, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("读取备份文件: %w", err)
	}
	if n >= 2 && head[0] == 'P' && head[1] == 'K' {
		return ModePlain, nil
	}
	if n < len(magic) || string(head) != magic {
		return "", errors.New("不是有效的 EcoPaste 备份文件")
	}
	length := make([]byte, 4)
	if _, err := io.ReadFull(file, length); err != nil {
		return "", errors.New("加密备份文件头无效")
	}
	size := int(binary.LittleEndian.Uint32(length))
	if size <= 0 || size > 64*1024 {
		return "", errors.New("加密备份文件头无效")
	}
	headerData := make([]byte, 4+size)
	copy(headerData, length)
	if _, err := io.ReadFull(file, headerData[4:]); err != nil {
		return "", errors.New("加密备份文件头无效")
	}
	header, _, err := readHeaderWithSize(headerData)
	if err != nil {
		return "", err
	}
	if header.Mode != ModePlain && header.Mode != ModeEncrypted {
		return "", errors.New("备份容器模式无效")
	}
	return header.Mode, nil
}

// StageImport validates path and leaves an extracted copy under the
// state directory for the next launch to swap in.
func StageImport(path, password, stateDir string) (ImportPlan, error) {
	var plan ImportPlan
	payload, err := readPayload(path, password)
	if err != nil {
		return plan, err
	}
	reader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		return plan, fmt.Errorf("读取备份内容: %w", err)
	}
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return plan, fmt.Errorf("创建导入目录: %w", err)
	}
	temp, err := os.MkdirTemp(stateDir, ".import-*")
	if err != nil {
		return plan, fmt.Errorf("创建导入临时目录: %w", err)
	}
	defer os.RemoveAll(temp)

	manifest, err := extractPayload(reader, temp)
	if err != nil {
		return plan, err
	}
	if manifest.FormatVersion != formatVersion {
		return plan, fmt.Errorf("暂不支持备份格式版本 %d", manifest.FormatVersion)
	}
	if _, err := os.Stat(filepath.Join(temp, "db", "clipboard.db")); err != nil {
		return plan, errors.New("备份文件缺少历史数据库")
	}
	if _, err := os.Stat(filepath.Join(temp, "config", "settings.json")); err != nil {
		return plan, errors.New("备份文件缺少设置文件")
	}
	pending := filepath.Join(stateDir, pendingImportDir)
	if err := os.RemoveAll(pending); err != nil {
		return plan, fmt.Errorf("清理旧导入: %w", err)
	}
	if err := os.Rename(temp, pending); err != nil {
		return plan, fmt.Errorf("准备导入: %w", err)
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

func extractPayload(reader *zip.Reader, root string) (backupManifest, error) {
	var manifest backupManifest
	foundManifest := false
	for _, entry := range reader.File {
		name, err := safeArchiveName(entry.Name)
		if err != nil {
			return manifest, err
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
			return manifest, fmt.Errorf("创建导入目录: %w", err)
		}
		if err := extractEntry(entry, path); err != nil {
			return manifest, err
		}
		if name == "manifest.json" {
			foundManifest = true
			data, err := os.ReadFile(path)
			if err != nil {
				return manifest, fmt.Errorf("读取备份清单: %w", err)
			}
			if err := json.Unmarshal(data, &manifest); err != nil {
				return manifest, fmt.Errorf("解析备份清单: %w", err)
			}
		}
	}
	if !foundManifest {
		return manifest, errors.New("备份文件缺少 manifest.json")
	}
	return manifest, nil
}

func extractEntry(entry *zip.File, path string) error {
	src, err := entry.Open()
	if err != nil {
		return fmt.Errorf("读取备份条目 %s: %w", entry.Name, err)
	}
	defer src.Close()
	dst, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("写入备份条目 %s: %w", entry.Name, err)
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("写入备份条目 %s: %w", entry.Name, err)
	}
	return nil
}

func safeArchiveName(name string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(name))
	if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
		return "", errors.New("备份内容路径越界")
	}
	return filepath.ToSlash(clean), nil
}

// ApplyPendingImport swaps a staged backup into the data root. It runs
// before the database opens, and restores the previous files on failure.
func ApplyPendingImport(paths *storage.Paths) (ImportResult, bool, error) {
	var result ImportResult
	pending := filepath.Join(paths.State, pendingImportDir)
	if _, err := os.Stat(pending); errors.Is(err, os.ErrNotExist) {
		return result, false, nil
	}
	manifestData, err := os.ReadFile(filepath.Join(pending, "manifest.json"))
	if err != nil {
		os.RemoveAll(pending)
		return result, true, fmt.Errorf("读取待导入清单: %w", err)
	}
	var manifest backupManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		os.RemoveAll(pending)
		return result, true, fmt.Errorf("解析待导入清单: %w", err)
	}
	if _, err := os.Stat(filepath.Join(pending, "db", "clipboard.db")); err != nil {
		os.RemoveAll(pending)
		return result, true, errors.New("待导入备份缺少历史数据库")
	}
	settingsSrc := filepath.Join(pending, "config", "settings.json")
	if _, err := os.Stat(settingsSrc); err != nil {
		os.RemoveAll(pending)
		return result, true, errors.New("待导入备份缺少设置文件")
	}

	rollback := filepath.Join(paths.State, "import-rollback")
	if err := os.RemoveAll(rollback); err != nil {
		return result, true, fmt.Errorf("清理旧回滚目录: %w", err)
	}
	if err := os.MkdirAll(rollback, 0o755); err != nil {
		return result, true, fmt.Errorf("创建回滚目录: %w", err)
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
		return fail(fmt.Errorf("备份当前数据库: %w", err))
	}
	if err := swap(paths.Resources, filepath.Join(rollback, "resources")); err != nil {
		return fail(fmt.Errorf("备份当前资源: %w", err))
	}
	if err := swap(settingsDst, filepath.Join(rollback, "settings.json")); err != nil {
		return fail(fmt.Errorf("备份当前设置: %w", err))
	}
	if err := swap(filepath.Join(pending, "db"), paths.DB); err != nil {
		return fail(fmt.Errorf("写入导入数据库: %w", err))
	}
	if err := swap(filepath.Join(pending, "resources"), paths.Resources); err != nil {
		return fail(fmt.Errorf("写入导入资源: %w", err))
	}
	if err := swap(settingsSrc, settingsDst); err != nil {
		return fail(fmt.Errorf("写入导入设置: %w", err))
	}
	if err := os.RemoveAll(pending); err != nil {
		return fail(fmt.Errorf("清理导入目录: %w", err))
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

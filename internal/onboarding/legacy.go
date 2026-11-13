package onboarding

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/clipboard"
	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
	"github.com/EcoPasteHub/EcoPaste/internal/storage"
)

// LegacyImportSelection is the part of the v0.6 history to import.
type LegacyImportSelection string

const (
	LegacyNormal   LegacyImportSelection = "normal"
	LegacyFavorite LegacyImportSelection = "favorite"
)

// LegacyDetection reports what a read-only scan found in the v0.6 data
// directory.
type LegacyDetection struct {
	Found               bool
	Path                string
	DatabaseFiles       []string
	ImportableDatabase  string
	ImportableItemCount int
	NormalItemCount     int
	FavoriteItemCount   int
	CheckedAt           time.Time
	Messages            []string
}

// LegacyImportResult counts the rows the importer wrote or skipped.
type LegacyImportResult struct {
	Imported         int
	Skipped          int
	ImportedNormal   int
	ImportedFavorite int
	SelectedTypes    []LegacyImportSelection
	ImportedAt       time.Time
}

// LegacyImporter reads a v0.6 EcoPaste installation and writes its history
// through the current clipboard pipeline. Base overrides the application
// support directory and is empty outside tests.
type LegacyImporter struct {
	DB       *database.DB
	Images   *storage.ImageStore
	Settings func() settings.Clipboard
	Base     string
	Mode     string
}

type legacyMatch struct {
	path                string
	databaseFiles       []string
	importableDatabase  string
	importableItemCount int
	normalItemCount     int
	favoriteItemCount   int
}

type legacyStats struct {
	items    int
	normal   int
	favorite int
}

// Detect scans the default and custom v0.6 data directories without
// changing them.
func (l *LegacyImporter) Detect(ctx context.Context) (LegacyDetection, error) {
	detection := LegacyDetection{CheckedAt: time.Now().UTC()}
	for _, dir := range l.candidates(&detection) {
		match, ok, err := l.inspectDir(ctx, dir, &detection)
		if err != nil {
			return detection, err
		}
		if !ok {
			continue
		}
		if detection.Found && match.importableItemCount <= detection.ImportableItemCount {
			continue
		}
		detection.Found = true
		detection.Path = match.path
		detection.DatabaseFiles = match.databaseFiles
		detection.ImportableDatabase = match.importableDatabase
		detection.ImportableItemCount = match.importableItemCount
		detection.NormalItemCount = match.normalItemCount
		detection.FavoriteItemCount = match.favoriteItemCount
	}
	return detection, nil
}

// Import writes the selected v0.6 rows into the current database. Rows
// that no longer map to clipboard content, or whose content already
// exists, are skipped.
func (l *LegacyImporter) Import(ctx context.Context, types []LegacyImportSelection) (LegacyImportResult, error) {
	selected := normalizeSelection(types)
	if len(selected) == 0 {
		return LegacyImportResult{}, fmt.Errorf("请至少选择一种导入类型")
	}
	if l.DB == nil {
		return LegacyImportResult{}, fmt.Errorf("剪贴板数据库未打开")
	}
	detection, err := l.Detect(ctx)
	if err != nil {
		return LegacyImportResult{}, err
	}
	if detection.ImportableDatabase == "" {
		return LegacyImportResult{}, fmt.Errorf("未找到可导入的旧版数据库")
	}

	db, err := openLegacyDatabase(detection.ImportableDatabase)
	if err != nil {
		return LegacyImportResult{}, err
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx,
		"SELECT type, value, favorite, createTime, note FROM history ORDER BY createTime ASC, id ASC")
	if err != nil {
		return LegacyImportResult{}, fmt.Errorf("读取旧版历史: %w", err)
	}
	defer rows.Close()

	service := &clipboard.Service{DB: l.DB, Images: l.Images, Settings: l.Settings}
	imagesDir := filepath.Join(detection.Path, "images")
	result := LegacyImportResult{SelectedTypes: selected}
	for rows.Next() {
		var (
			itemType sql.NullString
			value    sql.NullString
			favorite sql.NullInt64
			created  sql.NullString
			note     sql.NullString
		)
		if err := rows.Scan(&itemType, &value, &favorite, &created, &note); err != nil {
			return result, fmt.Errorf("读取旧版历史行: %w", err)
		}
		isFavorite := favorite.Valid && favorite.Int64 != 0
		selection := LegacyNormal
		if isFavorite {
			selection = LegacyFavorite
		}
		if !slices.Contains(selected, selection) {
			continue
		}
		payload, ok := l.payload(itemType.String, value.String, imagesDir)
		if !ok {
			result.Skipped++
			continue
		}
		item, err := service.BuildItem(payload)
		if err != nil {
			return result, err
		}
		if item == nil {
			result.Skipped++
			continue
		}
		item.IsFavorite = isFavorite
		item.CreatedAt = parseLegacyTime(created.String)
		item.UpdatedAt = item.CreatedAt
		if text := strings.TrimSpace(note.String); text != "" {
			item.Note = &text
		}
		if _, deduplicated, err := l.DB.Upsert(ctx, item); err != nil {
			return result, err
		} else if deduplicated {
			result.Skipped++
			continue
		}
		result.Imported++
		if isFavorite {
			result.ImportedFavorite++
		} else {
			result.ImportedNormal++
		}
	}
	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("遍历旧版历史: %w", err)
	}
	result.ImportedAt = time.Now().UTC()
	return result, nil
}

func (l *LegacyImporter) payload(itemType, value, imagesDir string) (clipboard.Payload, bool) {
	switch itemType {
	case "text":
		if strings.TrimSpace(value) == "" {
			return clipboard.Payload{}, false
		}
		return clipboard.Payload{Text: value}, true
	case "files":
		var paths []string
		if err := json.Unmarshal([]byte(value), &paths); err != nil || len(paths) == 0 {
			return clipboard.Payload{}, false
		}
		return clipboard.Payload{Files: paths}, true
	case "image":
		name := strings.TrimSpace(value)
		if name == "" || filepath.Base(name) != name {
			return clipboard.Payload{}, false
		}
		data, err := os.ReadFile(filepath.Join(imagesDir, name))
		if err != nil {
			return clipboard.Payload{}, false
		}
		return clipboard.Payload{PNG: data}, true
	default:
		return clipboard.Payload{}, false
	}
}

func normalizeSelection(types []LegacyImportSelection) []LegacyImportSelection {
	selected := make([]LegacyImportSelection, 0, len(types))
	for _, item := range types {
		if item != LegacyNormal && item != LegacyFavorite {
			continue
		}
		if !slices.Contains(selected, item) {
			selected = append(selected, item)
		}
	}
	return selected
}

func parseLegacyTime(value string) time.Time {
	text := strings.TrimSpace(value)
	if text == "" {
		return time.Now().UTC()
	}
	parsed, err := time.ParseInLocation("2006-01-02 15:04:05", text, time.Local)
	if err != nil {
		return time.Now().UTC()
	}
	return parsed.UTC()
}

func (l *LegacyImporter) candidates(detection *LegacyDetection) []string {
	var candidates []string
	for _, dir := range l.defaultCandidates() {
		candidates = appendUnique(candidates, dir)
		for _, custom := range l.customCandidates(dir, detection) {
			candidates = appendUnique(candidates, custom)
		}
	}
	return candidates
}

func (l *LegacyImporter) defaultCandidates() []string {
	base := l.Base
	if base == "" {
		dir, err := os.UserConfigDir()
		if err != nil {
			return nil
		}
		base = dir
	}
	return []string{
		filepath.Join(base, "com.ayangweb.EcoPaste"),
		filepath.Join(base, "EcoPaste"),
	}
}

func (l *LegacyImporter) customCandidates(dir string, detection *LegacyDetection) []string {
	var candidates []string
	for _, name := range legacyStoreNames(l.mode()) {
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var store struct {
			GlobalStore struct {
				Env struct {
					SaveDataDir string `json:"saveDataDir"`
				} `json:"env"`
			} `json:"globalStore"`
		}
		if err := json.Unmarshal(data, &store); err != nil {
			detection.Messages = append(detection.Messages,
				fmt.Sprintf("无法解析旧版 store %s: %v", path, err))
			continue
		}
		target := strings.TrimSpace(store.GlobalStore.Env.SaveDataDir)
		if target == "" {
			continue
		}
		candidates = append(candidates, target)
	}
	return candidates
}

func (l *LegacyImporter) inspectDir(ctx context.Context, dir string, detection *LegacyDetection) (legacyMatch, bool, error) {
	match := legacyMatch{path: dir}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return match, false, nil
	}
	files, err := legacyDatabaseFiles(dir, l.mode())
	if err != nil {
		return match, false, err
	}
	match.databaseFiles = files
	for _, path := range files {
		stats, ok, err := inspectLegacyDatabase(ctx, path)
		if err != nil {
			detection.Messages = append(detection.Messages,
				fmt.Sprintf("无法读取旧版数据库 %s: %v", path, err))
			continue
		}
		if !ok || stats.items <= match.importableItemCount {
			continue
		}
		match.importableDatabase = path
		match.importableItemCount = stats.items
		match.normalItemCount = stats.normal
		match.favoriteItemCount = stats.favorite
	}
	return match, true, nil
}

func legacyDatabaseFiles(dir, mode string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}
		depth := 0
		if rel != "." {
			depth = len(strings.Split(rel, string(filepath.Separator)))
		}
		if entry.IsDir() {
			if depth >= 4 {
				return filepath.SkipDir
			}
			return nil
		}
		if isLegacyDatabaseName(entry.Name(), mode) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("扫描旧版数据目录: %w", err)
	}
	return files, nil
}

func isLegacyDatabaseName(name, mode string) bool {
	expected, other := legacyDatabaseNames(mode)
	if slices.Contains(expected, name) {
		return true
	}
	if slices.Contains(other, name) {
		return false
	}
	isSQLite := strings.HasSuffix(name, ".db") ||
		strings.HasSuffix(name, ".sqlite") ||
		strings.HasSuffix(name, ".sqlite3")
	if !isSQLite {
		return false
	}
	return mode == storage.Prod && !strings.Contains(name, ".dev.") && !strings.HasSuffix(name, ".dev.db")
}

func legacyDatabaseNames(mode string) (expected, other []string) {
	dev := []string{".store.dev.db", ".window-state.dev.db", "EcoPaste.dev.db", "EcoPaste.v2.dev.db"}
	prod := []string{".store.db", ".window-state.db", "EcoPaste.db", "EcoPaste.v2.db"}
	if mode == storage.Dev {
		return dev, prod
	}
	return prod, dev
}

func legacyStoreNames(mode string) []string {
	if mode == storage.Dev {
		return []string{".store.dev.json"}
	}
	return []string{".store.json"}
}

func (l *LegacyImporter) mode() string {
	if l.Mode == storage.Dev || l.Mode == storage.Prod {
		return l.Mode
	}
	return storage.Mode()
}

func appendUnique(paths []string, path string) []string {
	if slices.Contains(paths, path) {
		return paths
	}
	return append(paths, path)
}

func openLegacyDatabase(path string) (*sql.DB, error) {
	u := url.URL{Scheme: "file", Path: path}
	query := u.Query()
	query.Set("mode", "ro")
	u.RawQuery = query.Encode()
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, fmt.Errorf("打开旧版数据库 %s: %w", path, err)
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("打开旧版数据库 %s: %w", path, err)
	}
	return db, nil
}

func inspectLegacyDatabase(ctx context.Context, path string) (legacyStats, bool, error) {
	db, err := openLegacyDatabase(path)
	if err != nil {
		return legacyStats{}, false, err
	}
	defer db.Close()

	var table string
	err = db.QueryRowContext(ctx,
		"SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'history' LIMIT 1").Scan(&table)
	if err == sql.ErrNoRows {
		return legacyStats{}, false, nil
	}
	if err != nil {
		return legacyStats{}, false, err
	}
	var stats legacyStats
	err = db.QueryRowContext(ctx, `SELECT COUNT(*),
		COALESCE(SUM(CASE WHEN favorite IS NULL OR favorite = 0 THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN favorite = 1 THEN 1 ELSE 0 END), 0) FROM history`).
		Scan(&stats.items, &stats.normal, &stats.favorite)
	if err != nil {
		return legacyStats{}, false, err
	}
	return stats, true, nil
}

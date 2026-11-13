package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

func TestDescribeFileEntries(t *testing.T) {
	root := t.TempDir()
	textPath := filepath.Join(root, "notes.txt")
	imagePath := filepath.Join(root, "photo.PNG")
	folderPath := filepath.Join(root, "folder")
	missingDirPath := filepath.Join(root, "gone")
	for path, data := range map[string]string{textPath: "hello", imagePath: "png"} {
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(folderPath, 0o755); err != nil {
		t.Fatal(err)
	}
	types := "f,f,d,d"
	entries := describeFileEntries([]string{textPath, imagePath, folderPath, missingDirPath}, &types, 10)
	if len(entries) != 4 {
		t.Fatalf("entries = %d", len(entries))
	}
	if !entries[0].Exists || entries[0].IsDir || entries[0].IsImage || entries[0].Size != 5 {
		t.Errorf("text entry = %+v", entries[0])
	}
	if !entries[1].Exists || !entries[1].IsImage || entries[1].Name != "photo.PNG" {
		t.Errorf("image entry = %+v", entries[1])
	}
	if !entries[2].Exists || !entries[2].IsDir {
		t.Errorf("folder entry = %+v", entries[2])
	}
	if entries[3].Exists || !entries[3].IsDir {
		t.Errorf("missing folder entry = %+v", entries[3])
	}
	if got := fileEntryLabel(entries[0]); got != "TXT" {
		t.Errorf("text label = %q", got)
	}
	if got := fileEntryLabel(entries[1]); got != "图片" {
		t.Errorf("image label = %q", got)
	}
	if got := fileEntryLabel(entries[2]); got != "文件夹" {
		t.Errorf("folder label = %q", got)
	}
	if got := fileEntryLabel(entries[3]); got != "已丢失" {
		t.Errorf("missing label = %q", got)
	}
	if got := fileSummary(entries[:1], 1); got != "TXT · 5 B" {
		t.Errorf("single summary = %q", got)
	}
	if got := fileSummary(entries, 4); got != "4 项 · 1 项已丢失" {
		t.Errorf("multi summary = %q", got)
	}
}

func TestFilePreviewMode(t *testing.T) {
	root := t.TempDir()
	imagePath := filepath.Join(root, "photo.png")
	if err := os.WriteFile(imagePath, []byte("image"), 0o644); err != nil {
		t.Fatal(err)
	}
	model, id := filePreviewModel(t, []string{imagePath}, "f")
	preview, err := model.Preview(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if preview.ImagePath != imagePath {
		t.Errorf("image path = %q", preview.ImagePath)
	}
	if preview.Title != "photo.png" || !strings.Contains(preview.Meta, "图片 · 5 B") {
		t.Errorf("preview = %+v", preview)
	}
}

func TestFilePreviewListMarksMissingEntries(t *testing.T) {
	root := t.TempDir()
	textPath := filepath.Join(root, "notes.txt")
	missingPath := filepath.Join(root, "gone.txt")
	if err := os.WriteFile(textPath, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	model, id := filePreviewModel(t, []string{textPath, missingPath}, "f,f")
	preview, err := model.Preview(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if preview.ImagePath != "" {
		t.Errorf("multi-file preview used image path %q", preview.ImagePath)
	}
	if preview.Title != "2 个文件" || !strings.Contains(preview.Meta, "2 项 · 1 项已丢失") {
		t.Errorf("preview = %+v", preview)
	}
	for _, want := range []string{"notes.txt", textPath, "TXT · 5 B", missingPath, "已丢失"} {
		if !strings.Contains(preview.Text, want) {
			t.Errorf("preview text missing %q:\n%s", want, preview.Text)
		}
	}
}

func TestFilePreviewSingleNonImage(t *testing.T) {
	root := t.TempDir()
	textPath := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(textPath, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	model, id := filePreviewModel(t, []string{textPath}, "f")
	preview, err := model.Preview(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if preview.ImagePath != "" || preview.Title != "notes.txt" {
		t.Errorf("preview = %+v", preview)
	}
	if !strings.Contains(preview.Text, textPath) || !strings.Contains(preview.Text, "TXT · 5 B") {
		t.Errorf("preview text = %q", preview.Text)
	}
}

func TestHistoryFileRowSummaryAndMissingTitle(t *testing.T) {
	root := t.TempDir()
	db, err := database.Open(filepath.Join(root, "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	missingPath := filepath.Join(root, "gone.pdf")
	model := &historyModel{runtime: &clipboardRuntime{db: db}}
	row := model.historyItem(context.Background(), database.Item{
		Kind:    database.KindFiles,
		Content: missingPath,
	})
	if row.Title != "gone.pdf（已丢失）" || row.FileSummary != "已丢失" {
		t.Errorf("row = %+v", row)
	}
}

func filePreviewModel(t *testing.T, files []string, types string) (*historyModel, string) {
	t.Helper()
	root := t.TempDir()
	db, err := database.Open(filepath.Join(root, "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	store, err := settings.Open(filepath.Join(root, settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	id, _, err := db.Upsert(context.Background(), &database.Item{
		Kind:      database.KindFiles,
		Content:   strings.Join(files, "\n"),
		FileTypes: &types,
		Platform:  "macos",
	})
	if err != nil {
		t.Fatal(err)
	}
	model := &historyModel{runtime: &clipboardRuntime{db: db}, store: store}
	return model, id
}

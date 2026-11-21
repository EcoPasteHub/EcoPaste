package backup

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkExtractPlainBackup(b *testing.B) {
	root := b.TempDir()
	path := filepath.Join(root, "large.ecopastebak")
	file, err := os.Create(path)
	if err != nil {
		b.Fatal(err)
	}
	writer := zip.NewWriter(file)
	if err := writeManifest(writer, backupManifest{FormatVersion: formatVersion}); err != nil {
		b.Fatal(err)
	}
	for _, name := range []string{"db/clipboard.db", "config/settings.json", "resources/large"} {
		entry, err := writer.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store})
		if err != nil {
			b.Fatal(err)
		}
		if name == "resources/large" {
			block := make([]byte, 1024*1024)
			for range 32 {
				if _, err := entry.Write(block); err != nil {
					b.Fatal(err)
				}
			}
		}
	}
	if err := writer.Close(); err != nil {
		b.Fatal(err)
	}
	if err := file.Close(); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		staged, _, err := extractImport(path, "", root, ".bench-*")
		if err != nil {
			b.Fatal(err)
		}
		if err := os.RemoveAll(staged); err != nil {
			b.Fatal(err)
		}
	}
}

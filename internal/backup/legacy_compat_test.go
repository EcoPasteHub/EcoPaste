package backup

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
)

func TestImportBackupsWrittenByLegacyApp(t *testing.T) {
	cases := []struct {
		name     string
		file     string
		password string
		mode     Mode
	}{
		{name: "plain", file: "legacy-plain.ecopastebak", mode: ModePlain},
		{name: "encrypted", file: "legacy-encrypted.ecopastebak", password: "legacy-password", mode: ModeEncrypted},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			archive := filepath.Join("testdata", tc.file)
			if mode, err := Inspect(archive); err != nil || mode != tc.mode {
				t.Fatalf("Inspect = %q, %v, want %q", mode, err, tc.mode)
			}
			paths := testImportPaths(t)
			db, err := database.Open(filepath.Join(paths.DB, "clipboard.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			result, err := Merge(context.Background(), archive, tc.password, db, paths.Resources)
			if err != nil {
				t.Fatal(err)
			}
			if result.Items != 3 || result.Skipped != 0 || result.Resources != 1 {
				t.Errorf("merge result = %+v", result)
			}
			if general, ok := result.Settings["general"].(map[string]any); !ok || general["dockIcon"] != true {
				t.Errorf("settings patch = %#v", result.Settings)
			}

			item, err := db.FindByID(context.Background(), "item-text")
			if err != nil {
				t.Fatal(err)
			}
			if item == nil {
				t.Fatal("legacy text row was not imported")
			}
			if item.Content != "https://example.com/legacy" || !item.IsFavorite || item.Note == nil ||
				*item.Note != "from legacy" || item.GroupID == nil || *item.GroupID != "group-1" ||
				item.SourceAppID == nil || *item.SourceAppID != "com.apple.Safari" {
				t.Errorf("legacy text row = %+v", item)
			}
			counts, err := db.Counts(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if counts.Items != 4 {
				t.Errorf("counts = %+v", counts)
			}
			resource := filepath.Join(paths.Resources, "clipboard-images", "origin", "aa", "aabbcc.png")
			data, err := os.ReadFile(resource)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != "\x89PNG\r\n\x1a\nlegacy" {
				t.Errorf("resource = %q", data)
			}
		})
	}
}

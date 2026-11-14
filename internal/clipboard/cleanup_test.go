package clipboard

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"testing"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

func TestRetentionCutoff(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		retention settings.Retention
		want      *time.Time
	}{
		{"disabled value", settings.Retention{Unit: settings.RetentionDays}, nil},
		{"forever", settings.Retention{Value: 7, Unit: settings.RetentionForever}, nil},
		{"hours", settings.Retention{Value: 2, Unit: settings.RetentionHours}, timePtr(now.Add(-2 * time.Hour))},
		{"days", settings.Retention{Value: 3, Unit: settings.RetentionDays}, timePtr(now.Add(-72 * time.Hour))},
		{"weeks", settings.Retention{Value: 1, Unit: settings.RetentionWeeks}, timePtr(now.Add(-7 * 24 * time.Hour))},
		{"months", settings.Retention{Value: 1, Unit: settings.RetentionMonths}, timePtr(now.Add(-30 * 24 * time.Hour))},
	}
	for _, tt := range tests {
		got := RetentionCutoff(tt.retention, now)
		if (got == nil) != (tt.want == nil) {
			t.Fatalf("%s: got %v, want %v", tt.name, got, tt.want)
		}
		if got != nil && !got.Equal(*tt.want) {
			t.Errorf("%s: got %v, want %v", tt.name, got, *tt.want)
		}
	}
}

func timePtr(t time.Time) *time.Time { return &t }

func TestCleanupRemovesExpiredImage(t *testing.T) {
	cfg := settings.Default().Clipboard
	cfg.History.Retention = settings.Retention{Value: 1, Unit: settings.RetentionDays}
	svc, db, images := testService(t, cfg)
	ctx := context.Background()
	now := time.Now().UTC()

	stored, err := images.StorePNG(cleanupPNG(t))
	if err != nil {
		t.Fatal(err)
	}
	item := &database.Item{
		Kind:      database.KindImage,
		Content:   stored.FileName,
		Platform:  "macos",
		CreatedAt: now.Add(-48 * time.Hour),
	}
	item.UpdatedAt = item.CreatedAt
	if _, _, err := db.Upsert(ctx, item); err != nil {
		t.Fatal(err)
	}

	result, err := svc.Cleanup(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if result.Removed != 1 {
		t.Errorf("removed = %d, want 1", result.Removed)
	}
	if _, err := images.Read(stored.FileName); err == nil {
		t.Error("image file still exists")
	}
}

func TestClearRemovesOnlyDeletedImageFiles(t *testing.T) {
	cfg := settings.Default().Clipboard
	svc, db, images := testService(t, cfg)
	ctx := context.Background()

	normal, err := images.StorePNG(cleanupPNG(t))
	if err != nil {
		t.Fatal(err)
	}
	favorite, err := images.StorePNG(cleanupPNGSize(t, 3, 2))
	if err != nil {
		t.Fatal(err)
	}
	for _, stored := range []struct {
		image    string
		favorite bool
	}{
		{image: normal.FileName},
		{image: favorite.FileName, favorite: true},
	} {
		item := &database.Item{
			Kind: database.KindImage, Content: stored.image, Platform: "macos",
			IsFavorite: stored.favorite,
		}
		if _, _, err := db.Upsert(ctx, item); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := svc.Clear(ctx, false, false); err != nil {
		t.Fatal(err)
	}
	if _, err := images.Read(normal.FileName); err == nil {
		t.Error("deleted image file still exists")
	}
	if _, err := images.Read(favorite.FileName); err != nil {
		t.Errorf("protected image file was removed: %v", err)
	}
	page, err := db.Page(ctx, database.Query{})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || !page.Items[0].IsFavorite {
		t.Errorf("remaining items = %+v", page.Items)
	}

	if _, err := svc.Clear(ctx, true, true); err != nil {
		t.Fatal(err)
	}
	if _, err := images.Read(favorite.FileName); err == nil {
		t.Error("favorite image file still exists after clearing all")
	}
}

func cleanupPNG(t *testing.T) []byte {
	return cleanupPNGSize(t, 2, 2)
}

func cleanupPNGSize(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	img.Set(0, 0, color.RGBA{B: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

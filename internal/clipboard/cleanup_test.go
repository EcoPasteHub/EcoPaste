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

func cleanupPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{B: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

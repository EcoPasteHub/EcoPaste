package clipboard

import (
	"context"
	"fmt"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

// Cleanup applies the current retention and item-count limits and removes
// image files that belonged to deleted rows.
func (s *Service) Cleanup(ctx context.Context, now time.Time) (database.CleanupResult, error) {
	cfg := settings.Clipboard{}
	if s.Settings != nil {
		cfg = s.Settings()
	}
	cutoff := RetentionCutoff(cfg.History.Retention, now)
	maxCount := int(cfg.History.MaxCount)
	if cutoff == nil && maxCount <= 0 {
		return database.CleanupResult{}, nil
	}
	result, err := s.DB.CleanupHistory(ctx, cutoff, maxCount)
	if err != nil {
		return result, err
	}
	if err := s.removeImages(result.ImageFiles); err != nil {
		return result, err
	}
	return result, nil
}

// Clear deletes history according to the caller's keep choices.
func (s *Service) Clear(ctx context.Context, deleteFavorites, deletePinned bool) (database.CleanupResult, error) {
	result, err := s.DB.ClearHistory(ctx, deleteFavorites, deletePinned)
	if err != nil {
		return result, err
	}
	if err := s.removeImages(result.ImageFiles); err != nil {
		return result, err
	}
	return result, nil
}

func (s *Service) removeImages(files []string) error {
	if s.Images == nil {
		return nil
	}
	var first error
	for _, file := range files {
		if err := s.Images.Remove(file); err != nil && first == nil {
			first = fmt.Errorf("remove cleaned image: %w", err)
		}
	}
	return first
}

// RetentionCutoff converts a retention setting into an absolute cutoff.
// Zero values and "forever" disable the age limit; months use 30 days.
func RetentionCutoff(retention settings.Retention, now time.Time) *time.Time {
	if retention.Value == 0 || retention.Unit == settings.RetentionForever {
		return nil
	}
	var duration time.Duration
	switch retention.Unit {
	case settings.RetentionHours:
		duration = time.Duration(retention.Value) * time.Hour
	case settings.RetentionDays:
		duration = time.Duration(retention.Value) * 24 * time.Hour
	case settings.RetentionWeeks:
		duration = time.Duration(retention.Value) * 7 * 24 * time.Hour
	case settings.RetentionMonths:
		duration = time.Duration(retention.Value) * 30 * 24 * time.Hour
	default:
		return nil
	}
	cutoff := now.Add(-duration)
	return &cutoff
}

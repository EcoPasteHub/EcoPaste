package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/google/renameio/v2"
)

const FileName = "settings.json"

// Store keeps the current settings in memory and writes every change
// atomically to config/settings.json.
type Store struct {
	mu      sync.RWMutex
	path    string
	current Settings
	subs    map[chan struct{}]struct{}
}

// Open loads the settings file, creating it with defaults on first run.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create settings dir: %w", err)
	}
	current, exists, err := load(path)
	if err != nil {
		return nil, err
	}
	s := &Store{path: path, current: current}
	if !exists {
		if err := s.write(current); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// Path returns the settings file's location.
func (s *Store) Path() string { return s.path }

// Changes returns a channel notified after each successful change, plus a
// function that stops the subscription. Notifications coalesce: one pending
// signal is enough for a consumer to read the latest snapshot, so a burst of
// updates never queues up behind a slow reader.
func (s *Store) Changes() (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	s.mu.Lock()
	if s.subs == nil {
		s.subs = map[chan struct{}]struct{}{}
	}
	s.subs[ch] = struct{}{}
	s.mu.Unlock()
	return ch, func() {
		s.mu.Lock()
		delete(s.subs, ch)
		s.mu.Unlock()
	}
}

// notifyLocked wakes every subscriber. The caller holds s.mu.
func (s *Store) notifyLocked() {
	for ch := range s.subs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// Snapshot returns a copy of the current settings.
func (s *Store) Snapshot() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneSettings(s.current)
}

// Update deep-merges a JSON object patch into the current settings,
// validates the result and persists it. Objects merge recursively;
// arrays and scalars replace the previous value.
func (s *Store) Update(patch map[string]any) (Settings, error) {
	if patch == nil {
		return Settings{}, errors.New("settings patch must be a JSON object")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	merged, err := merge(s.current, patch)
	if err != nil {
		return Settings{}, err
	}
	if err := validate(merged); err != nil {
		return Settings{}, err
	}
	if err := s.write(merged); err != nil {
		return Settings{}, err
	}
	s.current = merged
	s.notifyLocked()
	return cloneSettings(merged), nil
}

// Replace swaps the whole settings for next, validating and persisting
// it. It is used by backup imports.
func (s *Store) Replace(next Settings) error {
	if err := validate(next); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.write(next); err != nil {
		return err
	}
	s.current = cloneSettings(next)
	s.notifyLocked()
	return nil
}

// Reset restores the published defaults.
func (s *Store) Reset() (Settings, error) {
	next := Default()
	if err := s.Replace(next); err != nil {
		return Settings{}, err
	}
	return cloneSettings(next), nil
}

func (s *Store) write(next Settings) error {
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	// Earlier builds wrote through a fixed ".tmp" sibling; drop any residue so
	// it cannot accumulate next to the settings file.
	_ = os.Remove(s.path + ".tmp")
	if err := renameio.WriteFile(s.path, data, 0o644); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}
	return nil
}

func load(path string) (Settings, bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), false, nil
	}
	if err != nil {
		return Settings{}, false, fmt.Errorf("read settings: %w", err)
	}
	next := Default()
	if err := json.Unmarshal(data, &next); err != nil {
		log.Printf("settings file %s is unreadable, using defaults: %v", path, err)
		return Default(), true, nil
	}
	if err := validateDecoded(next); err != nil {
		log.Printf("settings file %s has invalid values, using defaults: %v", path, err)
		return Default(), true, nil
	}
	return next, true, nil
}

func merge(current Settings, patch map[string]any) (Settings, error) {
	data, err := json.Marshal(current)
	if err != nil {
		return Settings{}, fmt.Errorf("encode current settings: %w", err)
	}
	var base map[string]any
	if err := json.Unmarshal(data, &base); err != nil {
		return Settings{}, fmt.Errorf("decode current settings: %w", err)
	}
	deepMerge(base, patch)

	merged, err := json.Marshal(base)
	if err != nil {
		return Settings{}, fmt.Errorf("encode merged settings: %w", err)
	}
	next := Default()
	if err := json.Unmarshal(merged, &next); err != nil {
		return Settings{}, fmt.Errorf("invalid settings patch: %w", err)
	}
	return next, nil
}

func deepMerge(base, patch map[string]any) {
	for key, value := range patch {
		if existing, ok := base[key]; ok {
			if existingMap, ok := existing.(map[string]any); ok {
				if patchMap, ok := value.(map[string]any); ok {
					deepMerge(existingMap, patchMap)
					continue
				}
			}
		}
		base[key] = value
	}
}

func validate(s Settings) error {
	if err := validateDecoded(s); err != nil {
		return err
	}
	if value := s.Clipboard.Window.SelectGroupOnOpen; value != "preserve" && value != "all" {
		if !strings.HasPrefix(value, "group:") || strings.TrimSpace(strings.TrimPrefix(value, "group:")) == "" {
			return errors.New("open group selection is invalid")
		}
	}
	clipboard := normalizeShortcut(s.Shortcuts.OpenClipboard)
	preference := normalizeShortcut(s.Shortcuts.OpenPreference)
	if clipboard != "" && preference != "" && clipboard == preference {
		return errors.New("global shortcuts must be unique")
	}
	return nil
}

func validateDecoded(s Settings) error {
	for _, validate := range []func(Settings) error{
		validateAppearance,
		validateCapture,
		validateContent,
		validateHistory,
		validateWindow,
		validatePreview,
		validateUpdates,
	} {
		if err := validate(s); err != nil {
			return err
		}
	}
	return nil
}

func validateAppearance(s Settings) error {
	if err := validateOneOf("appearance.theme", s.Appearance.Theme, ThemeAuto, ThemeLight, ThemeDark); err != nil {
		return err
	}
	return validateOneOf("appearance.language", s.Appearance.Language, LanguageAuto, LanguageZhCN, LanguageEnUS)
}

func validateCapture(s Settings) error {
	for _, kind := range s.Clipboard.Capture.Order {
		if err := validateOneOf("clipboard.capture.order", kind,
			CaptureFiles, CaptureImage, CaptureHTML, CaptureRTF, CaptureText); err != nil {
			return err
		}
	}
	return nil
}

func validateContent(s Settings) error {
	if err := validateOneOf("clipboard.content.autoPaste", s.Clipboard.Content.AutoPaste,
		AutoPasteDisabled, AutoPasteSingleClickPaste, AutoPasteDoubleClickPaste,
		AutoPasteSingleClickCopy, AutoPasteDoubleClickCopy); err != nil {
		return err
	}
	if err := validateOneOf("clipboard.content.middleClick", s.Clipboard.Content.MiddleClick,
		MiddleClickDisabled, MiddleClickSingleClickPaste, MiddleClickSingleClickPlain,
		MiddleClickSingleClickCopy, MiddleClickSingleClickCopyPl); err != nil {
		return err
	}
	if err := validateOneOf("clipboard.content.sort", s.Clipboard.Content.Sort,
		SortCreatedAt, SortUpdatedAt, SortUseCount); err != nil {
		return err
	}
	if err := validateItemActions("clipboard.content.itemActions", s.Clipboard.Content.ItemActions); err != nil {
		return err
	}
	if err := validateItemActions("clipboard.content.itemActionOrder", s.Clipboard.Content.ItemActionOrder); err != nil {
		return err
	}
	return nil
}

func validateHistory(s Settings) error {
	return validateOneOf("clipboard.history.retention.unit", s.Clipboard.History.Retention.Unit,
		RetentionHours, RetentionDays, RetentionWeeks, RetentionMonths, RetentionForever)
}

func validateWindow(s Settings) error {
	if err := validateOneOf("clipboard.window.position", s.Clipboard.Window.Position,
		PositionRemember, PositionFollowCursor, PositionCenter); err != nil {
		return err
	}
	if err := validateOneOf("clipboard.window.selectRangeOnOpen", s.Clipboard.Window.SelectRangeOnOpen,
		RangePreserve, RangeAll, RangeFavorite); err != nil {
		return err
	}
	if err := validateOneOf("clipboard.window.selectCategoryOnOpen", s.Clipboard.Window.SelectCategoryOnOpen,
		CategoryPreserve, CategoryAll, CategoryText, CategoryImage, CategoryFiles); err != nil {
		return err
	}
	return nil
}

func validatePreview(s Settings) error {
	return validateOneOf("clipboard.preview.hoverDelayMs", s.Clipboard.Preview.HoverDelayMs,
		300, 500, 1000)
}

func validateUpdates(s Settings) error {
	return validateOneOf("update.frequency", s.Update.Frequency,
		UpdateDaily, UpdateWeekly, UpdateMonthly)
}

func validateItemActions(field string, actions []ItemAction) error {
	for _, action := range actions {
		if err := validateOneOf(field, action,
			ActionPaste, ActionPastePlain, ActionPastePath, ActionCopy, ActionCopyPlain,
			ActionOpenLink, ActionSendEmail, ActionReveal, ActionNote, ActionStar,
			ActionPin, ActionDelete); err != nil {
			return err
		}
	}
	return nil
}

func validateOneOf[T comparable](field string, value T, allowed ...T) error {
	if slices.Contains(allowed, value) {
		return nil
	}
	return fmt.Errorf("%s is invalid: %v", field, value)
}

func normalizeShortcut(value string) string {
	parts := strings.Split(value, "+")
	keys := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			keys = append(keys, strings.ToLower(part))
		}
	}
	return strings.Join(keys, "+")
}

func cloneSettings(src Settings) Settings {
	dst := src
	dst.Clipboard.Capture.Order = slices.Clone(src.Clipboard.Capture.Order)
	dst.Clipboard.Content.ItemActions = slices.Clone(src.Clipboard.Content.ItemActions)
	dst.Clipboard.Content.ItemActionOrder = slices.Clone(src.Clipboard.Content.ItemActionOrder)
	dst.Clipboard.Filters.ExcludedAppIDs = slices.Clone(src.Clipboard.Filters.ExcludedAppIDs)
	dst.Update.LastCheckedAt = cloneString(src.Update.LastCheckedAt)
	dst.Update.SkippedVersion = cloneString(src.Update.SkippedVersion)
	return dst
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

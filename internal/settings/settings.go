package settings

import (
	"runtime"
	"sort"
)

// Settings mirrors config/settings.json. Omitted fields take their defaults.
type Settings struct {
	General    General    `json:"general"`
	Appearance Appearance `json:"appearance"`
	Shortcuts  Shortcuts  `json:"shortcuts"`
	Clipboard  Clipboard  `json:"clipboard"`
	Logging    Logging    `json:"logging"`
}

type Logging struct {
	Level LogLevel `json:"level"`
}

type LogLevel string

const (
	LogLevelAll   LogLevel = "all"
	LogLevelError LogLevel = "error"
	LogLevelWarn  LogLevel = "warn"
	LogLevelInfo  LogLevel = "info"
	LogLevelDebug LogLevel = "debug"
)

type General struct {
	AutoStart bool `json:"autoStart"`
	TrayIcon  bool `json:"trayIcon"`
}

type Appearance struct {
	Theme    Theme    `json:"theme"`
	Language Language `json:"language"`
}

type Theme string

const (
	ThemeAuto  Theme = "auto"
	ThemeLight Theme = "light"
	ThemeDark  Theme = "dark"
)

type Language string

const (
	LanguageAuto Language = "auto"
	LanguageZhCN Language = "zh-CN"
	LanguageEnUS Language = "en-US"
)

type Shortcuts struct {
	OpenClipboard  string `json:"openClipboard"`
	OpenPreference string `json:"openPreference"`
}

type Clipboard struct {
	Capture    Capture    `json:"capture"`
	Content    Content    `json:"content"`
	Display    Display    `json:"display"`
	Sensitive  Sensitive  `json:"sensitive"`
	History    History    `json:"history"`
	Search     Search     `json:"search"`
	Window     Window     `json:"window"`
	Preview    Preview    `json:"preview"`
	Feedback   Feedback   `json:"feedback"`
	SourceApps SourceApps `json:"sourceApps"`
}

// SourceAppMode is how clipboard content from one application is handled.
type SourceAppMode string

const (
	// SourceAppAllowed captures content and allows the global shortcut. It is
	// the default and is represented by the absence of a stored rule.
	SourceAppAllowed SourceAppMode = "allowed"
	// SourceAppBlockCapture keeps the application's copies out of history.
	SourceAppBlockCapture SourceAppMode = "blockCapture"
	// SourceAppBlockWake ignores the global shortcut while the application is
	// frontmost.
	SourceAppBlockWake SourceAppMode = "blockWake"
	// SourceAppBlockCaptureAndWake applies both restrictions at once.
	SourceAppBlockCaptureAndWake SourceAppMode = "blockCaptureAndWake"
)

// SourceAppRule is one application's non-default handling mode.
type SourceAppRule struct {
	ID   string        `json:"id"`
	Mode SourceAppMode `json:"mode"`
}

// SourceApps stores only the applications the user has configured. Every
// other application follows the platform default, so the default state has no
// per-application records.
type SourceApps struct {
	Rules []SourceAppRule `json:"rules,omitempty"`
}

// ModeFor returns the configured mode for id, or its platform default.
func (s SourceApps) ModeFor(id string) SourceAppMode {
	for _, rule := range s.Rules {
		if rule.ID == id {
			return rule.Mode
		}
	}
	return DefaultSourceAppMode(id)
}

// BlocksCapture reports whether copies from id must stay out of history.
func (s SourceApps) BlocksCapture(id string) bool {
	mode := s.ModeFor(id)
	return mode == SourceAppBlockCapture || mode == SourceAppBlockCaptureAndWake
}

// BlocksWake reports whether the global shortcut is ignored while id is
// frontmost.
func (s SourceApps) BlocksWake(id string) bool {
	mode := s.ModeFor(id)
	return mode == SourceAppBlockWake || mode == SourceAppBlockCaptureAndWake
}

const (
	keychainAccessBundleID = "com.apple.keychainaccess"
	passwordsBundleID      = "com.apple.Passwords"
)

// DefaultSourceAppIDs returns the applications that must always appear in the
// source-app list because the platform gives them a non-default mode. Only
// macOS has such applications.
func DefaultSourceAppIDs() []string {
	if runtime.GOOS != "darwin" {
		return nil
	}
	return []string{keychainAccessBundleID, passwordsBundleID}
}

// DefaultSourceAppMode returns the platform default for an application. Only
// macOS excludes its password managers out of the box, and only from capture.
func DefaultSourceAppMode(id string) SourceAppMode {
	switch id {
	case keychainAccessBundleID, passwordsBundleID:
		if runtime.GOOS == "darwin" {
			return SourceAppBlockCapture
		}
	}
	return SourceAppAllowed
}

// ListedIDs returns every application the settings list must always show:
// platform defaults plus applications the user has configured. Running and
// historical apps are added by the app registry.
func (s SourceApps) ListedIDs() []string {
	ids := DefaultSourceAppIDs()
	seen := make(map[string]struct{}, len(ids)+len(s.Rules))
	for _, id := range ids {
		seen[id] = struct{}{}
	}
	for _, rule := range s.Rules {
		if rule.ID == "" {
			continue
		}
		if _, ok := seen[rule.ID]; ok {
			continue
		}
		seen[rule.ID] = struct{}{}
		ids = append(ids, rule.ID)
	}
	return ids
}

// IsSourceAppMode reports whether mode is one of the four supported states.
func IsSourceAppMode(mode SourceAppMode) bool {
	switch mode {
	case SourceAppAllowed, SourceAppBlockCapture, SourceAppBlockWake, SourceAppBlockCaptureAndWake:
		return true
	}
	return false
}

// NormalizeSourceAppRules keeps one valid rule per application in a stable
// order and drops rules that merely restate the platform default, so the
// default state needs no per-application records.
func NormalizeSourceAppRules(rules []SourceAppRule) []SourceAppRule {
	seen := make(map[string]struct{}, len(rules))
	out := make([]SourceAppRule, 0, len(rules))
	for _, rule := range rules {
		if rule.ID == "" || !IsSourceAppMode(rule.Mode) || rule.Mode == DefaultSourceAppMode(rule.ID) {
			continue
		}
		if _, ok := seen[rule.ID]; ok {
			continue
		}
		seen[rule.ID] = struct{}{}
		out = append(out, rule)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

type Capture struct {
	Text       bool          `json:"text"`
	HTML       bool          `json:"html"`
	RTF        bool          `json:"rtf"`
	Image      bool          `json:"image"`
	Files      bool          `json:"files"`
	MaxTextMB  uint32        `json:"maxTextMb"`
	MaxImageMB uint32        `json:"maxImageMb"`
	Order      []CaptureKind `json:"order"`
}

type CaptureKind string

const (
	CaptureFiles CaptureKind = "files"
	CaptureImage CaptureKind = "image"
	CaptureHTML  CaptureKind = "html"
	CaptureRTF   CaptureKind = "rtf"
	CaptureText  CaptureKind = "text"
)

// DefaultCaptureOrder ranks representations from most specific to plain text.
func DefaultCaptureOrder() []CaptureKind {
	return []CaptureKind{CaptureFiles, CaptureImage, CaptureHTML, CaptureRTF, CaptureText}
}

// IsCaptureKind reports whether value names a supported pasteboard
// representation.
func IsCaptureKind(value CaptureKind) bool {
	switch value {
	case CaptureFiles, CaptureImage, CaptureHTML, CaptureRTF, CaptureText:
		return true
	}
	return false
}

// NormalizeCaptureOrder keeps valid kinds once in their configured order,
// then appends any missing defaults so every supported kind remains
// reachable.
func NormalizeCaptureOrder(order []CaptureKind) []CaptureKind {
	seen := map[CaptureKind]struct{}{}
	normalized := make([]CaptureKind, 0, len(DefaultCaptureOrder()))
	for _, kind := range order {
		if !IsCaptureKind(kind) {
			continue
		}
		if _, ok := seen[kind]; ok {
			continue
		}
		seen[kind] = struct{}{}
		normalized = append(normalized, kind)
	}
	for _, kind := range DefaultCaptureOrder() {
		if _, ok := seen[kind]; ok {
			continue
		}
		normalized = append(normalized, kind)
	}
	return normalized
}

type Content struct {
	AutoPaste                         AutoPaste    `json:"autoPaste"`
	CopyPlain                         bool         `json:"copyPlain"`
	CopyThenHideWindow                bool         `json:"copyThenHideWindow"`
	ShowOriginalPreview               bool         `json:"showOriginalPreview"`
	DeleteConfirm                     bool         `json:"deleteConfirm"`
	DeleteFavoriteItems               bool         `json:"deleteFavoriteItems"`
	DeleteFavoriteConfirm             bool         `json:"deleteFavoriteConfirm"`
	DeletePinnedItems                 bool         `json:"deletePinnedItems"`
	DeletePinnedConfirm               bool         `json:"deletePinnedConfirm"`
	DeleteFavoriteItemsOnlyInFavGroup bool         `json:"deleteFavoriteItemsOnlyInFavoriteGroup"`
	AutoFavorite                      bool         `json:"autoFavorite"`
	Sort                              Sort         `json:"sort"`
	ItemActions                       []ItemAction `json:"itemActions"`
	ItemActionOrder                   []ItemAction `json:"itemActionOrder"`
}

type AutoPaste string

const (
	AutoPasteDisabled         AutoPaste = "disabled"
	AutoPasteSingleClickPaste AutoPaste = "singleClickPaste"
	AutoPasteDoubleClickPaste AutoPaste = "doubleClickPaste"
	AutoPasteSingleClickCopy  AutoPaste = "singleClickCopy"
	AutoPasteDoubleClickCopy  AutoPaste = "doubleClickCopy"
)

type Sort string

const (
	SortCreatedAt Sort = "createdAtDesc"
	SortUpdatedAt Sort = "updatedAtDesc"
	SortUseCount  Sort = "useCountDesc"
)

type ItemAction string

const (
	ActionPaste     ItemAction = "paste"
	ActionCopy      ItemAction = "copy"
	ActionCopyPlain ItemAction = "copyPlain"
	ActionOpenLink  ItemAction = "openLink"
	ActionSendEmail ItemAction = "sendEmail"
	ActionReveal    ItemAction = "reveal"
	ActionNote      ItemAction = "note"
	ActionStar      ItemAction = "star"
	ActionPin       ItemAction = "pinItem"
	ActionDelete    ItemAction = "delete"
)

type Display struct {
	TextMaxLines   uint8  `json:"textMaxLines"`
	ImageMaxHeight uint16 `json:"imageMaxHeight"`
	FileMaxCount   uint8  `json:"fileMaxCount"`
}

type Sensitive struct {
	CollectSecrets bool `json:"collectSecrets"`
	RedactSecrets  bool `json:"redactSecrets"`
}

type History struct {
	Retention Retention `json:"retention"`
	MaxCount  uint32    `json:"maxCount"`
}

type Retention struct {
	Value uint32        `json:"value"`
	Unit  RetentionUnit `json:"unit"`
}

type RetentionUnit string

const (
	RetentionHours   RetentionUnit = "hours"
	RetentionDays    RetentionUnit = "days"
	RetentionWeeks   RetentionUnit = "weeks"
	RetentionMonths  RetentionUnit = "months"
	RetentionForever RetentionUnit = "forever"
)

type Search struct {
	ClearOnHide bool `json:"clearOnHide"`
}

type Window struct {
	Position             WindowPosition `json:"position"`
	ScrollToTopOnOpen    bool           `json:"scrollToTopOnOpen"`
	SelectRangeOnOpen    RangeSelection `json:"selectRangeOnOpen"`
	SelectCategoryOnOpen CategorySelect `json:"selectCategoryOnOpen"`
	SelectGroupOnOpen    string         `json:"selectGroupOnOpen"`
}

type WindowPosition string

const (
	PositionRemember     WindowPosition = "remember"
	PositionFollowCursor WindowPosition = "followCursor"
	PositionCenter       WindowPosition = "center"
)

type RangeSelection string

const (
	RangePreserve RangeSelection = "preserve"
	RangeAll      RangeSelection = "all"
	RangeFavorite RangeSelection = "favorite"
)

type CategorySelect string

const (
	CategoryPreserve CategorySelect = "preserve"
	CategoryAll      CategorySelect = "all"
	CategoryText     CategorySelect = "text"
	CategoryImage    CategorySelect = "image"
	CategoryFiles    CategorySelect = "files"
)

type Preview struct {
	SpaceEnabled bool `json:"spaceEnabled"`
}

type Feedback struct {
	CopySound bool `json:"copySound"`
}

// Default returns the published defaults.
func Default() Settings {
	return Settings{
		General:    General{TrayIcon: true},
		Appearance: Appearance{Theme: ThemeAuto, Language: LanguageAuto},
		Shortcuts:  Shortcuts{OpenClipboard: "Alt+C", OpenPreference: "Alt+X"},
		Clipboard: Clipboard{
			Capture: Capture{
				Text: true, HTML: true, RTF: true, Image: true, Files: true,
				MaxTextMB: 4, MaxImageMB: 100,
				Order: DefaultCaptureOrder(),
			},
			Content: Content{
				AutoPaste:                         AutoPasteDoubleClickPaste,
				ShowOriginalPreview:               true,
				DeleteConfirm:                     true,
				DeleteFavoriteConfirm:             true,
				DeletePinnedConfirm:               true,
				DeleteFavoriteItemsOnlyInFavGroup: true,
				Sort:                              SortUpdatedAt,
				ItemActions:                       []ItemAction{ActionCopy, ActionStar, ActionPin, ActionDelete},
				ItemActionOrder: []ItemAction{
					ActionPaste, ActionCopy, ActionCopyPlain,
					ActionOpenLink, ActionSendEmail, ActionReveal, ActionNote, ActionStar, ActionPin, ActionDelete,
				},
			},
			Display:   Display{TextMaxLines: 3, ImageMaxHeight: 64, FileMaxCount: 3},
			Sensitive: Sensitive{CollectSecrets: true, RedactSecrets: true},
			History:   History{Retention: Retention{Unit: RetentionForever}, MaxCount: 0},
			Search:    Search{ClearOnHide: true},
			Window: Window{
				Position:             PositionFollowCursor,
				ScrollToTopOnOpen:    true,
				SelectRangeOnOpen:    RangePreserve,
				SelectCategoryOnOpen: CategoryPreserve,
				SelectGroupOnOpen:    "preserve",
			},
			Preview:    Preview{SpaceEnabled: true},
			Feedback:   Feedback{CopySound: false},
			SourceApps: SourceApps{},
		},
		Logging: Logging{Level: LogLevelError},
	}
}

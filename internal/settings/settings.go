package settings

// Settings mirrors config/settings.json. Every field is decoded over the
// defaults, so an older file that lacks a field keeps the new default.
type Settings struct {
	General    General    `json:"general"`
	Appearance Appearance `json:"appearance"`
	Shortcuts  Shortcuts  `json:"shortcuts"`
	Clipboard  Clipboard  `json:"clipboard"`
	Onboarding Onboarding `json:"onboarding"`
	Update     Update     `json:"update"`
}

type General struct {
	AutoStart  bool `json:"autoStart"`
	RunAsAdmin bool `json:"runAsAdmin"`
	TrayIcon   bool `json:"trayIcon"`
	DockIcon   bool `json:"dockIcon"`
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
	LanguageZhCN Language = "zh-CN"
	LanguageEnUS Language = "en-US"
)

type Shortcuts struct {
	OpenClipboard  string `json:"openClipboard"`
	OpenPreference string `json:"openPreference"`
	WinV           bool   `json:"winV"`
}

type Clipboard struct {
	Capture   Capture   `json:"capture"`
	Content   Content   `json:"content"`
	Display   Display   `json:"display"`
	Sensitive Sensitive `json:"sensitive"`
	History   History   `json:"history"`
	Search    Search    `json:"search"`
	Window    Window    `json:"window"`
	Preview   Preview   `json:"preview"`
	Feedback  Feedback  `json:"feedback"`
	Filters   Filters   `json:"filters"`
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

// DefaultCaptureOrder preserves the published preference: files, image,
// HTML, RTF, then plain text.
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
	MiddleClick                       MiddleClick  `json:"middleClick"`
	CopyPlain                         bool         `json:"copyPlain"`
	CopyThenHideWindow                bool         `json:"copyThenHideWindow"`
	PastePlain                        bool         `json:"pastePlain"`
	PasteFilesAsPath                  bool         `json:"pasteFilesAsPath"`
	ShowOriginalPreview               bool         `json:"showOriginalPreview"`
	DeleteConfirm                     bool         `json:"deleteConfirm"`
	DeleteFavoriteItems               bool         `json:"deleteFavoriteItems"`
	DeleteFavoriteConfirm             bool         `json:"deleteFavoriteConfirm"`
	DeletePinnedItems                 bool         `json:"deletePinnedItems"`
	DeletePinnedConfirm               bool         `json:"deletePinnedConfirm"`
	DeleteFavoriteItemsOnlyInFavGroup bool         `json:"deleteFavoriteItemsOnlyInFavoriteGroup"`
	AutoFavorite                      bool         `json:"autoFavorite"`
	UpdateOnReuse                     bool         `json:"updateOnReuse"`
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

type MiddleClick string

const (
	MiddleClickDisabled          MiddleClick = "disabled"
	MiddleClickSingleClickPaste  MiddleClick = "singleClickPaste"
	MiddleClickSingleClickPlain  MiddleClick = "singleClickPastePlain"
	MiddleClickSingleClickCopy   MiddleClick = "singleClickCopy"
	MiddleClickSingleClickCopyPl MiddleClick = "singleClickCopyPlain"
)

type Sort string

const (
	SortCreatedAt Sort = "createdAtDesc"
	SortUpdatedAt Sort = "updatedAtDesc"
	SortUseCount  Sort = "useCountDesc"
)

type ItemAction string

const (
	ActionPaste      ItemAction = "paste"
	ActionPastePlain ItemAction = "pastePlain"
	ActionPastePath  ItemAction = "pastePath"
	ActionCopy       ItemAction = "copy"
	ActionCopyPlain  ItemAction = "copyPlain"
	ActionOpenLink   ItemAction = "openLink"
	ActionSendEmail  ItemAction = "sendEmail"
	ActionReveal     ItemAction = "reveal"
	ActionNote       ItemAction = "note"
	ActionStar       ItemAction = "star"
	ActionPin        ItemAction = "pinItem"
	ActionDelete     ItemAction = "delete"
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
	Retention            Retention `json:"retention"`
	MaxCount             uint32    `json:"maxCount"`
	CleanupIntervalHours uint32    `json:"cleanupIntervalHours"`
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
	DefaultFocus bool `json:"defaultFocus"`
	ClearOnHide  bool `json:"clearOnHide"`
}

type Window struct {
	Position             WindowPosition `json:"position"`
	ScrollToTopOnOpen    bool           `json:"scrollToTopOnOpen"`
	SelectRangeOnOpen    RangeSelection `json:"selectRangeOnOpen"`
	SelectCategoryOnOpen CategorySelect `json:"selectCategoryOnOpen"`
	SelectGroupOnOpen    string         `json:"selectGroupOnOpen"`
	LightweightMode      bool           `json:"lightweightMode"`
	IdleDestroySeconds   uint32         `json:"idleDestroySeconds"`
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
	HoverEnabled bool   `json:"hoverEnabled"`
	HoverDelayMs uint32 `json:"hoverDelayMs"`
	SpaceEnabled bool   `json:"spaceEnabled"`
}

type Feedback struct {
	CopySound bool `json:"copySound"`
}

type Filters struct {
	ExcludedAppIDs []string `json:"excludedAppIds"`
}

type Onboarding struct {
	Completed    bool                   `json:"completed"`
	LastStep     uint32                 `json:"lastStep"`
	LegacyImport OnboardingLegacyImport `json:"legacyImport"`
}

type OnboardingLegacyImport struct {
	Checked     bool     `json:"checked"`
	Imported    bool     `json:"imported"`
	ImportTypes []string `json:"importTypes"`
	ImportedAt  *string  `json:"importedAt"`
}

type Update struct {
	AutoCheck      bool            `json:"autoCheck"`
	Frequency      UpdateFrequency `json:"frequency"`
	IncludeBeta    bool            `json:"includeBeta"`
	IncludeNightly bool            `json:"includeNightly"`
	LastCheckedAt  *string         `json:"lastCheckedAt"`
	SkippedVersion *string         `json:"skippedVersion"`
}

type UpdateFrequency string

const (
	UpdateDaily   UpdateFrequency = "daily"
	UpdateWeekly  UpdateFrequency = "weekly"
	UpdateMonthly UpdateFrequency = "monthly"
)

// Default returns the published defaults.
func Default() Settings {
	return Settings{
		General:    General{TrayIcon: true, DockIcon: false},
		Appearance: Appearance{Theme: ThemeAuto, Language: LanguageZhCN},
		Shortcuts:  Shortcuts{OpenClipboard: "Alt+C", OpenPreference: "Alt+X"},
		Clipboard: Clipboard{
			Capture: Capture{
				Text: true, HTML: true, RTF: true, Image: true, Files: true,
				MaxTextMB: 4, MaxImageMB: 100,
				Order: DefaultCaptureOrder(),
			},
			Content: Content{
				AutoPaste:                         AutoPasteDoubleClickPaste,
				MiddleClick:                       MiddleClickDisabled,
				ShowOriginalPreview:               true,
				DeleteConfirm:                     true,
				DeleteFavoriteConfirm:             true,
				DeletePinnedConfirm:               true,
				DeleteFavoriteItemsOnlyInFavGroup: true,
				Sort:                              SortUpdatedAt,
				ItemActions:                       []ItemAction{ActionCopy, ActionStar, ActionPin, ActionDelete},
				ItemActionOrder: []ItemAction{
					ActionPaste, ActionPastePlain, ActionPastePath, ActionCopy, ActionCopyPlain,
					ActionOpenLink, ActionSendEmail, ActionReveal, ActionNote, ActionStar, ActionPin, ActionDelete,
				},
			},
			Display:   Display{TextMaxLines: 3, ImageMaxHeight: 64, FileMaxCount: 3},
			Sensitive: Sensitive{CollectSecrets: true, RedactSecrets: true},
			History:   History{Retention: Retention{Unit: RetentionForever}, MaxCount: 0, CleanupIntervalHours: 0},
			Search:    Search{DefaultFocus: false, ClearOnHide: true},
			Window: Window{
				Position:             PositionFollowCursor,
				ScrollToTopOnOpen:    true,
				SelectRangeOnOpen:    RangePreserve,
				SelectCategoryOnOpen: CategoryPreserve,
				SelectGroupOnOpen:    "preserve",
				LightweightMode:      true,
				IdleDestroySeconds:   60,
			},
			Preview:  Preview{HoverEnabled: false, HoverDelayMs: 500, SpaceEnabled: true},
			Feedback: Feedback{CopySound: false},
			Filters:  Filters{ExcludedAppIDs: defaultExcludedApps()},
		},
		Update: Update{Frequency: UpdateDaily},
	}
}

func defaultExcludedApps() []string {
	return []string{"com.apple.keychainaccess", "com.apple.Passwords"}
}

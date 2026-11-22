package settings

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

type Filters struct {
	ExcludedAppIDs []string `json:"excludedAppIds"`
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
			Preview:  Preview{SpaceEnabled: true},
			Feedback: Feedback{CopySound: false},
			Filters:  Filters{ExcludedAppIDs: defaultExcludedApps()},
		},
		Logging: Logging{Level: LogLevelError},
	}
}

func defaultExcludedApps() []string {
	return []string{"com.apple.keychainaccess", "com.apple.Passwords"}
}

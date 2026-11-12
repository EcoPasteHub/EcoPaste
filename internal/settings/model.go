package settings

import (
	"os"
	"strings"
)

const (
	WindowOpenSelectionPreserve = "preserve"
	WindowOpenSelectionAll      = "all"
	WindowOpenGroupPrefix       = "group:"
)

// Settings mirrors the persisted camelCase settings document.
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
	Theme    string `json:"theme"`
	Language string `json:"language"`
}

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
	Text       bool     `json:"text"`
	HTML       bool     `json:"html"`
	RTF        bool     `json:"rtf"`
	Image      bool     `json:"image"`
	Files      bool     `json:"files"`
	MaxTextMB  uint32   `json:"maxTextMb"`
	MaxImageMB uint32   `json:"maxImageMb"`
	Order      []string `json:"order"`
}

type Content struct {
	AutoPaste                              string   `json:"autoPaste"`
	MiddleClick                            string   `json:"middleClick"`
	CopyPlain                              bool     `json:"copyPlain"`
	CopyThenHideWindow                     bool     `json:"copyThenHideWindow"`
	PastePlain                             bool     `json:"pastePlain"`
	PasteFilesAsPath                       bool     `json:"pasteFilesAsPath"`
	ShowOriginalPreview                    bool     `json:"showOriginalPreview"`
	DeleteConfirm                          bool     `json:"deleteConfirm"`
	DeleteFavoriteItems                    bool     `json:"deleteFavoriteItems"`
	DeleteFavoriteConfirm                  bool     `json:"deleteFavoriteConfirm"`
	DeletePinnedItems                      bool     `json:"deletePinnedItems"`
	DeletePinnedConfirm                    bool     `json:"deletePinnedConfirm"`
	DeleteFavoriteItemsOnlyInFavoriteGroup bool     `json:"deleteFavoriteItemsOnlyInFavoriteGroup"`
	AutoFavorite                           bool     `json:"autoFavorite"`
	UpdateOnReuse                          bool     `json:"updateOnReuse"`
	Sort                                   string   `json:"sort"`
	ItemActions                            []string `json:"itemActions"`
	ItemActionOrder                        []string `json:"itemActionOrder"`
}

type Display struct {
	TextMaxLines   uint8  `json:"textMaxLines"`
	ImageMaxHeight uint16 `json:"imageMaxHeight"`
	FileMaxCount   uint8  `json:"fileMaxCount"`
}

type Sensitive struct {
	CollectSecrets bool `json:"collectSecrets"`
	RedactSecrets  bool `json:"redactSecrets"`
}

type Filters struct {
	ExcludedAppIDs []string `json:"excludedAppIds"`
}

type History struct {
	Retention            Retention `json:"retention"`
	MaxCount             uint32    `json:"maxCount"`
	CleanupIntervalHours uint32    `json:"cleanupIntervalHours"`
}

type Retention struct {
	Value uint32 `json:"value"`
	Unit  string `json:"unit"`
}

type Search struct {
	DefaultFocus bool `json:"defaultFocus"`
	ClearOnHide  bool `json:"clearOnHide"`
}

type Window struct {
	Position             string `json:"position"`
	ScrollToTopOnOpen    bool   `json:"scrollToTopOnOpen"`
	SelectRangeOnOpen    string `json:"selectRangeOnOpen"`
	SelectCategoryOnOpen string `json:"selectCategoryOnOpen"`
	SelectGroupOnOpen    string `json:"selectGroupOnOpen"`
	LightweightMode      bool   `json:"lightweightMode"`
	IdleDestroySeconds   uint32 `json:"idleDestroySeconds"`
}

type Preview struct {
	HoverEnabled bool   `json:"hoverEnabled"`
	HoverDelayMS string `json:"hoverDelayMs"`
	SpaceEnabled bool   `json:"spaceEnabled"`
}

type Feedback struct {
	CopySound bool `json:"copySound"`
}

type Update struct {
	AutoCheck      bool    `json:"autoCheck"`
	Frequency      string  `json:"frequency"`
	IncludeBeta    bool    `json:"includeBeta"`
	IncludeNightly bool    `json:"includeNightly"`
	LastCheckedAt  *string `json:"lastCheckedAt"`
	SkippedVersion *string `json:"skippedVersion"`
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

// Default returns the source-compatible defaults, including the macOS
// sensitive-app exclusions.
func Default() Settings {
	return Settings{
		General: General{
			TrayIcon: true,
		},
		Appearance: Appearance{
			Theme:    "auto",
			Language: defaultLanguage(),
		},
		Shortcuts: Shortcuts{
			OpenClipboard:  "Alt+C",
			OpenPreference: "Alt+X",
		},
		Clipboard: Clipboard{
			Capture: Capture{
				Text:       true,
				HTML:       true,
				RTF:        true,
				Image:      true,
				Files:      true,
				MaxTextMB:  4,
				MaxImageMB: 100,
				Order:      []string{"files", "image", "html", "rtf", "text"},
			},
			Content: Content{
				AutoPaste:                              "doubleClickPaste",
				MiddleClick:                            "disabled",
				ShowOriginalPreview:                    true,
				DeleteConfirm:                          true,
				DeleteFavoriteConfirm:                  true,
				DeletePinnedConfirm:                    true,
				DeleteFavoriteItemsOnlyInFavoriteGroup: true,
				Sort:                                   "updatedAtDesc",
				ItemActions:                            []string{"copy", "star", "pinItem", "delete"},
				ItemActionOrder: []string{
					"paste", "pastePlain", "pastePath", "copy", "copyPlain",
					"openLink", "sendEmail", "reveal", "note", "star", "pinItem", "delete",
				},
			},
			Display: Display{
				TextMaxLines:   3,
				ImageMaxHeight: 64,
				FileMaxCount:   3,
			},
			Sensitive: Sensitive{
				CollectSecrets: true,
				RedactSecrets:  true,
			},
			History: History{
				Retention: Retention{Unit: "forever"},
			},
			Search: Search{
				ClearOnHide: true,
			},
			Window: Window{
				Position:             "followCursor",
				ScrollToTopOnOpen:    true,
				SelectRangeOnOpen:    "preserve",
				SelectCategoryOnOpen: "preserve",
				SelectGroupOnOpen:    WindowOpenSelectionPreserve,
				LightweightMode:      true,
				IdleDestroySeconds:   60,
			},
			Preview: Preview{
				HoverDelayMS: "ms500",
				SpaceEnabled: true,
			},
			Filters: Filters{
				ExcludedAppIDs: []string{"com.apple.keychainaccess", "com.apple.Passwords"},
			},
		},
		Onboarding: Onboarding{
			LegacyImport: OnboardingLegacyImport{ImportTypes: []string{}},
		},
		Update: Update{
			Frequency: "daily",
		},
	}
}

func defaultLanguage() string {
	locale := os.Getenv("LC_ALL")
	if locale == "" {
		locale = os.Getenv("LANG")
	}
	if strings.HasPrefix(strings.ToLower(locale), "zh") {
		return "zh-CN"
	}
	return "en-US"
}

package ui

import (
	"context"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
)

// HistoryItem is one row the list can show without knowing how history
// is stored.
type HistoryItem struct {
	ID            string
	GroupID       string
	Kind          string
	SubKind       string
	Title         string
	SourceApp     string
	SourceAppIcon string
	FileSummary   string
	Note          string
	ImagePath     string
	CreatedAt     time.Time
	Favorite      bool
	Pinned        bool
}

// HistoryPage is one page of history rows.
type HistoryPage struct {
	Items   []HistoryItem
	Total   int64
	HasMore bool
}

// HistoryQuery selects a page of history.
type HistoryQuery struct {
	Keyword string
	Filter  string
	GroupID string
	Limit   int
	Offset  int
}

// GroupOption is one user-created group shown in the panel.
type GroupOption struct {
	ID   string
	Name string
}

// HistoryPreview is the full content behind one list row.
type HistoryPreview struct {
	Enabled   bool
	Title     string
	Meta      string
	Text      string
	ImagePath string
}

// HistorySearchBehavior controls where focus starts and whether a hidden
// panel keeps its query.
type HistorySearchBehavior struct {
	DefaultFocus bool
	ClearOnHide  bool
}

// HistoryDisplay is the list density taken from clipboard.display.
type HistoryDisplay struct {
	TextMaxLines   int
	ImageMaxHeight int
	FileMaxCount   int
}

// HistoryOpenBehavior selects the initial list state when the panel opens.
// Filter is "preserve", "" (all), "favorite", "text", "image" or "files";
// GroupID is "preserve", "" (all groups) or a group id.
type HistoryOpenBehavior struct {
	ScrollTop bool
	Filter    string
	GroupID   string
}

// HistoryModel is the app service the history panel consumes. Copy
// reports whether the panel should close after copying.
type HistoryModel interface {
	List(context.Context, HistoryQuery) (HistoryPage, error)
	Copy(context.Context, string) (hide bool, err error)
	CopyPlain(context.Context, string) (hide bool, err error)
	Activate(context.Context, string) (paste bool, hide bool, err error)
	ActivatePlain(context.Context, string) (paste bool, hide bool, err error)
	ActivateSingleClick(context.Context, string) (paste bool, hide bool, err error)
	ActivateDoubleClick(context.Context, string) (paste bool, hide bool, err error)
	MiddleClick(context.Context, string) (hide bool, err error)
	SendPaste() error
	NeedsAccessibility(error) bool
	ToggleFavorite(context.Context, string) (bool, error)
	TogglePinned(context.Context, string) (bool, error)
	Delete(context.Context, string) error
	Open(context.Context, string) error
	Reveal(context.Context, string) error
	SaveImage(context.Context, string) error
	Drag(context.Context, string) (macos.DragPayload, error)
	EditNote(context.Context, string) (bool, error)
	Preview(context.Context, string) (HistoryPreview, error)
	Groups(context.Context) ([]GroupOption, error)
	CreateGroup(context.Context) (*GroupOption, bool, error)
	RenameGroup(context.Context, string) (bool, error)
	DeleteGroup(context.Context, string) (bool, error)
	MoveToGroup(context.Context, string, string) error
	SearchBehavior() HistorySearchBehavior
	Display() HistoryDisplay
	OpenBehavior() HistoryOpenBehavior
	QuickActions() []string
	Updates() <-chan struct{}
}

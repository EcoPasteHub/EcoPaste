package ui

import (
	"context"
	"time"
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

// HistoryModel is the app service the history panel consumes. Copy
// reports whether the panel should close after copying.
type HistoryModel interface {
	List(context.Context, HistoryQuery) (HistoryPage, error)
	Copy(context.Context, string) (hide bool, err error)
	CopyPlain(context.Context, string) (hide bool, err error)
	Activate(context.Context, string) (paste bool, hide bool, err error)
	ActivatePlain(context.Context, string) (paste bool, hide bool, err error)
	SendPaste() error
	NeedsAccessibility(error) bool
	ToggleFavorite(context.Context, string) (bool, error)
	TogglePinned(context.Context, string) (bool, error)
	Delete(context.Context, string) error
	Open(context.Context, string) error
	Reveal(context.Context, string) error
	SaveImage(context.Context, string) error
	EditNote(context.Context, string) (bool, error)
	Preview(context.Context, string) (HistoryPreview, error)
	Groups(context.Context) ([]GroupOption, error)
	CreateGroup(context.Context) (*GroupOption, bool, error)
	RenameGroup(context.Context, string) (bool, error)
	DeleteGroup(context.Context, string) (bool, error)
	MoveToGroup(context.Context, string, string) error
	Updates() <-chan struct{}
}

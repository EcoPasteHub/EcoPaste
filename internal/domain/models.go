package domain

import "time"

// Kind is the top-level clipboard payload type stored in clipboard_items.kind.
type Kind string

const (
	KindText  Kind = "text"
	KindImage Kind = "image"
	KindFiles Kind = "files"
)

// Sort selects the list order used by the clipboard history.
type Sort string

const (
	SortCreatedAt Sort = "createdAtDesc"
	SortUpdatedAt Sort = "updatedAtDesc"
	SortUseCount  Sort = "useCountDesc"
)

// Item is one row of clipboard_items.
type Item struct {
	ID             string
	Kind           Kind
	SubKind        string
	GroupID        string
	SourceAppID    string
	Content        string
	ContentHash    string
	SearchText     string
	Summary        string
	FileTypes      string
	Size           int64
	Width          int64
	Height         int64
	UseCount       int64
	IsFavorite     bool
	IsPinned       bool
	IsSensitive    bool
	Platform       string
	Note           string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	SourceAppName  string
	SourceAppIcon  string
	ImageThumbnail string
}

// Group is a user-created clipboard group.
type Group struct {
	ID        string
	Name      string
	Icon      string
	IsHidden  bool
	SortOrder int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

// App is a clipboard source application.
type App struct {
	ID        string
	Name      string
	IconFile  string
	Platform  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Query describes one page of clipboard history.
type Query struct {
	Keyword  string
	GroupID  string
	Kind     Kind
	Favorite *bool
	Pinned   *bool
	Sort     Sort
	Limit    int
	Offset   int
}

// Page is a history page plus the total matching count.
type Page struct {
	Items  []Item
	Total  int64
	Limit  int
	Offset int
}

package database

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/zeebo/blake3"
)

// Kind is the main content type of a clipboard item.
type Kind string

const (
	KindText  Kind = "text"
	KindImage Kind = "image"
	KindFiles Kind = "files"
)

// SubKind refines a text item.
type SubKind string

const (
	SubRTF   SubKind = "rtf"
	SubHTML  SubKind = "html"
	SubURL   SubKind = "url"
	SubEmail SubKind = "email"
	SubColor SubKind = "color"
	SubPath  SubKind = "path"
)

// Item is one clipboard history row.
type Item struct {
	ID            string    `json:"id"`
	Kind          Kind      `json:"kind"`
	SubKind       *SubKind  `json:"subKind,omitempty"`
	GroupID       *string   `json:"groupId,omitempty"`
	SourceAppID   *string   `json:"sourceAppId,omitempty"`
	Content       string    `json:"content"`
	ContentHash   string    `json:"contentHash"`
	SearchText    *string   `json:"searchText,omitempty"`
	Summary       *string   `json:"summary,omitempty"`
	FileTypes     *string   `json:"fileTypes,omitempty"`
	Size          *int64    `json:"size,omitempty"`
	Width         *int64    `json:"width,omitempty"`
	Height        *int64    `json:"height,omitempty"`
	UseCount      int64     `json:"useCount"`
	IsFavorite    bool      `json:"isFavorite"`
	IsPinned      bool      `json:"isPinned"`
	IsSensitive   bool      `json:"isSensitive"`
	Platform      string    `json:"platform"`
	Note          *string   `json:"note,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	SourceAppName *string   `json:"sourceAppName,omitempty"`
	SourceAppIcon *string   `json:"sourceAppIconFile,omitempty"`
}

// Sort selects the history order. Pinned items always come first.
type Sort string

const (
	SortCreatedAt Sort = "createdAtDesc"
	SortUpdatedAt Sort = "updatedAtDesc"
	SortUseCount  Sort = "useCountDesc"
)

// Query filters one page of history.
type Query struct {
	Kind     *Kind
	GroupID  *string
	Group    string
	Favorite *bool
	Pinned   *bool
	Keyword  string
	Sort     Sort
	Limit    int
	Offset   int
}

// Page is one history page with the total matching the same filters.
type Page struct {
	Items   []Item `json:"items"`
	Total   int64  `json:"total"`
	HasMore bool   `json:"hasMore"`
}

// Hash returns the published dedupe key, blake3("<kind>:<content>").
func Hash(kind Kind, content string) string {
	hasher := blake3.New()
	_, _ = hasher.Write([]byte(kind))
	_, _ = hasher.Write([]byte(":"))
	_, _ = hasher.Write([]byte(content))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

// NewID returns a UUID v4 string, the shape legacy rows use.
func NewID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// Upsert inserts item, or when its content hash already exists bumps the
// existing row's use count and updated_at. The bool reports a dedupe hit.
func (d *DB) Upsert(ctx context.Context, item *Item) (id string, deduplicated bool, err error) {
	if item.ContentHash == "" {
		item.ContentHash = Hash(item.Kind, item.Content)
	}
	existing, err := d.FindByHash(ctx, item.ContentHash)
	if err != nil {
		return "", false, err
	}
	if existing != nil {
		if err := d.IncrementUseCount(ctx, existing.ID); err != nil {
			return "", false, err
		}
		return existing.ID, true, nil
	}
	if item.ID == "" {
		item.ID = NewID()
	}
	if item.Platform == "" {
		item.Platform = "macos"
	}
	if item.UseCount == 0 {
		item.UseCount = 1
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now().UTC()
	}
	if item.UpdatedAt.IsZero() {
		item.UpdatedAt = item.CreatedAt
	}
	const insert = `INSERT INTO clipboard_items
		(id, kind, sub_kind, group_id, source_app_id, content, content_hash, search_text,
		 summary, file_types, size, width, height, use_count, is_favorite, is_pinned,
		 is_sensitive, platform, note, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	if _, err := d.sql.ExecContext(ctx, insert,
		item.ID, item.Kind, item.SubKind, item.GroupID, item.SourceAppID, item.Content,
		item.ContentHash, item.SearchText, item.Summary, item.FileTypes, item.Size,
		item.Width, item.Height, item.UseCount, item.IsFavorite, item.IsPinned,
		item.IsSensitive, item.Platform, item.Note,
		item.CreatedAt.UTC().Format(time.RFC3339Nano), item.UpdatedAt.UTC().Format(time.RFC3339Nano),
	); err != nil {
		return "", false, fmt.Errorf("insert clipboard item: %w", err)
	}
	return item.ID, false, nil
}

// IncrementUseCount marks a row as reused: one more use, a fresh
// updated_at and no new row.
func (d *DB) IncrementUseCount(ctx context.Context, id string) error {
	_, err := d.sql.ExecContext(ctx,
		"UPDATE clipboard_items SET use_count = use_count + 1, updated_at = ? WHERE id = ?", now(), id)
	if err != nil {
		return fmt.Errorf("increment item use count: %w", err)
	}
	return nil
}

const itemColumns = `id, kind, sub_kind, group_id, source_app_id, content, content_hash,
	search_text, summary, file_types, size, width, height, use_count, is_favorite,
	is_pinned, is_sensitive, platform, note, created_at, updated_at`

const listColumns = `clipboard_items.id, clipboard_items.kind, clipboard_items.sub_kind,
	clipboard_items.group_id, clipboard_items.source_app_id,
	CASE WHEN clipboard_items.kind = 'text' THEN '' ELSE clipboard_items.content END,
	clipboard_items.content_hash,
	CASE WHEN clipboard_items.kind = 'text' THEN NULL ELSE clipboard_items.search_text END,
	clipboard_items.summary, clipboard_items.file_types, clipboard_items.size,
	clipboard_items.width, clipboard_items.height, clipboard_items.use_count,
	clipboard_items.is_favorite, clipboard_items.is_pinned, clipboard_items.is_sensitive,
	clipboard_items.platform, clipboard_items.note, clipboard_items.created_at,
	clipboard_items.updated_at`

// FindByID returns the complete row, including content and search text.
func (d *DB) FindByID(ctx context.Context, id string) (*Item, error) {
	row := d.sql.QueryRowContext(ctx, "SELECT "+itemColumns+" FROM clipboard_items WHERE id = ?", id)
	item, err := scanItem(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find clipboard item: %w", err)
	}
	return item, nil
}

// FindByHash returns the newest row with the given content hash.
func (d *DB) FindByHash(ctx context.Context, hash string) (*Item, error) {
	row := d.sql.QueryRowContext(ctx,
		"SELECT "+itemColumns+" FROM clipboard_items WHERE content_hash = ? ORDER BY created_at DESC LIMIT 1", hash)
	item, err := scanItem(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find clipboard item by hash: %w", err)
	}
	return item, nil
}

// Page returns one page and the total that matches its filters. Text
// content is omitted from list rows; use FindByID for the full value.
func (d *DB) Page(ctx context.Context, q Query) (Page, error) {
	where, args := filterSQL(q)
	limit := q.Limit
	if limit <= 0 {
		limit = 50
	}

	order := "clipboard_items.is_pinned DESC, "
	switch q.Sort {
	case SortCreatedAt:
		order += "clipboard_items.created_at DESC"
	case SortUseCount:
		order += "clipboard_items.use_count DESC, clipboard_items.created_at DESC"
	default:
		order += "clipboard_items.updated_at DESC, clipboard_items.created_at DESC"
	}
	query := "SELECT " + listColumns + ", clipboard_apps.name FROM clipboard_items" +
		" LEFT JOIN clipboard_apps ON clipboard_apps.id = clipboard_items.source_app_id" +
		where + " ORDER BY " + order + " LIMIT ? OFFSET ?"
	rows, err := d.sql.QueryContext(ctx, query, append(args, limit, q.Offset)...)
	if err != nil {
		return Page{}, fmt.Errorf("query clipboard items: %w", err)
	}
	defer rows.Close()

	var page Page
	for rows.Next() {
		item, err := scanListItem(rows)
		if err != nil {
			return Page{}, fmt.Errorf("scan clipboard item: %w", err)
		}
		page.Items = append(page.Items, *item)
	}
	if err := rows.Err(); err != nil {
		return Page{}, fmt.Errorf("read clipboard items: %w", err)
	}

	countQuery := "SELECT COUNT(*) FROM clipboard_items" + where
	if err := d.sql.QueryRowContext(ctx, countQuery, args...).Scan(&page.Total); err != nil {
		return Page{}, fmt.Errorf("count clipboard items: %w", err)
	}
	page.HasMore = int64(q.Offset+len(page.Items)) < page.Total
	return page, nil
}

func filterSQL(q Query) (string, []any) {
	clauses := []string{"1 = 1"}
	var args []any
	keyword := strings.TrimSpace(q.Keyword)

	switch q.Group {
	case "text", "image", "files":
		clauses = append(clauses, "clipboard_items.kind = ?")
		args = append(args, q.Group)
	case "favorite":
		clauses = append(clauses, "clipboard_items.is_favorite = 1")
	}
	if q.Kind != nil {
		clauses = append(clauses, "clipboard_items.kind = ?")
		args = append(args, *q.Kind)
	}
	if q.GroupID != nil {
		clauses = append(clauses, "clipboard_items.group_id = ?")
		args = append(args, *q.GroupID)
	}
	if q.Favorite != nil {
		clauses = append(clauses, "clipboard_items.is_favorite = ?")
		args = append(args, *q.Favorite)
	}
	if q.Pinned != nil {
		clauses = append(clauses, "clipboard_items.is_pinned = ?")
		args = append(args, *q.Pinned)
	}
	if keyword != "" {
		if expr, ok := ftsExpr(keyword); ok {
			clauses = append(clauses, "clipboard_items.rowid IN (SELECT rowid FROM clipboard_items_fts WHERE clipboard_items_fts MATCH ?)")
			args = append(args, expr)
		} else {
			escaped := escapeLike(keyword)
			clauses = append(clauses, `(clipboard_items.search_text LIKE ? ESCAPE '\' OR clipboard_items.note LIKE ? ESCAPE '\')`)
			args = append(args, "%"+escaped+"%", "%"+escaped+"%")
		}
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

// ftsExpr builds a quoted prefix expression when every whitespace token
// is at least three characters, the trigram index's minimum.
func ftsExpr(keyword string) (string, bool) {
	tokens := strings.Fields(keyword)
	if len(tokens) == 0 {
		return "", false
	}
	for _, token := range tokens {
		if utf8.RuneCountInString(token) < 3 {
			return "", false
		}
	}
	quoted := make([]string, 0, len(tokens))
	for _, token := range tokens {
		quoted = append(quoted, `"`+strings.ReplaceAll(token, `"`, `""`)+`"*`)
	}
	return strings.Join(quoted, " "), true
}

func escapeLike(keyword string) string {
	var b strings.Builder
	for _, r := range keyword {
		if r == '\\' || r == '%' || r == '_' {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// ToggleFavorite flips the favorite flag without touching updated_at.
func (d *DB) ToggleFavorite(ctx context.Context, id string) (bool, error) {
	var value bool
	err := d.sql.QueryRowContext(ctx,
		"UPDATE clipboard_items SET is_favorite = NOT is_favorite WHERE id = ? RETURNING is_favorite", id).Scan(&value)
	if err != nil {
		return false, fmt.Errorf("toggle item favorite: %w", err)
	}
	return value, nil
}

// TogglePinned flips the pinned flag without touching updated_at.
func (d *DB) TogglePinned(ctx context.Context, id string) (bool, error) {
	var value bool
	err := d.sql.QueryRowContext(ctx,
		"UPDATE clipboard_items SET is_pinned = NOT is_pinned WHERE id = ? RETURNING is_pinned", id).Scan(&value)
	if err != nil {
		return false, fmt.Errorf("toggle item pinned: %w", err)
	}
	return value, nil
}

// SetNote replaces an item's note without touching updated_at.
func (d *DB) SetNote(ctx context.Context, id string, note *string) error {
	_, err := d.sql.ExecContext(ctx, "UPDATE clipboard_items SET note = ? WHERE id = ?", note, id)
	if err != nil {
		return fmt.Errorf("update item note: %w", err)
	}
	return nil
}

// SetGroup moves an item into a group without touching updated_at.
func (d *DB) SetGroup(ctx context.Context, id string, groupID *string) error {
	_, err := d.sql.ExecContext(ctx, "UPDATE clipboard_items SET group_id = ? WHERE id = ?", groupID, id)
	if err != nil {
		return fmt.Errorf("update item group: %w", err)
	}
	return nil
}

// Delete removes one item.
func (d *DB) Delete(ctx context.Context, id string) error {
	if _, err := d.sql.ExecContext(ctx, "DELETE FROM clipboard_items WHERE id = ?", id); err != nil {
		return fmt.Errorf("delete clipboard item: %w", err)
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanItem(row scanner) (*Item, error) {
	return scanItemRow(row, false)
}

func scanListItem(row scanner) (*Item, error) {
	return scanItemRow(row, true)
}

func scanItemRow(row scanner, withAppName bool) (*Item, error) {
	var (
		item                           Item
		subKind, groupID, sourceAppID  sql.NullString
		searchText, summary, fileTypes sql.NullString
		size, width, height            sql.NullInt64
		note, appName                  sql.NullString
		createdAt, updatedAt           string
	)
	dest := []any{
		&item.ID, &item.Kind, &subKind, &groupID, &sourceAppID, &item.Content,
		&item.ContentHash, &searchText, &summary, &fileTypes, &size, &width,
		&height, &item.UseCount, &item.IsFavorite, &item.IsPinned, &item.IsSensitive,
		&item.Platform, &note, &createdAt, &updatedAt,
	}
	if withAppName {
		dest = append(dest, &appName)
	}
	if err := row.Scan(dest...); err != nil {
		return nil, err
	}
	if subKind.Valid {
		value := SubKind(subKind.String)
		item.SubKind = &value
	}
	item.GroupID = nullable(groupID)
	item.SourceAppID = nullable(sourceAppID)
	item.SearchText = nullable(searchText)
	item.Summary = nullable(summary)
	item.FileTypes = nullable(fileTypes)
	item.Note = nullable(note)
	item.SourceAppName = nullable(appName)
	item.Size = nullableInt(size)
	item.Width = nullableInt(width)
	item.Height = nullableInt(height)
	item.CreatedAt = parseTime(createdAt)
	item.UpdatedAt = parseTime(updatedAt)
	return &item, nil
}

func nullable(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
}

func nullableInt(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	return &v.Int64
}

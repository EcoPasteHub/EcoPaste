package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/EcoPasteHub/EcoPaste/internal/domain"
)

// Database is the SQLite compatibility layer for an existing EcoPaste DB.
type Database struct {
	sql *sql.DB
}

// Probe describes the schema found at startup.
type Probe struct {
	Version     int
	JournalMode string
	SQLxVersion int
	SQLxSuccess int
	ItemRows    int64
	FTSRows     int64
	HasHistory  bool
}

// Open opens an existing database read-write. It never replays migrations.
func Open(path string) (*Database, error) {
	dsn := "file:" + url.PathEscape(path) + "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return &Database{sql: db}, nil
}

// Close releases the database handle.
func (d *Database) Close() error {
	if d == nil || d.sql == nil {
		return nil
	}
	return d.sql.Close()
}

// SQL exposes the underlying pool to focused repositories and tests.
func (d *Database) SQL() *sql.DB {
	return d.sql
}

// Probe reads schema identity without modifying the database.
func (d *Database) Probe(ctx context.Context) (Probe, error) {
	var p Probe
	if err := d.sql.QueryRowContext(ctx, "PRAGMA user_version").Scan(&p.Version); err != nil {
		return p, err
	}
	if err := d.sql.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&p.JournalMode); err != nil {
		return p, err
	}
	_ = d.sql.QueryRowContext(ctx,
		"SELECT version, success FROM _sqlx_migrations ORDER BY version DESC LIMIT 1",
	).Scan(&p.SQLxVersion, &p.SQLxSuccess)
	_ = d.sql.QueryRowContext(ctx, "SELECT COUNT(*) FROM clipboard_items").Scan(&p.ItemRows)
	_ = d.sql.QueryRowContext(ctx, "SELECT COUNT(*) FROM clipboard_items_fts").Scan(&p.FTSRows)
	var history int
	_ = d.sql.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='history'",
	).Scan(&history)
	p.HasHistory = history > 0
	return p, nil
}

// QueryPage returns a page and total using the same filter and sort
// semantics as the Rust implementation.
func (d *Database) QueryPage(ctx context.Context, q domain.Query) (domain.Page, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := q.Offset
	if offset < 0 {
		offset = 0
	}

	where, args, err := buildFilter(q)
	if err != nil {
		return domain.Page{}, err
	}

	order := "clipboard_items.is_pinned DESC, "
	switch q.Sort {
	case domain.SortCreatedAt:
		order += "clipboard_items.created_at DESC"
	case domain.SortUseCount:
		order += "clipboard_items.use_count DESC, clipboard_items.created_at DESC"
	default:
		order += "clipboard_items.updated_at DESC, clipboard_items.created_at DESC"
	}

	listSQL := listSelect + where + " ORDER BY " + order + " LIMIT ? OFFSET ?"
	listArgs := append(append([]any{}, args...), limit, offset)
	rows, err := d.sql.QueryContext(ctx, listSQL, listArgs...)
	if err != nil {
		return domain.Page{}, fmt.Errorf("query clipboard items: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Item, 0, limit)
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return domain.Page{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return domain.Page{}, err
	}

	var total int64
	if err := d.sql.QueryRowContext(ctx, countSelect+where, args...).Scan(&total); err != nil {
		return domain.Page{}, fmt.Errorf("count clipboard items: %w", err)
	}
	return domain.Page{Items: items, Total: total, Limit: limit, Offset: offset}, nil
}

// Find returns a full item, including text content omitted from list rows.
func (d *Database) Find(ctx context.Context, id string) (domain.Item, error) {
	row := d.sql.QueryRowContext(ctx, fullSelect+" WHERE clipboard_items.id = ?", id)
	item, err := scanItem(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Item{}, err
	}
	if err != nil {
		return domain.Item{}, err
	}
	return item, nil
}

const listSelect = `SELECT clipboard_items.id, clipboard_items.kind,
	clipboard_items.sub_kind, clipboard_items.group_id, clipboard_items.source_app_id,
	CASE WHEN clipboard_items.kind = 'text' THEN '' ELSE clipboard_items.content END,
	clipboard_items.content_hash,
	CASE WHEN clipboard_items.kind = 'text' THEN NULL ELSE clipboard_items.search_text END,
	clipboard_items.summary, clipboard_items.file_types, clipboard_items.size,
	clipboard_items.width, clipboard_items.height, clipboard_items.use_count,
	clipboard_items.is_favorite, clipboard_items.is_pinned, clipboard_items.is_sensitive,
	clipboard_items.platform, clipboard_items.note, clipboard_items.created_at,
	clipboard_items.updated_at, clipboard_apps.name, clipboard_apps.icon_file
	FROM clipboard_items
	LEFT JOIN clipboard_apps ON clipboard_apps.id = clipboard_items.source_app_id`

const fullSelect = `SELECT clipboard_items.id, clipboard_items.kind,
	clipboard_items.sub_kind, clipboard_items.group_id, clipboard_items.source_app_id,
	clipboard_items.content, clipboard_items.content_hash, clipboard_items.search_text,
	clipboard_items.summary, clipboard_items.file_types, clipboard_items.size,
	clipboard_items.width, clipboard_items.height, clipboard_items.use_count,
	clipboard_items.is_favorite, clipboard_items.is_pinned, clipboard_items.is_sensitive,
	clipboard_items.platform, clipboard_items.note, clipboard_items.created_at,
	clipboard_items.updated_at, clipboard_apps.name, clipboard_apps.icon_file
	FROM clipboard_items
	LEFT JOIN clipboard_apps ON clipboard_apps.id = clipboard_items.source_app_id`

const countSelect = `SELECT COUNT(*) FROM clipboard_items`

type scanner interface {
	Scan(dest ...any) error
}

func scanItem(row scanner) (domain.Item, error) {
	var item domain.Item
	var kind, subKind, groupID, sourceAppID, searchText, summary, fileTypes, note sql.NullString
	var sourceName, sourceIcon sql.NullString
	var size, width, height sql.NullInt64
	var createdAt, updatedAt sql.NullString
	var favorite, pinned, sensitive int

	err := row.Scan(
		&item.ID, &kind, &subKind, &groupID, &sourceAppID, &item.Content,
		&item.ContentHash, &searchText, &summary, &fileTypes, &size, &width, &height,
		&item.UseCount, &favorite, &pinned, &sensitive, &item.Platform,
		&note, &createdAt, &updatedAt, &sourceName, &sourceIcon,
	)
	if err != nil {
		return domain.Item{}, err
	}

	item.Kind = domain.Kind(kind.String)
	item.SubKind = subKind.String
	item.GroupID = groupID.String
	item.SourceAppID = sourceAppID.String
	item.SearchText = searchText.String
	item.Summary = summary.String
	item.FileTypes = fileTypes.String
	item.Size = size.Int64
	item.Width = width.Int64
	item.Height = height.Int64
	item.IsFavorite = favorite != 0
	item.IsPinned = pinned != 0
	item.IsSensitive = sensitive != 0
	item.Note = note.String
	item.SourceAppName = sourceName.String
	item.SourceAppIcon = sourceIcon.String
	item.CreatedAt, err = parseTimestamp(createdAt.String)
	if err != nil {
		return domain.Item{}, err
	}
	item.UpdatedAt, err = parseTimestamp(updatedAt.String)
	if err != nil {
		return domain.Item{}, err
	}
	return item, nil
}

func parseTimestamp(value string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported clipboard timestamp %q", value)
}

func buildFilter(q domain.Query) (string, []any, error) {
	clauses := []string{" WHERE 1 = 1"}
	args := []any{}

	keyword := strings.TrimSpace(q.Keyword)
	if keyword != "" {
		fts, useFTS := keywordClause(keyword)
		if useFTS {
			clauses = append(clauses,
				" AND clipboard_items.rowid IN (SELECT rowid FROM clipboard_items_fts WHERE clipboard_items_fts MATCH ?)")
			args = append(args, fts)
		} else {
			pattern := "%" + escapeLike(keyword) + "%"
			clauses = append(clauses,
				` AND (clipboard_items.search_text LIKE ? ESCAPE '\' OR clipboard_items.note LIKE ? ESCAPE '\')`)
			args = append(args, pattern, pattern)
		}
	}
	if q.Kind != "" {
		clauses = append(clauses, " AND clipboard_items.kind = ?")
		args = append(args, q.Kind)
	}
	if q.GroupID != "" {
		clauses = append(clauses, " AND clipboard_items.group_id = ?")
		args = append(args, q.GroupID)
	}
	if q.Favorite != nil {
		clauses = append(clauses, " AND clipboard_items.is_favorite = ?")
		args = append(args, boolInt(*q.Favorite))
	}
	if q.Pinned != nil {
		clauses = append(clauses, " AND clipboard_items.is_pinned = ?")
		args = append(args, boolInt(*q.Pinned))
	}
	return strings.Join(clauses, ""), args, nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

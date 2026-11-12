CREATE TABLE clipboard_groups (
    id          TEXT    PRIMARY KEY,
    name        TEXT    NOT NULL,
    icon        TEXT    NOT NULL,
    is_hidden   INTEGER NOT NULL DEFAULT 0,
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT    NOT NULL,
    updated_at  TEXT    NOT NULL
);

CREATE TABLE clipboard_apps (
    id          TEXT    PRIMARY KEY,
    name        TEXT    NOT NULL,
    icon_file   TEXT,
    platform    TEXT    NOT NULL,
    created_at  TEXT    NOT NULL,
    updated_at  TEXT    NOT NULL
);

CREATE TABLE clipboard_items (
    id            TEXT    PRIMARY KEY,
    kind          TEXT    NOT NULL,
    sub_kind      TEXT,
    group_id      TEXT    REFERENCES clipboard_groups(id) ON DELETE SET NULL,
    source_app_id TEXT    REFERENCES clipboard_apps(id)   ON DELETE SET NULL,
    content       TEXT    NOT NULL,
    content_hash  TEXT    NOT NULL,
    search_text   TEXT,
    summary       TEXT,
    file_types    TEXT,
    size          INTEGER,
    width         INTEGER,
    height        INTEGER,
    use_count     INTEGER NOT NULL DEFAULT 1,
    is_favorite   INTEGER NOT NULL DEFAULT 0,
    is_pinned     INTEGER NOT NULL DEFAULT 0,
    is_sensitive  INTEGER NOT NULL DEFAULT 0,
    platform      TEXT    NOT NULL,
    note          TEXT,
    created_at    TEXT    NOT NULL,
    updated_at    TEXT    NOT NULL,
    CHECK (kind <> 'files' OR trim(content) <> '')
);

CREATE INDEX idx_clipboard_items_content_hash ON clipboard_items (content_hash);
CREATE INDEX idx_clipboard_items_source_app_id ON clipboard_items (source_app_id);

CREATE TABLE file_type_icons (
    cache_key   TEXT NOT NULL,
    platform    TEXT NOT NULL,
    icon_file   TEXT NOT NULL,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL,
    PRIMARY KEY (cache_key, platform)
);

CREATE VIRTUAL TABLE clipboard_items_fts USING fts5(
    search_text,
    note,
    content='clipboard_items',
    content_rowid='rowid',
    tokenize='trigram'
);

CREATE TRIGGER clipboard_items_ai AFTER INSERT ON clipboard_items BEGIN
    INSERT INTO clipboard_items_fts(rowid, search_text, note)
    VALUES (new.rowid, new.search_text, new.note);
END;

CREATE TRIGGER clipboard_items_ad AFTER DELETE ON clipboard_items BEGIN
    INSERT INTO clipboard_items_fts(clipboard_items_fts, rowid, search_text, note)
    VALUES ('delete', old.rowid, old.search_text, old.note);
END;

CREATE TRIGGER clipboard_items_au AFTER UPDATE ON clipboard_items BEGIN
    INSERT INTO clipboard_items_fts(clipboard_items_fts, rowid, search_text, note)
    VALUES ('delete', old.rowid, old.search_text, old.note);
    INSERT INTO clipboard_items_fts(rowid, search_text, note)
    VALUES (new.rowid, new.search_text, new.note);
END;

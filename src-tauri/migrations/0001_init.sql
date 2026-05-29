CREATE TABLE clipboard_groups (
    id          TEXT    PRIMARY KEY,
    name        TEXT    NOT NULL,
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT    NOT NULL
);

CREATE TABLE clipboard_items (
    id           TEXT    PRIMARY KEY,
    kind         TEXT    NOT NULL,
    sub_kind     TEXT,
    group_id     TEXT    REFERENCES clipboard_groups(id) ON DELETE SET NULL,
    content      TEXT    NOT NULL,
    search_text  TEXT,
    size         INTEGER,
    width        INTEGER,
    height       INTEGER,
    use_count    INTEGER NOT NULL DEFAULT 1,
    is_favorite  INTEGER NOT NULL DEFAULT 0,
    is_pinned    INTEGER NOT NULL DEFAULT 0,
    platform     TEXT    NOT NULL,
    note         TEXT,
    created_at   TEXT    NOT NULL,
    updated_at   TEXT    NOT NULL
);

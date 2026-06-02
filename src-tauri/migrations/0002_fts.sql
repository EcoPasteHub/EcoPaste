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

-- +goose Up

CREATE TABLE IF NOT EXISTS references_t (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_id INTEGER NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    sequence_number INTEGER NOT NULL DEFAULT 0,
    filename TEXT NOT NULL DEFAULT '',
    path TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE VIRTUAL TABLE IF NOT EXISTS refs_fts USING fts5(
    title, summary,
    content=references_t,
    content_rowid=id
);

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS refs_ai AFTER INSERT ON references_t BEGIN
    INSERT INTO refs_fts(rowid, title, summary)
    VALUES (new.id, new.title, new.summary);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS refs_ad AFTER DELETE ON references_t BEGIN
    INSERT INTO refs_fts(refs_fts, rowid, title, summary)
    VALUES('delete', old.id, old.title, old.summary);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS refs_au AFTER UPDATE ON references_t BEGIN
    INSERT INTO refs_fts(refs_fts, rowid, title, summary)
    VALUES('delete', old.id, old.title, old.summary);
    INSERT INTO refs_fts(rowid, title, summary)
    VALUES (new.id, new.title, new.summary);
END;
-- +goose StatementEnd

-- +goose Down

DROP TRIGGER IF EXISTS refs_au;
DROP TRIGGER IF EXISTS refs_ad;
DROP TRIGGER IF EXISTS refs_ai;
DROP TABLE IF EXISTS refs_fts;
DROP TABLE IF EXISTS references_t;

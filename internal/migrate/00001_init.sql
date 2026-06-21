-- +goose Up

CREATE TABLE IF NOT EXISTS workspaces (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    topic TEXT NOT NULL DEFAULT '',
    path TEXT NOT NULL DEFAULT '',
    mission_why TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    last_studied TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS lessons (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_id INTEGER NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    sequence_number INTEGER NOT NULL DEFAULT 0,
    filename TEXT NOT NULL DEFAULT '',
    path TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS learning_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_id INTEGER NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    sequence_number INTEGER NOT NULL DEFAULT 0,
    filename TEXT NOT NULL DEFAULT '',
    path TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active',
    superseded_by INTEGER DEFAULT NULL,
    summary TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    default_view TEXT NOT NULL DEFAULT 'dashboard',
    items_per_page INTEGER NOT NULL DEFAULT 25,
    lessons_dir TEXT NOT NULL DEFAULT 'lessons',
    records_dir TEXT NOT NULL DEFAULT 'learning-records',
    reference_dir TEXT NOT NULL DEFAULT 'reference',
    assets_dir TEXT NOT NULL DEFAULT 'assets'
);

INSERT OR IGNORE INTO settings (id) VALUES (1);

-- FTS5 index for lessons
CREATE VIRTUAL TABLE IF NOT EXISTS lessons_fts USING fts5(
    title, summary,
    content=lessons,
    content_rowid=id
);

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS lessons_ai AFTER INSERT ON lessons BEGIN
    INSERT INTO lessons_fts(rowid, title, summary)
    VALUES (new.id, new.title, new.summary);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS lessons_ad AFTER DELETE ON lessons BEGIN
    INSERT INTO lessons_fts(lessons_fts, rowid, title, summary)
    VALUES('delete', old.id, old.title, old.summary);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS lessons_au AFTER UPDATE ON lessons BEGIN
    INSERT INTO lessons_fts(lessons_fts, rowid, title, summary)
    VALUES('delete', old.id, old.title, old.summary);
    INSERT INTO lessons_fts(rowid, title, summary)
    VALUES (new.id, new.title, new.summary);
END;
-- +goose StatementEnd

-- FTS5 index for learning records
CREATE VIRTUAL TABLE IF NOT EXISTS records_fts USING fts5(
    title, summary,
    content=learning_records,
    content_rowid=id
);

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS records_ai AFTER INSERT ON learning_records BEGIN
    INSERT INTO records_fts(rowid, title, summary)
    VALUES (new.id, new.title, new.summary);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS records_ad AFTER DELETE ON learning_records BEGIN
    INSERT INTO records_fts(records_fts, rowid, title, summary)
    VALUES('delete', old.id, old.title, old.summary);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS records_au AFTER UPDATE ON learning_records BEGIN
    INSERT INTO records_fts(records_fts, rowid, title, summary)
    VALUES('delete', old.id, old.title, old.summary);
    INSERT INTO records_fts(rowid, title, summary)
    VALUES (new.id, new.title, new.summary);
END;
-- +goose StatementEnd

-- +goose Down

DROP TRIGGER IF EXISTS records_au;
DROP TRIGGER IF EXISTS records_ad;
DROP TRIGGER IF EXISTS records_ai;
DROP TABLE IF EXISTS records_fts;

DROP TRIGGER IF EXISTS lessons_au;
DROP TRIGGER IF EXISTS lessons_ad;
DROP TRIGGER IF EXISTS lessons_ai;
DROP TABLE IF EXISTS lessons_fts;

DROP TABLE IF EXISTS settings;
DROP TABLE IF EXISTS learning_records;
DROP TABLE IF EXISTS lessons;
DROP TABLE IF EXISTS workspaces;

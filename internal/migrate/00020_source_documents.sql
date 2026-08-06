-- +goose Up

-- Source documents: raw files the user hands to the agent, extracted into a
-- managed record. The raw file is kept in the workspace's sources/ folder
-- (immutable provenance, re-extractable); the sanitized text is indexed in
-- sources_fts so the AGENT can retrieve passages. THIS TABLE IS NOT WIRED
-- INTO USER SEARCH — getSourceDocs-family methods and QuerySources are the
-- only readers; WorkspaceStore.Search / Store.Search never touch it.

CREATE TABLE source_documents (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_id     INTEGER NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    title            TEXT NOT NULL,
    slug             TEXT NOT NULL,
    filename         TEXT NOT NULL,
    path             TEXT NOT NULL,
    source_ext       TEXT NOT NULL DEFAULT '',
    format           TEXT NOT NULL DEFAULT '',
    method           TEXT NOT NULL DEFAULT '',
    sha256           TEXT NOT NULL DEFAULT '',
    pages            INTEGER NOT NULL DEFAULT 0,
    estimated_tokens INTEGER NOT NULL DEFAULT 0,
    chapters         TEXT NOT NULL DEFAULT '[]',
    text             TEXT NOT NULL DEFAULT '',
    created_at       TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at       TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_source_documents_workspace ON source_documents(workspace_id);

CREATE VIRTUAL TABLE sources_fts USING fts5(
    title, text,
    content=source_documents,
    content_rowid=id,
    tokenize = 'porter unicode61'
);

-- +goose StatementBegin
CREATE TRIGGER sources_ai AFTER INSERT ON source_documents BEGIN
    INSERT INTO sources_fts(rowid, title, text)
    VALUES (new.id, new.title, new.text);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER sources_ad AFTER DELETE ON source_documents BEGIN
    INSERT INTO sources_fts(sources_fts, rowid, title, text)
    VALUES('delete', old.id, old.title, old.text);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER sources_au AFTER UPDATE OF title, text ON source_documents BEGIN
    INSERT INTO sources_fts(sources_fts, rowid, title, text)
    VALUES('delete', old.id, old.title, old.text);
    INSERT INTO sources_fts(rowid, title, text)
    VALUES (new.id, new.title, new.text);
END;
-- +goose StatementEnd

INSERT INTO sources_fts(sources_fts) VALUES('rebuild');

-- +goose Down

DROP TRIGGER IF EXISTS sources_au;
DROP TRIGGER IF EXISTS sources_ad;
DROP TRIGGER IF EXISTS sources_ai;
DROP TABLE IF EXISTS sources_fts;
DROP TABLE IF EXISTS source_documents;
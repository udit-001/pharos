-- +goose Up

-- Global scratchpad: loose, unstructured capture surface for resources and
-- intentions ("I want to learn X later"). Deliberately NOT workspace-scoped —
-- no workspace_id anywhere — so it floats above all workspaces and is sealed
-- from them (fundamental to the design). The agent is the sole operator for
-- v1 (command surface: pharos scrap / pharos tag).

CREATE TABLE scraps (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    slug       TEXT NOT NULL UNIQUE,
    title      TEXT NOT NULL,
    body       TEXT NOT NULL DEFAULT '',
    status     TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE tags (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Many-to-many between scraps and tags. Deleting a scrap or a tag removes its
-- join rows (cascade) — the tag description is the semantic payload searched
-- along with the scrap body.
CREATE TABLE scrap_tags (
    scrap_id INTEGER NOT NULL REFERENCES scraps(id) ON DELETE CASCADE,
    tag_id   INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (scrap_id, tag_id)
);

CREATE INDEX idx_scrap_tags_tag ON scrap_tags(tag_id);

-- FTS5 index over scrap title + body, following the source_documents external-
-- content pattern (content <-> base table, triggered, porter unicode61).
-- NOTE: tag name/description is NOT indexed here — it can't be (tags live in a
-- separate table, not on the scraps row). Tag search is done by the store layer
-- as a LIKE-join against tags (see SearchScraps / ftsTermsLike in scraps.go),
-- so a tag term still surfaces its scraps, just without bm25 ranking.
CREATE VIRTUAL TABLE scraps_fts USING fts5(
    title, body,
    content=scraps,
    content_rowid=id,
    tokenize = 'porter unicode61'
);

-- +goose StatementBegin
CREATE TRIGGER scraps_ai AFTER INSERT ON scraps BEGIN
    INSERT INTO scraps_fts(rowid, title, body)
    VALUES (new.id, new.title, new.body);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER scraps_ad AFTER DELETE ON scraps BEGIN
    INSERT INTO scraps_fts(scraps_fts, rowid, title, body)
    VALUES('delete', old.id, old.title, old.body);
END;
-- +goose StatementEnd

-- Scoped to indexed columns so non-indexed UPDATEs (status, updated_at) skip
-- the FTS delete+insert (the LEARN-106 pattern).
-- +goose StatementBegin
CREATE TRIGGER scraps_au AFTER UPDATE OF title, body ON scraps BEGIN
    INSERT INTO scraps_fts(scraps_fts, rowid, title, body)
    VALUES('delete', old.id, old.title, old.body);
    INSERT INTO scraps_fts(rowid, title, body)
    VALUES (new.id, new.title, new.body);
END;
-- +goose StatementEnd

INSERT INTO scraps_fts(scraps_fts) VALUES('rebuild');

-- +goose Down

DROP TRIGGER IF EXISTS scraps_au;
DROP TRIGGER IF EXISTS scraps_ad;
DROP TRIGGER IF EXISTS scraps_ai;
DROP TABLE IF EXISTS scraps_fts;
DROP INDEX IF EXISTS idx_scrap_tags_tag;
DROP TABLE IF EXISTS scrap_tags;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS scraps;
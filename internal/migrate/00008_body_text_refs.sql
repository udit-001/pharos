-- +goose Up

ALTER TABLE references_t ADD COLUMN body_text TEXT NOT NULL DEFAULT '';

DROP TRIGGER IF EXISTS refs_au;
DROP TRIGGER IF EXISTS refs_ad;
DROP TRIGGER IF EXISTS refs_ai;
DROP TABLE IF EXISTS refs_fts;

CREATE VIRTUAL TABLE refs_fts USING fts5(
    title, summary, body_text,
    content=references_t,
    content_rowid=id
);

-- +goose StatementBegin
CREATE TRIGGER refs_ai AFTER INSERT ON references_t BEGIN
    INSERT INTO refs_fts(rowid, title, summary, body_text)
    VALUES (new.id, new.title, new.summary, new.body_text);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER refs_ad AFTER DELETE ON references_t BEGIN
    INSERT INTO refs_fts(refs_fts, rowid, title, summary, body_text)
    VALUES('delete', old.id, old.title, old.summary, old.body_text);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER refs_au AFTER UPDATE ON references_t BEGIN
    INSERT INTO refs_fts(refs_fts, rowid, title, summary, body_text)
    VALUES('delete', old.id, old.title, old.summary, old.body_text);
    INSERT INTO refs_fts(rowid, title, summary, body_text)
    VALUES (new.id, new.title, new.summary, new.body_text);
END;
-- +goose StatementEnd

INSERT INTO refs_fts(refs_fts) VALUES('rebuild');

-- +goose Down

DROP TRIGGER IF EXISTS refs_au;
DROP TRIGGER IF EXISTS refs_ad;
DROP TRIGGER IF EXISTS refs_ai;
DROP TABLE IF EXISTS refs_fts;

ALTER TABLE references_t DROP COLUMN body_text;

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

INSERT INTO refs_fts(refs_fts) VALUES('rebuild');

-- +goose Up

ALTER TABLE lessons ADD COLUMN body_text TEXT NOT NULL DEFAULT '';

DROP TRIGGER IF EXISTS lessons_au;
DROP TRIGGER IF EXISTS lessons_ad;
DROP TRIGGER IF EXISTS lessons_ai;
DROP TABLE IF EXISTS lessons_fts;

CREATE VIRTUAL TABLE lessons_fts USING fts5(
    title, summary, body_text,
    content=lessons,
    content_rowid=id
);

-- +goose StatementBegin
CREATE TRIGGER lessons_ai AFTER INSERT ON lessons BEGIN
    INSERT INTO lessons_fts(rowid, title, summary, body_text)
    VALUES (new.id, new.title, new.summary, new.body_text);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER lessons_ad AFTER DELETE ON lessons BEGIN
    INSERT INTO lessons_fts(lessons_fts, rowid, title, summary, body_text)
    VALUES('delete', old.id, old.title, old.summary, old.body_text);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER lessons_au AFTER UPDATE ON lessons BEGIN
    INSERT INTO lessons_fts(lessons_fts, rowid, title, summary, body_text)
    VALUES('delete', old.id, old.title, old.summary, old.body_text);
    INSERT INTO lessons_fts(rowid, title, summary, body_text)
    VALUES (new.id, new.title, new.summary, new.body_text);
END;
-- +goose StatementEnd

INSERT INTO lessons_fts(lessons_fts) VALUES('rebuild');

-- +goose Down

DROP TRIGGER IF EXISTS lessons_au;
DROP TRIGGER IF EXISTS lessons_ad;
DROP TRIGGER IF EXISTS lessons_ai;
DROP TABLE IF EXISTS lessons_fts;

ALTER TABLE lessons DROP COLUMN body_text;

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

INSERT INTO lessons_fts(lessons_fts) VALUES('rebuild');

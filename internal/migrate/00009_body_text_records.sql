-- +goose Up

ALTER TABLE learning_records ADD COLUMN body_text TEXT NOT NULL DEFAULT '';

DROP TRIGGER IF EXISTS records_au;
DROP TRIGGER IF EXISTS records_ad;
DROP TRIGGER IF EXISTS records_ai;
DROP TABLE IF EXISTS records_fts;

CREATE VIRTUAL TABLE records_fts USING fts5(
    title, summary, body_text,
    content=learning_records,
    content_rowid=id
);

-- +goose StatementBegin
CREATE TRIGGER records_ai AFTER INSERT ON learning_records BEGIN
    INSERT INTO records_fts(rowid, title, summary, body_text)
    VALUES (new.id, new.title, new.summary, new.body_text);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER records_ad AFTER DELETE ON learning_records BEGIN
    INSERT INTO records_fts(records_fts, rowid, title, summary, body_text)
    VALUES('delete', old.id, old.title, old.summary, old.body_text);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER records_au AFTER UPDATE ON learning_records BEGIN
    INSERT INTO records_fts(records_fts, rowid, title, summary, body_text)
    VALUES('delete', old.id, old.title, old.summary, old.body_text);
    INSERT INTO records_fts(rowid, title, summary, body_text)
    VALUES (new.id, new.title, new.summary, new.body_text);
END;
-- +goose StatementEnd

INSERT INTO records_fts(records_fts) VALUES('rebuild');

-- +goose Down

DROP TRIGGER IF EXISTS records_au;
DROP TRIGGER IF EXISTS records_ad;
DROP TRIGGER IF EXISTS records_ai;
DROP TABLE IF EXISTS records_fts;

ALTER TABLE learning_records DROP COLUMN body_text;

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

INSERT INTO records_fts(records_fts) VALUES('rebuild');

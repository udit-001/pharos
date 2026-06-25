-- +goose Up

DROP TRIGGER IF EXISTS lessons_au;
DROP TRIGGER IF EXISTS lessons_ad;
DROP TRIGGER IF EXISTS lessons_ai;
DROP TABLE IF EXISTS lessons_fts;

CREATE VIRTUAL TABLE lessons_fts USING fts5(
    title, summary, body_text,
    content=lessons,
    content_rowid=id,
    tokenize = 'porter unicode61'
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

DROP TRIGGER IF EXISTS refs_au;
DROP TRIGGER IF EXISTS refs_ad;
DROP TRIGGER IF EXISTS refs_ai;
DROP TABLE IF EXISTS refs_fts;

CREATE VIRTUAL TABLE refs_fts USING fts5(
    title, summary, body_text,
    content=references_t,
    content_rowid=id,
    tokenize = 'porter unicode61'
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

DROP TRIGGER IF EXISTS records_au;
DROP TRIGGER IF EXISTS records_ad;
DROP TRIGGER IF EXISTS records_ai;
DROP TABLE IF EXISTS records_fts;

CREATE VIRTUAL TABLE records_fts USING fts5(
    title, summary, body_text,
    content=learning_records,
    content_rowid=id,
    tokenize = 'porter unicode61'
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

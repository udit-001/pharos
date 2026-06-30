-- +goose Up
--
-- Scope the FTS5 _au (AFTER UPDATE) triggers to the indexed columns
-- (title, summary, body_text) so non-indexed UPDATEs skip the FTS
-- delete+insert. Per sqlite.org/lang_createtrigger.html, "UPDATE OF"
-- fires when a named column appears on the LHS of a SET term, so the
-- trigger never under-fires on a real indexed change (LEARN-106).
--
-- Safe because no code updates the content_rowid (id); indexed-column
-- UPDATEs (the Revise paths) still SET title/summary/body_text and fire.

DROP TRIGGER IF EXISTS lessons_au;
DROP TRIGGER IF EXISTS refs_au;
DROP TRIGGER IF EXISTS records_au;

-- +goose StatementBegin
CREATE TRIGGER lessons_au AFTER UPDATE OF title, summary, body_text ON lessons BEGIN
    INSERT INTO lessons_fts(lessons_fts, rowid, title, summary, body_text)
    VALUES('delete', old.id, old.title, old.summary, old.body_text);
    INSERT INTO lessons_fts(rowid, title, summary, body_text)
    VALUES (new.id, new.title, new.summary, new.body_text);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER refs_au AFTER UPDATE OF title, summary, body_text ON references_t BEGIN
    INSERT INTO refs_fts(refs_fts, rowid, title, summary, body_text)
    VALUES('delete', old.id, old.title, old.summary, old.body_text);
    INSERT INTO refs_fts(rowid, title, summary, body_text)
    VALUES (new.id, new.title, new.summary, new.body_text);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER records_au AFTER UPDATE OF title, summary, body_text ON learning_records BEGIN
    INSERT INTO records_fts(records_fts, rowid, title, summary, body_text)
    VALUES('delete', old.id, old.title, old.summary, old.body_text);
    INSERT INTO records_fts(rowid, title, summary, body_text)
    VALUES (new.id, new.title, new.summary, new.body_text);
END;
-- +goose StatementEnd

-- +goose Down
--
-- Restore the un-scoped AFTER UPDATE triggers (fire on every update).

DROP TRIGGER IF EXISTS lessons_au;
DROP TRIGGER IF EXISTS refs_au;
DROP TRIGGER IF EXISTS records_au;

-- +goose StatementBegin
CREATE TRIGGER lessons_au AFTER UPDATE ON lessons BEGIN
    INSERT INTO lessons_fts(lessons_fts, rowid, title, summary, body_text)
    VALUES('delete', old.id, old.title, old.summary, old.body_text);
    INSERT INTO lessons_fts(rowid, title, summary, body_text)
    VALUES (new.id, new.title, new.summary, new.body_text);
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

-- +goose StatementBegin
CREATE TRIGGER records_au AFTER UPDATE ON learning_records BEGIN
    INSERT INTO records_fts(records_fts, rowid, title, summary, body_text)
    VALUES('delete', old.id, old.title, old.summary, old.body_text);
    INSERT INTO records_fts(rowid, title, summary, body_text)
    VALUES (new.id, new.title, new.summary, new.body_text);
END;
-- +goose StatementEnd

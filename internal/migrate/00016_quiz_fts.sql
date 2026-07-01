-- +goose Up
--
-- Quizzes are DB-only (no file on disk), so the FTS index is maintained
-- entirely by triggers — no IndexQuizzes / file-extraction step. Mirrors
-- the lessons/refs/records FTS pattern (00012) with the 00015 column-scoped
-- _au trigger. Indexes title + description (no body_text).

CREATE VIRTUAL TABLE quizzes_fts USING fts5(
    title, description,
    content=quizzes,
    content_rowid=id,
    tokenize = 'porter unicode61'
);

-- +goose StatementBegin
CREATE TRIGGER quizzes_ai AFTER INSERT ON quizzes BEGIN
    INSERT INTO quizzes_fts(rowid, title, description)
    VALUES (new.id, new.title, new.description);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER quizzes_ad AFTER DELETE ON quizzes BEGIN
    INSERT INTO quizzes_fts(quizzes_fts, rowid, title, description)
    VALUES('delete', old.id, old.title, old.description);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER quizzes_au AFTER UPDATE OF title, description ON quizzes BEGIN
    INSERT INTO quizzes_fts(quizzes_fts, rowid, title, description)
    VALUES('delete', old.id, old.title, old.description);
    INSERT INTO quizzes_fts(rowid, title, description)
    VALUES (new.id, new.title, new.description);
END;
-- +goose StatementEnd

INSERT INTO quizzes_fts(quizzes_fts) VALUES('rebuild');

-- +goose Down

DROP TRIGGER IF EXISTS quizzes_au;
DROP TRIGGER IF EXISTS quizzes_ad;
DROP TRIGGER IF EXISTS quizzes_ai;
DROP TABLE IF EXISTS quizzes_fts;

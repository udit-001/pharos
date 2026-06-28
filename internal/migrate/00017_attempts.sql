-- +goose Up
CREATE TABLE attempts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    quiz_attempt_id INTEGER NOT NULL REFERENCES quiz_attempts(id) ON DELETE CASCADE,
    question_id INTEGER NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    correct INTEGER,
    response TEXT NOT NULL DEFAULT '',
    latency_ms INTEGER,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_attempts_quiz_attempt ON attempts(quiz_attempt_id);
CREATE INDEX idx_attempts_question ON attempts(question_id);

-- +goose Down
DROP TABLE attempts;

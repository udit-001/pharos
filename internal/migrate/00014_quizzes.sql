-- +goose Up

-- Questions: the item bank quizzes draw from. DB-only (no file on disk).
CREATE TABLE questions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_id INTEGER NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    slug TEXT NOT NULL,
    mode TEXT NOT NULL,
    config TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_questions_workspace ON questions(workspace_id);
CREATE UNIQUE INDEX idx_questions_ws_slug ON questions(workspace_id, slug);

-- Quizzes: ordered lists of question slugs. lesson_seq is a nullable soft
-- reference to the lesson whose skill this quiz practices (NULL = unlinked,
-- e.g. a general review quiz). Soft, not a FK: lessons can be renumbered or
-- deleted, and the link is resolved by (workspace_id, sequence_number).
CREATE TABLE quizzes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_id INTEGER NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    slug TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    items TEXT NOT NULL DEFAULT '[]',
    lesson_seq INTEGER,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_quizzes_workspace ON quizzes(workspace_id);
CREATE UNIQUE INDEX idx_quizzes_ws_slug ON quizzes(workspace_id, slug);
CREATE INDEX idx_quizzes_lesson ON quizzes(workspace_id, lesson_seq);

-- Quiz attempts: one run through a quiz. Status: in_progress -> completed | abandoned.
CREATE TABLE quiz_attempts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_id INTEGER NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    quiz_id INTEGER NOT NULL REFERENCES quizzes(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'in_progress',
    started_at TEXT NOT NULL DEFAULT (datetime('now')),
    completed_at TEXT
);

CREATE INDEX idx_quiz_attempts_workspace ON quiz_attempts(workspace_id);
CREATE INDEX idx_quiz_attempts_quiz ON quiz_attempts(quiz_id);

-- Attempts: one answered question within a quiz attempt. For choice mode the
-- server sets correct; for recall the client self-grades.
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
DROP TABLE quiz_attempts;
DROP TABLE quizzes;
DROP TABLE questions;

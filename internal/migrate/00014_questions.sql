-- +goose Up
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

-- +goose Down
DROP TABLE questions;

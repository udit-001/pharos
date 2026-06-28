-- +goose Up
CREATE TABLE quizzes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_id INTEGER NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    slug TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    items TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_quizzes_workspace ON quizzes(workspace_id);
CREATE UNIQUE INDEX idx_quizzes_ws_slug ON quizzes(workspace_id, slug);

-- +goose Down
DROP TABLE quizzes;

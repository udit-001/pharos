-- +goose Up
CREATE TABLE glossary_terms (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_id INTEGER NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    term TEXT NOT NULL,
    definition TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_glossary_terms_workspace ON glossary_terms(workspace_id);
CREATE UNIQUE INDEX idx_glossary_terms_ws_term ON glossary_terms(workspace_id, term);

-- +goose Down
DROP TABLE glossary_terms;

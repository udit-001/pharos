-- +goose Up

-- Highlights: a colored mark anchored to a text span in a document, with an
-- optional note. Target is polymorphic via (doc_type, doc_id) — 'lesson' or
-- 'ref' — so highlights inject into both lessons and refs. workspace_id
-- scopes every query. anchor_data holds the text-anchoring JSON {text,
-- prefix, suffix} that the client uses to re-locate the span on revisit.
CREATE TABLE highlights (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_id INTEGER NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    doc_type TEXT NOT NULL DEFAULT 'lesson',
    doc_id INTEGER NOT NULL,
    color TEXT NOT NULL DEFAULT '#EBCB8B',
    note_text TEXT NOT NULL DEFAULT '',
    anchor_data TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_highlights_workspace ON highlights(workspace_id);
CREATE INDEX idx_highlights_doc ON highlights(workspace_id, doc_type, doc_id);

-- +goose Down
DROP TABLE IF EXISTS highlights;

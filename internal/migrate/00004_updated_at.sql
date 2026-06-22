-- +goose Up
ALTER TABLE lessons ADD COLUMN updated_at TEXT NOT NULL DEFAULT '';
ALTER TABLE learning_records ADD COLUMN updated_at TEXT NOT NULL DEFAULT '';
ALTER TABLE references_t ADD COLUMN updated_at TEXT NOT NULL DEFAULT '';

-- Backfill updated_at from created_at for existing rows
UPDATE lessons SET updated_at = created_at WHERE updated_at = '';
UPDATE learning_records SET updated_at = created_at WHERE updated_at = '';
UPDATE references_t SET updated_at = created_at WHERE updated_at = '';

-- +goose Down
ALTER TABLE lessons DROP COLUMN updated_at;
ALTER TABLE learning_records DROP COLUMN updated_at;
ALTER TABLE references_t DROP COLUMN updated_at;

-- +goose Up
-- References are addressed by slug (descriptive name), not sequence number.
-- Add slug column, drop sequence_number (alpha — no data migration).
ALTER TABLE references_t ADD COLUMN slug TEXT NOT NULL DEFAULT '';

-- +goose Down
-- SQLite doesn't support DROP COLUMN before 3.35.0, but goose handles this.
ALTER TABLE references_t DROP COLUMN slug;

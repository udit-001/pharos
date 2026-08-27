-- +goose Up
-- Add slug column for stable lesson identity. Backfill, file rename, and
-- link rewrite happen in Go code (db.backfillSlugs) after this migration.
ALTER TABLE lessons ADD COLUMN slug TEXT NOT NULL DEFAULT '';
ALTER TABLE learning_records ADD COLUMN slug TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE lessons DROP COLUMN slug;
ALTER TABLE learning_records DROP COLUMN slug;

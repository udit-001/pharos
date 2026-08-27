-- +goose Up
-- Replace last_*_seq (position/ID-based) with last_*_slug (stable identity).
-- Backfill from lessons/learning_records/refs tables where possible.

ALTER TABLE workspaces ADD COLUMN last_lesson_slug TEXT DEFAULT NULL;
ALTER TABLE workspaces ADD COLUMN last_record_slug TEXT DEFAULT NULL;
ALTER TABLE workspaces ADD COLUMN last_ref_slug TEXT DEFAULT NULL;

-- Backfill lesson slug: join on sequence_number to get the lesson's slug.
UPDATE workspaces
SET last_lesson_slug = (
    SELECT l.slug FROM lessons l
    WHERE l.workspace_id = workspaces.id
      AND l.sequence_number = workspaces.last_lesson_seq
)
WHERE last_lesson_seq IS NOT NULL AND last_lesson_slug IS NULL;

-- Backfill record slug: join on sequence_number.
UPDATE workspaces
SET last_record_slug = (
    SELECT r.slug FROM learning_records r
    WHERE r.workspace_id = workspaces.id
      AND r.sequence_number = workspaces.last_record_seq
)
WHERE last_record_seq IS NOT NULL AND last_record_slug IS NULL;

-- Backfill ref slug: last_ref_seq stored row ID, not sequence number.
UPDATE workspaces
SET last_ref_slug = (
    SELECT r.slug FROM references_t r
    WHERE r.id = workspaces.last_ref_seq
)
WHERE last_ref_seq IS NOT NULL AND last_ref_slug IS NULL;

-- Drop old columns.
ALTER TABLE workspaces DROP COLUMN last_lesson_seq;
ALTER TABLE workspaces DROP COLUMN last_record_seq;
ALTER TABLE workspaces DROP COLUMN last_ref_seq;

-- +goose Down
ALTER TABLE workspaces ADD COLUMN last_lesson_seq INTEGER DEFAULT NULL;
ALTER TABLE workspaces ADD COLUMN last_record_seq INTEGER DEFAULT NULL;
ALTER TABLE workspaces ADD COLUMN last_ref_seq INTEGER DEFAULT NULL;

UPDATE workspaces
SET last_lesson_seq = (
    SELECT l.sequence_number FROM lessons l
    WHERE l.workspace_id = workspaces.id AND l.slug = workspaces.last_lesson_slug
)
WHERE last_lesson_slug IS NOT NULL;

UPDATE workspaces
SET last_record_seq = (
    SELECT r.sequence_number FROM learning_records r
    WHERE r.workspace_id = workspaces.id AND r.slug = workspaces.last_record_slug
)
WHERE last_record_slug IS NOT NULL;

UPDATE workspaces
SET last_ref_seq = (
    SELECT r.id FROM references_t r
    WHERE r.workspace_id = workspaces.id AND r.slug = workspaces.last_ref_slug
)
WHERE last_ref_slug IS NOT NULL;

ALTER TABLE workspaces DROP COLUMN last_lesson_slug;
ALTER TABLE workspaces DROP COLUMN last_record_slug;
ALTER TABLE workspaces DROP COLUMN last_ref_slug;

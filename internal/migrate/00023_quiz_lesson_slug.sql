-- +goose Up
-- Replace lesson_seq (position-based) with lesson_slug (identity-based).
-- Backfill lesson_slug from the lessons table for existing links.

-- Add the new column.
ALTER TABLE quizzes ADD COLUMN lesson_slug TEXT DEFAULT NULL;

-- Backfill: join on workspace_id + sequence_number to get the lesson's slug.
UPDATE quizzes
SET lesson_slug = (
    SELECT l.slug FROM lessons l
    WHERE l.workspace_id = quizzes.workspace_id
      AND l.sequence_number = quizzes.lesson_seq
)
WHERE lesson_seq IS NOT NULL AND lesson_slug IS NULL;

-- Drop the old column and its index.
DROP INDEX IF EXISTS idx_quizzes_lesson;
ALTER TABLE quizzes DROP COLUMN lesson_seq;

-- Add index for the new column.
CREATE INDEX idx_quizzes_lesson_slug ON quizzes(workspace_id, lesson_slug);

-- +goose Down
DROP INDEX IF EXISTS idx_quizzes_lesson_slug;
ALTER TABLE quizzes ADD COLUMN lesson_seq INTEGER DEFAULT NULL;

-- Backfill from lesson_slug.
UPDATE quizzes
SET lesson_seq = (
    SELECT l.sequence_number FROM lessons l
    WHERE l.workspace_id = quizzes.workspace_id
      AND l.slug = quizzes.lesson_slug
)
WHERE lesson_slug IS NOT NULL AND lesson_seq IS NULL;

ALTER TABLE quizzes DROP COLUMN lesson_slug;
CREATE INDEX idx_quizzes_lesson ON quizzes(workspace_id, lesson_seq);

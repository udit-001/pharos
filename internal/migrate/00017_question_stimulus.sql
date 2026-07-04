-- +goose Up

-- Questions may now carry an optional stimulus: a standalone HTML file
-- (chart/diagram/passage) rendered in an iframe during the quiz attempt and
-- review. filename/path are empty when a question has no stimulus (the common
-- case). No body_text / FTS table — question search is intentionally out of
-- scope; the stimulus is a presentational artifact, not indexed content.
ALTER TABLE questions ADD COLUMN filename TEXT NOT NULL DEFAULT '';
ALTER TABLE questions ADD COLUMN path TEXT NOT NULL DEFAULT '';

-- +goose Down

ALTER TABLE questions DROP COLUMN filename;
ALTER TABLE questions DROP COLUMN path;

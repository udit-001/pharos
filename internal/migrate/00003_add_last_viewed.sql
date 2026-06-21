-- +goose Up
ALTER TABLE workspaces ADD COLUMN last_lesson_seq INTEGER DEFAULT NULL;
ALTER TABLE workspaces ADD COLUMN last_record_seq INTEGER DEFAULT NULL;
ALTER TABLE workspaces ADD COLUMN last_ref_seq INTEGER DEFAULT NULL;

-- +goose Down
ALTER TABLE workspaces DROP COLUMN last_lesson_seq;
ALTER TABLE workspaces DROP COLUMN last_record_seq;
ALTER TABLE workspaces DROP COLUMN last_ref_seq;

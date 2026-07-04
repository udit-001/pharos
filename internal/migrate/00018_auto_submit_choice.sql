-- +goose Up
ALTER TABLE settings ADD COLUMN auto_submit_choice INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE settings DROP COLUMN auto_submit_choice;

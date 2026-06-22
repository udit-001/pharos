-- +goose Up
ALTER TABLE settings ADD COLUMN last_active_workspace TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE settings DROP COLUMN last_active_workspace;

-- +goose Up
-- Normalize space-format timestamps (from datetime('now')) to RFC3339Nano format
-- so ORDER BY last_studied DESC works correctly across all rows.
UPDATE workspaces
SET last_studied = REPLACE(last_studied, ' ', 'T') || 'Z'
WHERE last_studied LIKE '% %';

-- +goose Down
-- No-op: the old format is broken, and we can't reconstruct original values
-- without knowing which were already correct vs which were migrated.

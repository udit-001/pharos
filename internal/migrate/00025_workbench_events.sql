-- +goose Up
-- Workbench event journal: stores sql-workbench browser events for
-- workspace-scoped query and replay. Event id is the primary key so
-- re-POSTed events are idempotent no-ops (decision 2, LEARN-206).
--
-- Columns:
--   id          TEXT PRIMARY KEY  — bench event id (idempotency key)
--   workspace_id INTEGER          — FK to workspaces
--   namespace   TEXT               — event namespace (bench-scoped)
--   type        TEXT               — event type (query|error|info|dataset)
--   ts          TEXT               — ISO-8601 timestamp
--   payload     TEXT               — JSON blob (full event, extra fields kept verbatim)

CREATE TABLE IF NOT EXISTS workbench_events (
    id           TEXT PRIMARY KEY,
    workspace_id INTEGER NOT NULL,
    namespace    TEXT    NOT NULL DEFAULT '',
    type         TEXT    NOT NULL,
    ts           TEXT    NOT NULL,
    payload      TEXT    NOT NULL DEFAULT '{}'
);

-- Query index: workspace + namespace + timestamp for log ordering.
CREATE INDEX IF NOT EXISTS idx_workbench_events_ws_ns_ts
    ON workbench_events (workspace_id, namespace, ts);

-- Type index: filter by event type (query, error, info, dataset).
CREATE INDEX IF NOT EXISTS idx_workbench_events_ws_type
    ON workbench_events (workspace_id, type);

-- +goose Down
DROP INDEX IF EXISTS idx_workbench_events_ws_type;
DROP INDEX IF EXISTS idx_workbench_events_ws_ns_ts;
DROP TABLE IF EXISTS workbench_events;

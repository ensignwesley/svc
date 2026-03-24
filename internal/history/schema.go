package history

// DDL for the history database.
// Two tables: checks (immutable facts) and incidents (derived meaning).
//
// Design invariants:
//   - checks rows are never updated or deleted (append-only)
//   - incidents rows are created on FIRST failure, updated on recovery
//   - recovered_at IS NULL means the incident is still open (service still down)
//   - incidents can always be rebuilt from checks if the table is lost

const schema = `
CREATE TABLE IF NOT EXISTS checks (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    service_id  TEXT    NOT NULL,
    host        TEXT    NOT NULL DEFAULT 'localhost',
    checked_at  INTEGER NOT NULL,   -- unix timestamp (seconds)
    status      TEXT    NOT NULL,   -- 'up' | 'down' | 'timeout' | 'error'
    latency_ms  INTEGER,            -- NULL when down/timeout/error
    error       TEXT                -- NULL when up
);

CREATE TABLE IF NOT EXISTS incidents (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    service_id    TEXT    NOT NULL,
    host          TEXT    NOT NULL DEFAULT 'localhost',
    started_at    INTEGER NOT NULL,  -- unix timestamp of first failing check
    recovered_at  INTEGER,           -- NULL = still open (service still down)
    duration_sec  INTEGER,           -- NULL until recovered
    check_count   INTEGER NOT NULL DEFAULT 1,  -- failing checks in this incident
    first_error   TEXT,              -- error from the first failing check
    last_error    TEXT               -- error from the most recent failing check
);

CREATE INDEX IF NOT EXISTS idx_checks_service ON checks(service_id, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_checks_time    ON checks(checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_incidents_svc  ON incidents(service_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_incidents_open ON incidents(service_id) WHERE recovered_at IS NULL;
`

-- sqlc analysis only. golang-migrate ignores this file.
-- At runtime the import creates a temp view with this name over the attached database.
CREATE TABLE IF NOT EXISTS history_src (
    id INTEGER PRIMARY KEY,
    command TEXT NOT NULL,
    cwd TEXT NOT NULL,
    timestamp INTEGER NOT NULL,
    exit_code INTEGER NOT NULL,
    duration_ms INTEGER NOT NULL
);

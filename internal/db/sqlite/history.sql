-- name: RecordHistory :exec
INSERT INTO history (command, cwd, timestamp, exit_code, duration_ms)
VALUES (?, ?, ?, ?, ?);

-- name: CountHistorySrc :one
SELECT COUNT(*) FROM history_src;

-- name: CopyAttachedHistory :exec
INSERT INTO history (command, cwd, timestamp, exit_code, duration_ms)
SELECT command, cwd, timestamp, exit_code, duration_ms FROM history_src;

-- name: GetHistory :many
SELECT * FROM history
ORDER BY timestamp DESC
LIMIT ?;

-- name: SearchHistory :many
SELECT * FROM history
WHERE command LIKE ?
ORDER BY timestamp DESC
LIMIT ?;

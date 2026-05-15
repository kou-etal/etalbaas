-- name: ListEventHistoryByFunctionID :many
SELECT * FROM event_history
WHERE function_id = sqlc.arg('function_id')
  AND project_id = sqlc.arg('project_id')
  AND (sqlc.narg('status_filter')::text IS NULL OR status = sqlc.narg('status_filter'))
  AND (sqlc.narg('since')::timestamptz IS NULL OR created_at >= sqlc.narg('since'))
  AND (sqlc.narg('until')::timestamptz IS NULL OR created_at <= sqlc.narg('until'))
  AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR created_at < sqlc.narg('cursor_created_at')
    OR (created_at = sqlc.narg('cursor_created_at') AND id < sqlc.narg('cursor_id'))
  )
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('page_size');

-- name: GetProjectByIDAndTenantID :one
SELECT * FROM projects
WHERE id = $1 AND tenant_id = $2 AND status != 'deleted';

-- name: GetEventByID :one
SELECT * FROM event_history
WHERE id = $1 AND project_id = $2;

-- name: GetTenantByID :one
SELECT id, email, display_name, avatar_url, plan, status, created_at, updated_at
FROM tenants
WHERE id = $1 AND status != 'deleted';

-- name: UpdateTenantProfile :one
UPDATE tenants
SET display_name = COALESCE(sqlc.narg('display_name'), display_name),
    avatar_url = COALESCE(sqlc.narg('avatar_url'), avatar_url),
    updated_at = now()
WHERE id = $1 AND status != 'deleted'
RETURNING id, email, display_name, avatar_url, plan, status, created_at, updated_at;

-- name: ListTenants :many
SELECT id, email, display_name, avatar_url, plan, status, created_at, updated_at
FROM tenants
WHERE status != 'deleted'
  AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR created_at < sqlc.narg('cursor_created_at')
    OR (created_at = sqlc.narg('cursor_created_at') AND id < sqlc.narg('cursor_id'))
  )
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('page_size');

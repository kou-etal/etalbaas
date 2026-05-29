-- name: CreateProject :one
INSERT INTO projects (
    id, tenant_id, display_name, description,
    postgres_enabled, postgres_extensions, redis_enabled, postgrest_enabled
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: GetProjectByIDAndTenantID :one
SELECT * FROM projects
WHERE id = $1 AND tenant_id = $2 AND status != 'deleted';

-- name: ListProjectsByTenantID :many
SELECT * FROM projects
WHERE tenant_id = sqlc.arg('tenant_id')
  AND status != 'deleted'
  AND (
    sqlc.narg('cursor_created_at')::timestamptz IS NULL
    OR created_at < sqlc.narg('cursor_created_at')::timestamptz
    OR (created_at = sqlc.narg('cursor_created_at')::timestamptz AND id < sqlc.narg('cursor_id'))
  )
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('page_size');

-- name: UpdateProjectStatus :one
UPDATE projects
SET status = $3
WHERE id = $1 AND tenant_id = $2 AND status != 'deleted'
RETURNING *;

-- name: GetProjectStatus :one
SELECT status FROM projects
WHERE id = $1 AND tenant_id = $2 AND status != 'deleted';

-- name: CreateApiKey :one
INSERT INTO api_keys (
    id, project_id, name, key_hash, key_prefix, role, expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: ListApiKeysByProjectID :many
SELECT * FROM api_keys
WHERE project_id = sqlc.arg('project_id')
  AND (
    sqlc.narg('cursor_created_at')::timestamptz IS NULL
    OR created_at < sqlc.narg('cursor_created_at')::timestamptz
    OR (created_at = sqlc.narg('cursor_created_at')::timestamptz AND id < sqlc.narg('cursor_id'))
  )
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('page_size');

-- name: RevokeApiKey :one
UPDATE api_keys
SET revoked_at = now()::timestamptz
WHERE id = $1 AND project_id = $2 AND revoked_at IS NULL
RETURNING *;

-- name: CreateSecretMetadata :one
INSERT INTO secrets_metadata (
    id, project_id, name, description
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: GetSecretMetadataByID :one
SELECT * FROM secrets_metadata
WHERE id = $1 AND project_id = $2;

-- name: ListSecretsByProjectID :many
SELECT * FROM secrets_metadata
WHERE project_id = sqlc.arg('project_id')
  AND (
    sqlc.narg('cursor_created_at')::timestamptz IS NULL
    OR created_at < sqlc.narg('cursor_created_at')::timestamptz
    OR (created_at = sqlc.narg('cursor_created_at')::timestamptz AND id < sqlc.narg('cursor_id'))
  )
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('page_size');

-- name: UpdateSecretMetadataUpdatedAt :one
UPDATE secrets_metadata
SET updated_at = now()::timestamptz
WHERE id = $1 AND project_id = $2
RETURNING *;

-- name: DeleteSecretMetadata :one
DELETE FROM secrets_metadata
WHERE id = $1 AND project_id = $2
RETURNING *;

-- name: CountActiveProjectsByTenantID :one
SELECT COUNT(*) FROM projects
WHERE tenant_id = $1 AND status != 'deleted';

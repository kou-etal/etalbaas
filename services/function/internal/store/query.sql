-- name: CreateFunction :one
INSERT INTO functions (
    id, project_id, name, display_name, kind, mode,
    source_type, source_config, source_storage_path,
    runtime_preset, runtime_requirements, runtime_dockerfile,
    timeout_sec, gpu_config, triggers, env_vars
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9,
    $10, $11, $12,
    $13, $14, $15, $16
)
RETURNING *;

-- name: GetFunctionByIDAndProjectID :one
SELECT * FROM functions
WHERE id = $1 AND project_id = $2 AND status != 'deleted';

-- name: ListFunctionsByProjectID :many
SELECT * FROM functions
WHERE project_id = sqlc.arg('project_id')
  AND status != 'deleted'
  AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR created_at < sqlc.narg('cursor_created_at')
    OR (created_at = sqlc.narg('cursor_created_at') AND id < sqlc.narg('cursor_id'))
  )
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('page_size');

-- name: UpdateFunction :one
UPDATE functions
SET display_name = $3,
    mode = $4,
    source_type = $5,
    source_config = $6,
    source_storage_path = $7,
    runtime_preset = $8,
    runtime_requirements = $9,
    runtime_dockerfile = $10,
    timeout_sec = $11,
    gpu_config = $12,
    triggers = $13,
    env_vars = $14
WHERE id = $1 AND project_id = $2 AND status != 'deleted'
RETURNING *;

-- name: DeleteFunction :one
UPDATE functions
SET status = 'deleted'
WHERE id = $1 AND project_id = $2 AND status != 'deleted'
RETURNING *;

-- name: GetProjectByIDAndTenantID :one
SELECT * FROM projects
WHERE id = $1 AND tenant_id = $2 AND status != 'deleted';

-- name: UpdateFunctionBuildStatus :one
UPDATE functions
SET status = $3,
    build_image_ref = $4,
    build_image_digest = $5,
    build_duration_sec = $6,
    last_built_at = $7,
    updated_at = now()
WHERE id = $1 AND project_id = $2 AND status != 'deleted'
RETURNING *;

-- name: CountActiveFunctionsByProjectID :one
SELECT COUNT(*) FROM functions
WHERE project_id = $1 AND status != 'deleted';

-- name: ListInvocationsByFunctionID :many
SELECT * FROM invocations
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

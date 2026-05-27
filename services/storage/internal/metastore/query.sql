-- name: GetProjectByIDAndTenantID :one
SELECT id, tenant_id, status
FROM projects
WHERE id = $1 AND tenant_id = $2 AND status != 'deleted';

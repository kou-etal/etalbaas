-- name: CreateBucket :one
INSERT INTO storage.buckets (id, name, project_id, access_level, file_size_limit, allowed_mime_types)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetBucketByID :one
SELECT * FROM storage.buckets WHERE id = $1 AND project_id = $2;

-- name: GetBucketByName :one
SELECT * FROM storage.buckets WHERE name = $1 AND project_id = $2;

-- name: ListBuckets :many
SELECT * FROM storage.buckets
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateBucket :one
UPDATE storage.buckets
SET access_level = COALESCE(sqlc.narg('access_level'), access_level),
    file_size_limit = COALESCE(sqlc.narg('file_size_limit'), file_size_limit),
    allowed_mime_types = COALESCE(sqlc.narg('allowed_mime_types'), allowed_mime_types),
    updated_at = now()::timestamptz
WHERE id = $1 AND project_id = $2
RETURNING *;

-- name: DeleteBucket :one
DELETE FROM storage.buckets WHERE id = $1 AND project_id = $2 RETURNING *;

-- name: CountObjectsByBucketID :one
SELECT count(*) FROM storage.objects WHERE bucket_id = $1;

-- name: CreateObject :one
INSERT INTO storage.objects (bucket_id, name, owner, size, mime_type, etag, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpsertObject :one
INSERT INTO storage.objects (bucket_id, name, owner, size, mime_type, etag, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (bucket_id, name) DO UPDATE
SET size = EXCLUDED.size, mime_type = EXCLUDED.mime_type, etag = EXCLUDED.etag,
    metadata = EXCLUDED.metadata, owner = EXCLUDED.owner, updated_at = now()::timestamptz
RETURNING *;

-- name: GetObjectByBucketAndName :one
SELECT * FROM storage.objects WHERE bucket_id = $1 AND name = $2;

-- name: ListObjectsByBucket :many
SELECT * FROM storage.objects
WHERE bucket_id = $1
ORDER BY name
LIMIT $2 OFFSET $3;

-- name: DeleteObject :exec
DELETE FROM storage.objects WHERE bucket_id = $1 AND name = $2;

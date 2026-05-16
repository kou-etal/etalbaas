-- Schema for sqlc codegen only. The actual schema is created by the Operator
-- in each tenant DB via CNPG postInitSQL (see cloudnativepg.go).

CREATE SCHEMA IF NOT EXISTS storage;

CREATE TABLE storage.buckets (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    project_id TEXT NOT NULL,
    access_level TEXT NOT NULL DEFAULT 'protected',
    file_size_limit BIGINT,
    allowed_mime_types TEXT[],
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(project_id, name)
);

CREATE TABLE storage.objects (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    bucket_id TEXT NOT NULL REFERENCES storage.buckets(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    owner UUID,
    size BIGINT,
    mime_type TEXT,
    etag TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(bucket_id, name)
);

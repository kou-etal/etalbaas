BEGIN;


CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;


CREATE TABLE tenants (
    id            uuid        PRIMARY KEY,
    email         text        NOT NULL UNIQUE,
    display_name  text        NOT NULL,
    avatar_url    text,
    plan          text        NOT NULL DEFAULT 'free',
    status        text        NOT NULL DEFAULT 'active',
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT tenants_plan_check   CHECK (plan IN ('free', 'pro', 'enterprise')),
    CONSTRAINT tenants_status_check CHECK (status IN ('active', 'suspended', 'deleted'))
);

CREATE TRIGGER tenants_updated_at
    BEFORE UPDATE ON tenants
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();


CREATE TABLE projects (
    id                  text        PRIMARY KEY,
    tenant_id           uuid        NOT NULL REFERENCES tenants(id),
    display_name        text        NOT NULL,
    description         text        DEFAULT '',
    status              text        NOT NULL DEFAULT 'pending',
    postgres_enabled    boolean     NOT NULL DEFAULT true,
    postgres_extensions text[]      DEFAULT '{pgvector,pgcrypto}',
    redis_enabled       boolean     NOT NULL DEFAULT false,
    postgrest_enabled   boolean     NOT NULL DEFAULT false,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT projects_status_check     CHECK (status IN ('pending', 'provisioning', 'ready', 'paused', 'failed', 'deleted')),
    CONSTRAINT projects_postgrest_check  CHECK (NOT (postgrest_enabled AND NOT postgres_enabled))
);

CREATE INDEX idx_projects_tenant_id ON projects(tenant_id);
CREATE INDEX idx_projects_status    ON projects(status) WHERE status != 'deleted';

CREATE TRIGGER projects_updated_at
    BEFORE UPDATE ON projects
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();


CREATE TABLE api_keys (
    id          uuid        PRIMARY KEY,
    project_id  text        NOT NULL REFERENCES projects(id),
    name        text        NOT NULL,
    key_hash    text        NOT NULL,
    key_prefix  text        NOT NULL,
    role        text        NOT NULL,
    expires_at  timestamptz,
    revoked_at  timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT api_keys_role_check CHECK (role IN ('anon', 'service_role')),
    CONSTRAINT api_keys_project_name_unique UNIQUE (project_id, name)
);

CREATE INDEX idx_api_keys_key_hash   ON api_keys(key_hash) WHERE revoked_at IS NULL;
CREATE INDEX idx_api_keys_project_id ON api_keys(project_id);


CREATE TABLE functions (
    id                    uuid        PRIMARY KEY,
    project_id            text        NOT NULL REFERENCES projects(id),
    name                  text        NOT NULL,
    display_name          text        NOT NULL,
    kind                  text        NOT NULL,
    mode                  text        NOT NULL DEFAULT 'sync',
    source_type           text        NOT NULL DEFAULT 'inline',
    source_config         jsonb       DEFAULT '{}',
    source_storage_path   text,
    runtime_preset        text,
    runtime_requirements  text[]      DEFAULT '{}',
    runtime_dockerfile    text,
    timeout_sec           integer     NOT NULL DEFAULT 30,
    gpu_config            jsonb,
    triggers              jsonb       NOT NULL DEFAULT '[]',
    env_vars              jsonb       NOT NULL DEFAULT '{}',
    status                text        NOT NULL DEFAULT 'pending',
    build_image_ref       text,
    build_image_digest    text,
    build_duration_sec    integer,
    last_built_at         timestamptz,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT functions_kind_check        CHECK (kind IN ('heavy-job', 'heavy-deployment', 'light-deployment')),
    CONSTRAINT functions_mode_check        CHECK (mode IN ('sync', 'async', 'stream')),
    CONSTRAINT functions_source_type_check CHECK (source_type IN ('git', 'zip', 'inline')),
    CONSTRAINT functions_status_check      CHECK (status IN ('pending', 'building', 'ready', 'failed', 'deleted')),
    CONSTRAINT functions_runtime_check     CHECK (
        (runtime_preset IS NULL AND runtime_dockerfile IS NOT NULL)
        OR (runtime_preset IS NOT NULL AND runtime_dockerfile IS NULL)
    ),
    CONSTRAINT functions_project_name_unique UNIQUE (project_id, name)
);

CREATE INDEX idx_functions_project_id ON functions(project_id);

CREATE TRIGGER functions_updated_at
    BEFORE UPDATE ON functions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();


CREATE TABLE invocations (
    id                uuid        PRIMARY KEY,
    function_id       uuid        NOT NULL REFERENCES functions(id),
    project_id        text        NOT NULL,
    trigger_type      text        NOT NULL,
    mode              text        NOT NULL,
    status            text        NOT NULL DEFAULT 'pending',
    error_message     text,
    duration_ms       integer,
    gpu_provider      text,
    gpu_type          text,
    gpu_duration_ms   integer,
    cold_start_ms     integer,
    memory_peak_bytes bigint,
    cpu_millis        bigint,
    retry_count       integer     NOT NULL DEFAULT 0,
    trace_id          text,
    started_at        timestamptz,
    completed_at      timestamptz,
    created_at        timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT invocations_trigger_type_check CHECK (trigger_type IN ('http', 'database_change', 'object_storage')),
    CONSTRAINT invocations_mode_check         CHECK (mode IN ('sync', 'async', 'stream')),
    CONSTRAINT invocations_status_check       CHECK (status IN ('pending', 'running', 'succeeded', 'failed', 'timeout', 'cancelled'))
);

CREATE INDEX idx_invocations_project_created  ON invocations(project_id, created_at DESC);
CREATE INDEX idx_invocations_function_created ON invocations(function_id, created_at DESC);


CREATE TABLE event_history (
    id             uuid        PRIMARY KEY,
    project_id     text        NOT NULL,
    function_id    uuid        NOT NULL REFERENCES functions(id),
    invocation_id  uuid        REFERENCES invocations(id),
    trigger_type   text        NOT NULL,
    trigger_data   jsonb       NOT NULL DEFAULT '{}',
    status         text        NOT NULL DEFAULT 'received',
    attempt_count  integer     NOT NULL DEFAULT 1,
    last_error     text,
    trace_id       text,
    created_at     timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT event_history_trigger_type_check CHECK (trigger_type IN ('database_change', 'object_storage')),
    CONSTRAINT event_history_status_check       CHECK (status IN ('received', 'delivered', 'retrying', 'failed'))
);

CREATE INDEX idx_event_history_project_created ON event_history(project_id, created_at DESC);
CREATE INDEX idx_event_history_invocation      ON event_history(invocation_id) WHERE invocation_id IS NOT NULL;


CREATE TABLE usage_daily (
    id                   uuid        PRIMARY KEY,
    project_id           text        NOT NULL REFERENCES projects(id),
    date                 date        NOT NULL,
    cpu_seconds          bigint      NOT NULL DEFAULT 0,
    memory_mb_seconds    bigint      NOT NULL DEFAULT 0,
    storage_bytes        bigint      NOT NULL DEFAULT 0,
    function_invocations bigint      NOT NULL DEFAULT 0,
    egress_bytes         bigint      NOT NULL DEFAULT 0,
    db_size_bytes        bigint      NOT NULL DEFAULT 0,
    created_at           timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT usage_daily_project_date_unique UNIQUE (project_id, date)
);


CREATE TABLE platform_events (
    id          uuid        PRIMARY KEY,
    tenant_id   uuid        NOT NULL,
    project_id  text        NOT NULL,
    event_type  text        NOT NULL,
    payload     jsonb       NOT NULL DEFAULT '{}',
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_platform_events_tenant_created  ON platform_events(tenant_id, created_at DESC);
CREATE INDEX idx_platform_events_project_created ON platform_events(project_id, created_at DESC);


CREATE TABLE secrets_metadata (
    id          uuid        PRIMARY KEY,
    project_id  text        NOT NULL REFERENCES projects(id),
    name        text        NOT NULL,
    description text        DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT secrets_metadata_project_name_unique UNIQUE (project_id, name)
);

CREATE TRIGGER secrets_metadata_updated_at
    BEFORE UPDATE ON secrets_metadata
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

COMMIT;

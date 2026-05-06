# Meta Database Design

メタDB（platform-system 用 PostgreSQL）のスキーマ設計。
テナントプロジェクト DB（project-{id} NS 内の CloudNativePG）とは別。

## 設計方針

- **PK**: UUID v7（アプリ側生成、時系列ソート可能）
- **project_id**: ランダム8文字 text（サブドメインと同一）
- **tenant_id**: GoTrue user_id（UUID）
- **Soft delete**: status カラムで管理（物理削除しない）
- **Enum**: text + CHECK 制約（PG enum は ALTER TYPE が面倒）
- **タイムスタンプ**: 全テーブル created_at / updated_at（trigger で自動更新）
- **JSONB**: 構造が可変なフィールドに限定使用（trigger config, gpu config 等）
- **マイグレーション**: golang-migrate, `deploy/migrations/meta/` に配置

## ER関係

```
tenants 1──N projects
              ├── 1──N functions
              │        ├── 1──N invocations
              │        └── 1──N event_history
              ├── 1──N api_keys
              ├── 1──N secrets_metadata
              ├── 1──N usage_daily
              └── 1──N platform_events
```

## スキーマ定義

```mermaid
erDiagram
    tenants ||--o{ projects : ""
    projects ||--o{ functions : ""
    projects ||--o{ api_keys : ""
    projects ||--o{ secrets_metadata : ""
    projects ||--o{ usage_daily : ""
    projects ||--o{ platform_events : ""
    functions ||--o{ invocations : ""
    functions ||--o{ event_history : ""
    invocations |o--o| event_history : ""

    tenants {
        uuid id PK "= GoTrue user_id"
        text email UK "NOT NULL"
        text display_name "NOT NULL"
        text avatar_url
        text plan "NOT NULL DEFAULT 'free'"
        text status "NOT NULL DEFAULT 'active'"
        timestamptz created_at "NOT NULL DEFAULT now()"
        timestamptz updated_at "NOT NULL DEFAULT now()"
    }

    projects {
        text id PK "8文字ランダム = サブドメイン"
        uuid tenant_id FK "NOT NULL"
        text display_name "NOT NULL"
        text description "DEFAULT ''"
        text status "NOT NULL DEFAULT 'pending'"
        boolean postgres_enabled "NOT NULL DEFAULT true"
        text_arr postgres_extensions "DEFAULT '{pgvector,pgcrypto}'"
        boolean redis_enabled "NOT NULL DEFAULT false"
        boolean postgrest_enabled "NOT NULL DEFAULT false"
        timestamptz created_at "NOT NULL DEFAULT now()"
        timestamptz updated_at "NOT NULL DEFAULT now()"
    }

    api_keys {
        uuid id PK
        text project_id FK "NOT NULL"
        text name "NOT NULL, UQ(project_id,name)"
        text key_hash "NOT NULL, SHA-256"
        text key_prefix "NOT NULL, 先頭8文字"
        text role "NOT NULL, anon/service_role"
        timestamptz expires_at "NULL = 無期限"
        timestamptz revoked_at "NULL = 有効"
        timestamptz created_at "NOT NULL DEFAULT now()"
    }

    functions {
        uuid id PK
        text project_id FK "NOT NULL"
        text name "NOT NULL, UQ(project_id,name)"
        text display_name "NOT NULL"
        text kind "NOT NULL"
        text mode "NOT NULL DEFAULT 'sync'"
        text source_type "NOT NULL DEFAULT 'inline'"
        jsonb source_config "DEFAULT '{}'"
        text source_storage_path
        text runtime_preset "NULL = custom"
        text_arr runtime_requirements "DEFAULT '{}'"
        text runtime_dockerfile "custom時に必須"
        integer timeout_sec "NOT NULL DEFAULT 30"
        jsonb gpu_config "NULL = GPU不使用"
        jsonb triggers "NOT NULL DEFAULT '[]'"
        jsonb env_vars "NOT NULL DEFAULT '{}'"
        text status "NOT NULL DEFAULT 'pending'"
        text build_image_ref
        text build_image_digest
        integer build_duration_sec
        timestamptz last_built_at
        timestamptz created_at "NOT NULL DEFAULT now()"
        timestamptz updated_at "NOT NULL DEFAULT now()"
    }

    invocations {
        uuid id PK
        uuid function_id FK "NOT NULL"
        text project_id "NOT NULL, 非正規化"
        text trigger_type "NOT NULL"
        text mode "NOT NULL"
        text status "NOT NULL DEFAULT 'pending'"
        text error_message
        integer duration_ms
        text gpu_provider
        timestamptz started_at
        timestamptz completed_at
        timestamptz created_at "NOT NULL DEFAULT now()"
    }

    event_history {
        uuid id PK
        text project_id "NOT NULL"
        uuid function_id FK "NOT NULL"
        uuid invocation_id FK "NULL = 未実行"
        text trigger_type "NOT NULL"
        text trigger_source "NOT NULL"
        text status "NOT NULL DEFAULT 'delivered'"
        integer attempt_count "NOT NULL DEFAULT 1"
        text last_error
        timestamptz created_at "NOT NULL DEFAULT now()"
    }

    usage_daily {
        uuid id PK
        text project_id FK "NOT NULL"
        date date "NOT NULL, UQ(project_id,date)"
        bigint cpu_seconds "NOT NULL DEFAULT 0"
        bigint memory_mb_seconds "NOT NULL DEFAULT 0"
        bigint storage_bytes "NOT NULL DEFAULT 0"
        bigint function_invocations "NOT NULL DEFAULT 0"
        bigint egress_bytes "NOT NULL DEFAULT 0"
        bigint db_size_bytes "NOT NULL DEFAULT 0"
        timestamptz created_at "NOT NULL DEFAULT now()"
    }

    platform_events {
        uuid id PK
        uuid tenant_id "NOT NULL, 非正規化"
        text project_id "NOT NULL"
        text event_type "NOT NULL"
        jsonb payload "NOT NULL DEFAULT '{}'"
        timestamptz created_at "NOT NULL DEFAULT now()"
    }

    secrets_metadata {
        uuid id PK
        text project_id FK "NOT NULL"
        text name "NOT NULL, UQ(project_id,name)"
        text description "DEFAULT ''"
        timestamptz created_at "NOT NULL DEFAULT now()"
        timestamptz updated_at "NOT NULL DEFAULT now()"
    }
```

## CHECK 制約

| テーブル | カラム | 許可値 |
|---------|--------|--------|
| tenants | plan | `free`, `pro`, `enterprise` |
| tenants | status | `active`, `suspended`, `deleted` |
| projects | status | `pending`, `provisioning`, `ready`, `paused`, `failed`, `deleted` |
| projects | postgrest_enabled | `NOT (postgrest_enabled AND NOT postgres_enabled)` |
| api_keys | role | `anon`, `service_role` |
| functions | kind | `heavy-job`, `heavy-deployment`, `light-deployment` |
| functions | mode | `sync`, `async`, `stream` |
| functions | source_type | `git`, `zip`, `inline` |
| functions | status | `pending`, `building`, `ready`, `failed`, `deleted` |
| functions | runtime | `(preset IS NULL AND dockerfile IS NOT NULL) OR (preset IS NOT NULL AND dockerfile IS NULL)` |
| invocations | trigger_type | `http`, `database_change`, `object_storage` |
| invocations | mode | `sync`, `async`, `stream` |
| invocations | status | `pending`, `running`, `succeeded`, `failed`, `timeout`, `cancelled` |
| event_history | trigger_type | `database_change`, `object_storage` |
| event_history | status | `delivered`, `failed`, `retrying` |

## インデックス

| テーブル | インデックス | 条件 |
|---------|-------------|------|
| projects | `idx_projects_tenant_id(tenant_id)` | |
| projects | `idx_projects_status(status)` | `WHERE status != 'deleted'` |
| api_keys | `idx_api_keys_key_hash(key_hash)` | `WHERE revoked_at IS NULL` |
| api_keys | `idx_api_keys_project_id(project_id)` | |
| functions | `idx_functions_project_id(project_id)` | |
| invocations | `idx_invocations_project_created(project_id, created_at DESC)` | |
| invocations | `idx_invocations_function_created(function_id, created_at DESC)` | |
| event_history | `idx_event_history_project_created(project_id, created_at DESC)` | |
| event_history | `idx_event_history_invocation(invocation_id)` | `WHERE invocation_id IS NOT NULL` |
| platform_events | `idx_platform_events_tenant_created(tenant_id, created_at DESC)` | |
| platform_events | `idx_platform_events_project_created(project_id, created_at DESC)` | |

## 補足設計

### Operator が kind から自動決定する値（DBカラムに持たない）

| kind | 実行方式 | replicas | CPU | Memory |
|------|---------|----------|-----|--------|
| heavy-job | Job (使い捨て) | min=0, max=1 | 1000m | 1Gi |
| heavy-deployment | Deployment (KEDA, scale_down=300s) | min=0, max=10 | 500m | 512Mi |
| light-deployment | Deployment (HPA) | min=1, max=10 | 250m | 256Mi |

### タイムアウト上限（アプリ層バリデーション）

| kind | GPU | 上限 |
|------|-----|------|
| light-deployment | - | 30s |
| heavy-* | CPU のみ | 3600s |
| heavy-* | GPU あり | 無制限 |

### gpu_config JSONB 例

```json
{"type": "H100", "provider": "runpod", "product": "serverless"}
```

NULL の場合は GPU 不使用。

### イベント → 実行フロー

1. イベント発火 → event_history INSERT (`invocation_id = NULL`)
2. Function 実行開始 → invocations INSERT
3. `invocation_id` 確定 → event_history UPDATE
4. `status = failed` でリトライ尽きた場合 → `invocation_id = NULL` のまま（未実行イベントの可視化）

### Secret → Function 紐付け方針

- `linked_function_ids (uuid[])` は不採用（FK制約不可、削除時の配列更新が面倒）
- functions の `env_vars` に Secret 名を記述（例: `{"OPENAI_API_KEY": ""}`）
- Operator が Pod 起動時に k8s Secret から値を注入

### Phase 1 スコープ外

- cron トリガーは実装しない（triggers JSONB にも含めない）
- usage_daily はテーブル作成のみ、集計ロジックは実装しない

## 共通 Trigger（updated_at 自動更新）

```sql
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```

対象テーブル: tenants, projects, functions, secrets_metadata

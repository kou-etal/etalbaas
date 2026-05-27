import { gotrue } from "@/lib/auth/gotrue-client";

const API_BASE_URL = (process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080").replace(/\/$/, "");

async function getAuthHeaders(): Promise<Record<string, string>> {
  const { data } = await gotrue.getSession();
  if (data.session?.access_token) {
    return { Authorization: `Bearer ${data.session.access_token}` };
  }
  return {};
}

async function rpc<T = Record<string, unknown>>(service: string, method: string, input: Record<string, unknown> = {}): Promise<T> {
  const url = `${API_BASE_URL}/${service}/${method}`;
  const authHeaders = await getAuthHeaders();

  const response = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...authHeaders,
    },
    body: JSON.stringify(input),
  });

  if (!response.ok) {
    const status = response.status;
    throw new Error(`RPC ${method} failed with status ${status}`);
  }

  const text = await response.text();
  if (!text) return {} as T;
  return JSON.parse(text) as T;
}

// --- Input types ---

interface CreateProjectInput {
  displayName: string;
  description: string;
  postgresEnabled?: boolean;
  postgresExtensions?: string[];
  redisEnabled?: boolean;
  postgrestEnabled?: boolean;
}

interface ProjectIdInput {
  projectId: string;
}

interface CreateApiKeyInput {
  projectId: string;
  name: string;
  role: string;
  expiresInDays?: number;
}

interface RevokeApiKeyInput {
  projectId: string;
  apiKeyId: string;
}

interface CreateFunctionInput {
  projectId: string;
  name: string;
  displayName?: string;
  kind: string;
  mode: string;
  inlineSource?: { code: string; filename: string };
  gitSource?: { repoUrl: string; branch: string; subpath: string };
  presetRuntime?: { preset: string; requirements?: string[] };
  customRuntime?: { dockerfile: string };
  timeoutSec?: number;
  gpuConfig?: { type: string; provider: string; product: string };
  triggers?: Array<{
    databaseChange?: { table: string; events: string[]; filter?: string };
    objectStorage?: { bucket: string; events: string[]; prefix: string };
  }>;
  envVars?: Array<{ name: string; value?: string; secretName?: string }>;
}

interface FunctionIdInput {
  projectId: string;
  functionId: string;
}

interface UpdateFunctionInput {
  projectId: string;
  functionId: string;
  [key: string]: unknown;
}

interface ListEventHistoryInput {
  projectId: string;
  functionId?: string;
}

interface CreateSecretInput {
  projectId: string;
  name: string;
  value: string;
  description?: string;
}

interface DeleteSecretInput {
  projectId: string;
  secretId: string;
}

interface UpdateSecretValueInput {
  projectId: string;
  secretId: string;
  value: string;
}

interface ListInvocationsInput {
  projectId: string;
  functionId: string;
  statusFilter?: string;
}

interface Invocation {
  id: string;
  functionId: string;
  projectId: string;
  triggerType: string;
  mode: string;
  status: string;
  errorMessage: string;
  retryCount: number;
  durationMs: number;
  coldStartMs: number;
  gpuDurationMs: number;
  memoryPeakBytes: number;
  cpuMillis: number;
  gpuProvider: string;
  gpuType: string;
  traceId: string;
  startedAt: string;
  completedAt: string;
  createdAt: string;
}

interface CreateBucketInput {
  projectId: string;
  name: string;
  accessLevel: string;
}

// --- Response types ---

interface Project {
  id: string;
  tenantId: string;
  displayName: string;
  description: string;
  status: string;
  postgresEnabled: boolean;
  postgresExtensions: string[];
  redisEnabled: boolean;
  postgrestEnabled: boolean;
  createdAt: string;
  updatedAt: string;
}

interface EnvVar {
  name: string;
  value?: string;
  secretName?: string;
}

interface FunctionItem {
  id: string;
  projectId: string;
  name: string;
  displayName: string;
  kind: string;
  mode: string;
  inlineSource?: { code: string; filename: string };
  gitSource?: { repoUrl: string; branch: string; subpath: string };
  presetRuntime?: { preset: string; requirements: string[] };
  customRuntime?: { dockerfile: string };
  timeoutSec: number;
  gpuConfig?: { type: string; provider: string; product: string };
  triggers?: Array<{
    databaseChange?: { table: string; events: string[]; filter?: string; includeColumns?: string[] };
    objectStorage?: { bucket: string; events: string[]; prefix: string };
  }>;
  envVars: EnvVar[];
  status: string;
  buildImageRef: string;
  buildImageDigest: string;
  buildDurationSec: number;
  lastBuiltAt: string;
  createdAt: string;
  updatedAt: string;
}

interface EventItem {
  id: string;
  projectId: string;
  functionId: string;
  invocationId: string;
  trigger?: {
    databaseChange?: { table: string; event: string };
    objectStorage?: { bucket: string; objectKey: string; event: string };
  };
  status: string;
  attemptCount: number;
  lastError: string;
  traceId: string;
  createdAt: string;
}

interface SecretItem {
  id: string;
  projectId: string;
  name: string;
  description: string;
  createdAt: string;
  updatedAt: string;
}

interface ApiKeyItem {
  id: string;
  projectId: string;
  name: string;
  keyPrefix: string;
  role: string;
  expiresAt?: string;
  revokedAt?: string;
  createdAt: string;
}

interface Bucket {
  id: string;
  projectId: string;
  name: string;
  accessLevel: string;
  fileSizeLimit: number;
  allowedMimeTypes: string[];
  createdAt: string;
  updatedAt: string;
}

interface UserInfo {
  id: string;
  email: string;
  tenantId: string;
}

export interface StorageObject {
  id: string;
  bucket_id: string;
  name: string;
  size: number | null;
  mime_type: string | null;
  etag: string | null;
  created_at: string;
  updated_at: string;
}

// --- Service clients ---

// Project Service
export const projectClient = {
  createProject: (input: CreateProjectInput) =>
    rpc<{ project: Project }>("etalbaas.project.v1.ProjectService", "CreateProject", input as unknown as Record<string, unknown>),
  listProjects: (input: Record<string, unknown> = {}) =>
    rpc<{ projects: Project[] }>("etalbaas.project.v1.ProjectService", "ListProjects", input),
  getProject: (input: ProjectIdInput) =>
    rpc<{ project: Project }>("etalbaas.project.v1.ProjectService", "GetProject", input as unknown as Record<string, unknown>),
  deleteProject: (input: ProjectIdInput) =>
    rpc<{ project: Project }>("etalbaas.project.v1.ProjectService", "DeleteProject", input as unknown as Record<string, unknown>),
  pauseProject: (input: ProjectIdInput) =>
    rpc<{ project: Project }>("etalbaas.project.v1.ProjectService", "PauseProject", input as unknown as Record<string, unknown>),
  resumeProject: (input: ProjectIdInput) =>
    rpc<{ project: Project }>("etalbaas.project.v1.ProjectService", "ResumeProject", input as unknown as Record<string, unknown>),
  createApiKey: (input: CreateApiKeyInput) =>
    rpc<{ apiKey: ApiKeyItem; rawKey: string }>("etalbaas.project.v1.ProjectService", "CreateApiKey", input as unknown as Record<string, unknown>),
  listApiKeys: (input: ProjectIdInput) =>
    rpc<{ apiKeys: ApiKeyItem[] }>("etalbaas.project.v1.ProjectService", "ListApiKeys", input as unknown as Record<string, unknown>),
  revokeApiKey: (input: RevokeApiKeyInput) =>
    rpc<{ apiKey: ApiKeyItem }>("etalbaas.project.v1.ProjectService", "RevokeApiKey", input as unknown as Record<string, unknown>),
};

// Function Service
export const functionClient = {
  createFunction: (input: CreateFunctionInput) =>
    rpc<{ function: FunctionItem }>("etalbaas.function.v1.FunctionService", "CreateFunction", input as unknown as Record<string, unknown>),
  getFunction: (input: FunctionIdInput) =>
    rpc<{ function: FunctionItem }>("etalbaas.function.v1.FunctionService", "GetFunction", input as unknown as Record<string, unknown>),
  listFunctions: (input: ProjectIdInput) =>
    rpc<{ functions: FunctionItem[] }>("etalbaas.function.v1.FunctionService", "ListFunctions", input as unknown as Record<string, unknown>),
  updateFunction: (input: UpdateFunctionInput) =>
    rpc<{ function: FunctionItem }>("etalbaas.function.v1.FunctionService", "UpdateFunction", input as unknown as Record<string, unknown>),
  deleteFunction: (input: FunctionIdInput) =>
    rpc<Record<string, unknown>>("etalbaas.function.v1.FunctionService", "DeleteFunction", input as unknown as Record<string, unknown>),
  listInvocations: (input: ListInvocationsInput) =>
    rpc<{ invocations: Invocation[] }>("etalbaas.function.v1.FunctionService", "ListInvocations", input as unknown as Record<string, unknown>),
};

// Event Service
export const eventClient = {
  listEventHistory: (input: ListEventHistoryInput) =>
    rpc<{ events: EventItem[] }>("etalbaas.event.v1.EventService", "ListEventHistory", input as unknown as Record<string, unknown>),
};

// Secret Service
export const secretClient = {
  createSecret: (input: CreateSecretInput) =>
    rpc<{ secret: SecretItem }>("etalbaas.secret.v1.SecretService", "CreateSecret", input as unknown as Record<string, unknown>),
  listSecrets: (input: ProjectIdInput) =>
    rpc<{ secrets: SecretItem[] }>("etalbaas.secret.v1.SecretService", "ListSecrets", input as unknown as Record<string, unknown>),
  deleteSecret: (input: DeleteSecretInput) =>
    rpc<Record<string, unknown>>("etalbaas.secret.v1.SecretService", "DeleteSecret", input as unknown as Record<string, unknown>),
  updateSecretValue: (input: UpdateSecretValueInput) =>
    rpc<{ secret: SecretItem }>("etalbaas.secret.v1.SecretService", "UpdateSecretValue", input as unknown as Record<string, unknown>),
};

// Storage Service
export const storageClient = {
  createBucket: (input: CreateBucketInput) =>
    rpc<{ bucket: Bucket }>("etalbaas.storage.v1.StorageService", "CreateBucket", input as unknown as Record<string, unknown>),
  listBuckets: (input: ProjectIdInput) =>
    rpc<{ buckets: Bucket[] }>("etalbaas.storage.v1.StorageService", "ListBuckets", input as unknown as Record<string, unknown>),
};

// Tenant Service
export const tenantClient = {
  getMe: (input: Record<string, unknown> = {}) =>
    rpc<{ user: UserInfo }>("etalbaas.tenant.v1.TenantService", "GetMe", input),
};

// Storage REST (dashboard file operations via JWT auth)
async function storageRest<T>(method: string, path: string, body?: BodyInit | null, extraHeaders?: Record<string, string>): Promise<T> {
  const url = `${API_BASE_URL}${path}`;
  const authHeaders = await getAuthHeaders();

  const response = await fetch(url, {
    method,
    headers: {
      ...authHeaders,
      ...extraHeaders,
    },
    body,
  });

  if (!response.ok) {
    throw new Error(`Storage ${method} ${path} failed with status ${response.status}`);
  }

  const text = await response.text();
  if (!text) return {} as T;
  return JSON.parse(text) as T;
}

// --- postgres-meta client (DDL / Schema → /api/database proxy) ---

async function pgMetaFetch<T>(
  method: string,
  projectId: string,
  path: string,
  body?: unknown,
): Promise<T> {
  const params = new URLSearchParams({ project_id: projectId });
  const url = `/api/database/${path.replace(/^\//, "")}?${params}`;
  const authHeaders = await getAuthHeaders();

  const response = await fetch(url, {
    method,
    headers: {
      "Content-Type": "application/json",
      ...authHeaders,
    },
    body: body ? JSON.stringify(body) : undefined,
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || `pgMeta ${method} ${path} failed with status ${response.status}`);
  }

  const text = await response.text();
  if (!text) return [] as unknown as T;
  return JSON.parse(text) as T;
}

export interface PgTable {
  id: number;
  schema: string;
  name: string;
  rls_enabled: boolean;
  rls_forced: boolean;
  replica_identity: string;
  bytes: number;
  size: string;
  live_rows_estimate: number;
  dead_rows_estimate: number;
  comment: string | null;
  columns: PgColumn[];
  primary_keys: { schema: string; table_name: string; name: string }[];
}

export interface PgColumn {
  table_id: number;
  schema: string;
  table: string;
  id: string;
  ordinal_position: number;
  name: string;
  default_value: string | null;
  data_type: string;
  format: string;
  is_identity: boolean;
  identity_generation: string | null;
  is_generated: boolean;
  is_nullable: boolean;
  is_updatable: boolean;
  is_unique: boolean;
  enums: string[];
  comment: string | null;
}

export interface PgPolicy {
  id: number;
  schema: string;
  table: string;
  table_id: number;
  name: string;
  action: string;
  roles: string[];
  command: string;
  definition: string;
  check: string | null;
}

export interface PgIndex {
  id: number;
  table_id: number;
  schema: string;
  table: string;
  name: string;
  columns: string;
  comment: string | null;
}

// postgres-meta /query returns a plain array of row objects (e.g. [{ "result": 1 }])
export type QueryResult = Record<string, unknown>[];

export const pgMetaClient = {
  // Tables
  listTables: (projectId: string) =>
    pgMetaFetch<PgTable[]>("GET", projectId, "tables"),
  getTable: (projectId: string, tableId: number) =>
    pgMetaFetch<PgTable>("GET", projectId, `tables/${tableId}`),
  createTable: (projectId: string, body: { name: string; schema?: string; comment?: string }) =>
    pgMetaFetch<PgTable>("POST", projectId, "tables", body),
  updateTable: (projectId: string, tableId: number, body: { name?: string; rls_enabled?: boolean; comment?: string }) =>
    pgMetaFetch<PgTable>("PATCH", projectId, `tables/${tableId}`, body),
  deleteTable: (projectId: string, tableId: number) =>
    pgMetaFetch<PgTable>("DELETE", projectId, `tables/${tableId}`),

  // Columns — postgres-meta doesn't filter by table_id server-side,
  // so we always fetch all columns. Filtering is done in the hook.
  listColumns: (projectId: string, _tableId?: number) =>
    pgMetaFetch<PgColumn[]>("GET", projectId, "columns"),
  createColumn: (projectId: string, body: { table_id: number; name: string; type: string; default_value?: string; is_nullable?: boolean; is_unique?: boolean; comment?: string }) =>
    pgMetaFetch<PgColumn>("POST", projectId, "columns", body),
  updateColumn: (projectId: string, columnId: string, body: { name?: string; type?: string; default_value?: string; is_nullable?: boolean; comment?: string }) =>
    pgMetaFetch<PgColumn>("PATCH", projectId, `columns/${columnId}`, body),
  deleteColumn: (projectId: string, columnId: string) =>
    pgMetaFetch<PgColumn>("DELETE", projectId, `columns/${columnId}`),

  // Policies (RLS)
  listPolicies: (projectId: string) =>
    pgMetaFetch<PgPolicy[]>("GET", projectId, "policies"),
  createPolicy: (projectId: string, body: { name: string; table: string; schema?: string; definition: string; check?: string; action?: string; command?: string; roles?: string[] }) =>
    pgMetaFetch<PgPolicy>("POST", projectId, "policies", body),
  updatePolicy: (projectId: string, policyId: number, body: { name?: string; definition?: string; check?: string; roles?: string[] }) =>
    pgMetaFetch<PgPolicy>("PATCH", projectId, `policies/${policyId}`, body),
  deletePolicy: (projectId: string, policyId: number) =>
    pgMetaFetch<PgPolicy>("DELETE", projectId, `policies/${policyId}`),

  // SQL Editor
  executeQuery: (projectId: string, query: string) =>
    pgMetaFetch<QueryResult>("POST", projectId, "query", { query }),
};

// --- PostgREST data client (DML → /api/postgrest proxy, SQL injection safe) ---

async function postgrestFetch<T>(
  method: string,
  projectId: string,
  path: string,
  body?: unknown,
  extraHeaders?: Record<string, string>,
): Promise<T> {
  const params = new URLSearchParams({ project_id: projectId });
  // path may contain query params (e.g. /users?id=eq.1), merge with project_id
  const [basePath, queryString] = path.replace(/^\//, "").split("?");
  if (queryString) {
    const existing = new URLSearchParams(queryString);
    existing.forEach((v, k) => params.set(k, v));
  }
  const url = `/api/postgrest/${basePath}?${params}`;
  const authHeaders = await getAuthHeaders();

  const response = await fetch(url, {
    method,
    headers: {
      "Content-Type": "application/json",
      ...authHeaders,
      ...extraHeaders,
    },
    body: body ? JSON.stringify(body) : undefined,
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || `PostgREST ${method} ${path} failed with status ${response.status}`);
  }

  // HEAD requests return content-range header for row count
  if (method === "HEAD") {
    const contentRange = response.headers.get("content-range");
    // Format: "0-24/100" or "*/100"
    const total = contentRange?.split("/")[1];
    return (total ? parseInt(total, 10) : 0) as unknown as T;
  }

  const text = await response.text();
  if (!text) return [] as unknown as T;
  return JSON.parse(text) as T;
}

export const postgrestDataClient = {
  getTableData: (projectId: string, table: string, limit = 50, offset = 0) =>
    postgrestFetch<Record<string, unknown>[]>(
      "GET",
      projectId,
      `/${table}?limit=${limit}&offset=${offset}`,
    ),

  insertRow: (projectId: string, table: string, data: Record<string, unknown>) =>
    postgrestFetch<Record<string, unknown>>(
      "POST",
      projectId,
      `/${table}`,
      data,
      { Prefer: "return=representation" },
    ),

  updateRow: (projectId: string, table: string, pkCol: string, pkVal: unknown, data: Record<string, unknown>) =>
    postgrestFetch<Record<string, unknown>>(
      "PATCH",
      projectId,
      `/${table}?${pkCol}=eq.${encodeURIComponent(String(pkVal))}`,
      data,
      { Prefer: "return=representation" },
    ),

  deleteRow: (projectId: string, table: string, pkCol: string, pkVal: unknown) =>
    postgrestFetch<Record<string, unknown>>(
      "DELETE",
      projectId,
      `/${table}?${pkCol}=eq.${encodeURIComponent(String(pkVal))}`,
    ),

  getRowCount: (projectId: string, table: string) =>
    postgrestFetch<number>("HEAD", projectId, `/${table}`, null, {
      Prefer: "count=exact",
    }),
};

export const storageRestClient = {
  listObjects: (projectId: string, bucket: string, limit?: number, offset?: number) => {
    const params = new URLSearchParams({ project_id: projectId, bucket });
    if (limit) params.set("limit", String(limit));
    if (offset) params.set("offset", String(offset));
    return storageRest<StorageObject[]>("GET", `/storage/v1/dashboard/objects?${params}`);
  },

  uploadObject: async (projectId: string, bucket: string, path: string, file: File) => {
    const params = new URLSearchParams({ project_id: projectId, bucket, path });
    return storageRest<StorageObject>("POST", `/storage/v1/dashboard/upload?${params}`, file, {
      "Content-Type": file.type || "application/octet-stream",
    });
  },

  downloadObject: async (projectId: string, bucket: string, path: string): Promise<Response> => {
    const params = new URLSearchParams({ project_id: projectId, bucket, path });
    const url = `${API_BASE_URL}/storage/v1/dashboard/download?${params}`;
    const authHeaders = await getAuthHeaders();
    const response = await fetch(url, { headers: authHeaders });
    if (!response.ok) {
      throw new Error(`Download failed with status ${response.status}`);
    }
    return response;
  },

  deleteObject: (projectId: string, bucket: string, path: string) => {
    const params = new URLSearchParams({ project_id: projectId, bucket, path });
    return storageRest<{ status: string }>("DELETE", `/storage/v1/dashboard/delete?${params}`);
  },
};

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

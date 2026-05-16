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
  name: string;
  description: string;
  region: string;
}

interface ProjectIdInput {
  projectId: string;
}

interface CreateApiKeyInput {
  projectId: string;
  name: string;
}

interface RevokeApiKeyInput {
  projectId: string;
  keyId: string;
}

interface CreateFunctionInput {
  projectId: string;
  name: string;
  runtime: string;
  kind: string;
  mode: string;
}

interface FunctionIdInput {
  functionId: string;
}

interface UpdateFunctionInput {
  functionId: string;
  [key: string]: unknown;
}

interface ListEventHistoryInput {
  projectId: string;
}

interface CreateSecretInput {
  projectId: string;
  key: string;
  value: string;
}

interface DeleteSecretInput {
  projectId: string;
  key: string;
}

interface CreateBucketInput {
  projectId: string;
  name: string;
  isPublic: boolean;
}

// --- Response types ---

interface Project {
  id: string;
  name: string;
  description: string;
  region: string;
  status: string;
  createdAt: string;
}

interface FunctionItem {
  id: string;
  name: string;
  runtime: string;
  kind: string;
  mode: string;
  status: string;
  entrypoint: string;
  image: string;
  lastBuildAt: string;
  updatedAt: string;
  timeout: number;
  minReplicas: number;
  maxReplicas: number;
  gpu: string;
  resources: { cpu: string; memory: string };
  envVars: Record<string, string>;
}

interface EventItem {
  id: string;
  subject: string;
  type: string;
  createdAt: string;
  data?: Record<string, unknown>;
}

interface SecretItem {
  key: string;
  createdAt: string;
}

interface ApiKeyItem {
  id: string;
  name: string;
  prefix: string;
  createdAt: string;
}

interface Bucket {
  id: string;
  name: string;
  isPublic: boolean;
  objectCount: number;
  createdAt: string;
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
    rpc<Record<string, unknown>>("etalbaas.project.v1.ProjectService", "DeleteProject", input as unknown as Record<string, unknown>),
  createApiKey: (input: CreateApiKeyInput) =>
    rpc<{ apiKey: ApiKeyItem; key: string }>("etalbaas.project.v1.ProjectService", "CreateApiKey", input as unknown as Record<string, unknown>),
  listApiKeys: (input: ProjectIdInput) =>
    rpc<{ apiKeys: ApiKeyItem[] }>("etalbaas.project.v1.ProjectService", "ListApiKeys", input as unknown as Record<string, unknown>),
  revokeApiKey: (input: RevokeApiKeyInput) =>
    rpc<Record<string, unknown>>("etalbaas.project.v1.ProjectService", "RevokeApiKey", input as unknown as Record<string, unknown>),
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
};

// Event Service
export const eventClient = {
  listEventHistory: (input: ListEventHistoryInput) =>
    rpc<{ events: EventItem[] }>("etalbaas.event.v1.EventService", "ListEventHistory", input as unknown as Record<string, unknown>),
};

// Secret Service
export const secretClient = {
  createSecret: (input: CreateSecretInput) =>
    rpc<Record<string, unknown>>("etalbaas.secret.v1.SecretService", "CreateSecret", input as unknown as Record<string, unknown>),
  listSecrets: (input: ProjectIdInput) =>
    rpc<{ secrets: SecretItem[] }>("etalbaas.secret.v1.SecretService", "ListSecrets", input as unknown as Record<string, unknown>),
  deleteSecret: (input: DeleteSecretInput) =>
    rpc<Record<string, unknown>>("etalbaas.secret.v1.SecretService", "DeleteSecret", input as unknown as Record<string, unknown>),
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

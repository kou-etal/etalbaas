/**
 * E2E Test API Helpers
 *
 * Direct API calls for test data setup/teardown. Uses the same RPC
 * protocol as the dashboard (Connect over HTTP POST) and the REST
 * storage endpoints.
 */

// Allow self-signed certificates in Kind cluster
process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";

const GOTRUE_API_URL =
  process.env.E2E_GOTRUE_API_URL || "http://localhost:9999";
const API_URL =
  process.env.E2E_API_URL || "https://api.local.etalbaas.dev";
const BASE_URL =
  process.env.E2E_BASE_URL || "https://dashboard.local.etalbaas.dev";
const EMAIL = process.env.E2E_USER_EMAIL || "e2e-test@etalbaas.dev";
const PASSWORD = process.env.E2E_USER_PASSWORD || "E2eTestPassword123!";

/* ===== Auth ===== */

let cachedToken: string | null = null;

export async function getAuthToken(): Promise<string> {
  if (cachedToken) return cachedToken;

  const res = await fetch(`${GOTRUE_API_URL}/token?grant_type=password`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email: EMAIL, password: PASSWORD }),
  });

  if (!res.ok) {
    throw new Error(`GoTrue login failed (${res.status}): ${await res.text()}`);
  }

  const data = await res.json();
  cachedToken = data.access_token as string;
  return cachedToken;
}

/* ===== RPC helper ===== */

async function rpc<T = Record<string, unknown>>(
  service: string,
  method: string,
  input: Record<string, unknown> = {},
): Promise<T> {
  const token = await getAuthToken();
  const url = `${API_URL}/${service}/${method}`;

  const res = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(input),
  });

  if (!res.ok) {
    const body = await res.text();
    throw new Error(`RPC ${service}/${method} failed (${res.status}): ${body}`);
  }

  const text = await res.text();
  if (!text) return {} as T;
  return JSON.parse(text) as T;
}

/* ===== REST helper (storage dashboard endpoints) ===== */

async function storageRest<T = unknown>(
  method: string,
  path: string,
  body?: BodyInit | null,
  extraHeaders?: Record<string, string>,
): Promise<T> {
  const token = await getAuthToken();
  const url = `${API_URL}${path}`;

  const res = await fetch(url, {
    method,
    headers: {
      Authorization: `Bearer ${token}`,
      ...extraHeaders,
    },
    body,
  });

  if (!res.ok) {
    const errBody = await res.text();
    throw new Error(
      `Storage REST ${method} ${path} failed (${res.status}): ${errBody}`,
    );
  }

  const text = await res.text();
  if (!text) return {} as T;
  return JSON.parse(text) as T;
}

/* ===== Unique name generator ===== */

export function uniqueName(prefix: string): string {
  const ts = Date.now().toString(36);
  const rand = Math.random().toString(36).slice(2, 6);
  return `e2e-${prefix}-${ts}-${rand}`;
}

/* ===== Project CRUD ===== */

interface Project {
  id: string;
  tenantId: string;
  displayName: string;
  description: string;
  status: string;
  postgresEnabled: boolean;
  redisEnabled: boolean;
  postgrestEnabled: boolean;
  postgresExtensions: string[];
  createdAt: string;
  updatedAt: string;
}

export async function createTestProject(
  prefix: string,
  options: {
    postgresEnabled?: boolean;
    redisEnabled?: boolean;
    postgrestEnabled?: boolean;
    postgresExtensions?: string[];
  } = {},
): Promise<Project> {
  const res = await rpc<{ project: Project }>(
    "etalbaas.project.v1.ProjectService",
    "CreateProject",
    {
      displayName: uniqueName(prefix),
      description: `E2E test project (${prefix})`,
      postgresEnabled: options.postgresEnabled ?? false,
      redisEnabled: options.redisEnabled ?? false,
      postgrestEnabled: options.postgrestEnabled ?? false,
      postgresExtensions: options.postgresExtensions ?? [],
    },
  );
  return res.project;
}

export async function deleteTestProject(projectId: string): Promise<void> {
  await rpc("etalbaas.project.v1.ProjectService", "DeleteProject", {
    projectId,
  });
}

export async function getProject(projectId: string): Promise<Project> {
  const res = await rpc<{ project: Project }>(
    "etalbaas.project.v1.ProjectService",
    "GetProject",
    { projectId },
  );
  return res.project;
}

export async function listProjects(): Promise<Project[]> {
  const res = await rpc<{ projects: Project[] }>(
    "etalbaas.project.v1.ProjectService",
    "ListProjects",
  );
  return res.projects || [];
}

/* ===== Function CRUD ===== */

interface FunctionItem {
  id: string;
  projectId: string;
  name: string;
  displayName: string;
  kind: string;
  mode: string;
  status: string;
  createdAt: string;
}

export async function createTestFunction(
  projectId: string,
  prefix: string,
  options: {
    kind?: string;
    mode?: string;
    inlineSource?: { code: string; filename: string };
  } = {},
): Promise<FunctionItem> {
  const name = uniqueName(prefix);
  const res = await rpc<{ function: FunctionItem }>(
    "etalbaas.function.v1.FunctionService",
    "CreateFunction",
    {
      projectId,
      name,
      displayName: name,
      kind: options.kind ?? "light-deployment",
      mode: options.mode ?? "sync",
      inlineSource: options.inlineSource ?? {
        code: 'export default async function(req) { return new Response("ok"); }',
        filename: "handler.js",
      },
      presetRuntime: { preset: "node-20" },
    },
  );
  return res.function;
}

export async function deleteTestFunction(
  projectId: string,
  functionId: string,
): Promise<void> {
  await rpc("etalbaas.function.v1.FunctionService", "DeleteFunction", {
    projectId,
    functionId,
  });
}

export async function listFunctions(projectId: string): Promise<FunctionItem[]> {
  const res = await rpc<{ functions: FunctionItem[] }>(
    "etalbaas.function.v1.FunctionService",
    "ListFunctions",
    { projectId },
  );
  return res.functions || [];
}

/* ===== Secret CRUD ===== */

interface SecretItem {
  id: string;
  projectId: string;
  name: string;
  description: string;
  createdAt: string;
  updatedAt: string;
}

export async function createTestSecret(
  projectId: string,
  name: string,
  value: string,
  description?: string,
): Promise<SecretItem> {
  const res = await rpc<{ secret: SecretItem }>(
    "etalbaas.secret.v1.SecretService",
    "CreateSecret",
    { projectId, name, value, description: description ?? "" },
  );
  return res.secret;
}

export async function deleteTestSecret(
  projectId: string,
  secretId: string,
): Promise<void> {
  await rpc("etalbaas.secret.v1.SecretService", "DeleteSecret", {
    projectId,
    secretId,
  });
}

export async function listSecrets(projectId: string): Promise<SecretItem[]> {
  const res = await rpc<{ secrets: SecretItem[] }>(
    "etalbaas.secret.v1.SecretService",
    "ListSecrets",
    { projectId },
  );
  return res.secrets || [];
}

/* ===== API Key CRUD ===== */

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

export async function createTestApiKey(
  projectId: string,
  name: string,
  role: string,
  expiresInDays?: number,
): Promise<{ apiKey: ApiKeyItem; rawKey: string }> {
  return rpc<{ apiKey: ApiKeyItem; rawKey: string }>(
    "etalbaas.project.v1.ProjectService",
    "CreateApiKey",
    { projectId, name, role, expiresInDays },
  );
}

export async function revokeTestApiKey(
  projectId: string,
  apiKeyId: string,
): Promise<void> {
  await rpc("etalbaas.project.v1.ProjectService", "RevokeApiKey", {
    projectId,
    apiKeyId,
  });
}

export async function listApiKeys(projectId: string): Promise<ApiKeyItem[]> {
  const res = await rpc<{ apiKeys: ApiKeyItem[] }>(
    "etalbaas.project.v1.ProjectService",
    "ListApiKeys",
    { projectId },
  );
  return res.apiKeys || [];
}

/* ===== Storage Bucket CRUD ===== */

interface Bucket {
  id: string;
  projectId: string;
  name: string;
  accessLevel: string;
  createdAt: string;
}

export async function createTestBucket(
  projectId: string,
  name: string,
  accessLevel: string = "private",
): Promise<Bucket> {
  const res = await rpc<{ bucket: Bucket }>(
    "etalbaas.storage.v1.StorageService",
    "CreateBucket",
    { projectId, name, accessLevel },
  );
  return res.bucket;
}

export async function listBuckets(projectId: string): Promise<Bucket[]> {
  const res = await rpc<{ buckets: Bucket[] }>(
    "etalbaas.storage.v1.StorageService",
    "ListBuckets",
    { projectId },
  );
  return res.buckets || [];
}

/* ===== Storage Object Operations (dashboard REST) ===== */

interface StorageObject {
  id: string;
  bucket_id: string;
  name: string;
  size: number | null;
  mime_type: string | null;
  created_at: string;
}

export async function uploadTestFile(
  projectId: string,
  bucket: string,
  path: string,
  content: string,
  contentType: string = "text/plain",
): Promise<StorageObject> {
  const params = new URLSearchParams({ project_id: projectId, bucket, path });
  return storageRest<StorageObject>(
    "POST",
    `/storage/v1/dashboard/upload?${params}`,
    content,
    { "Content-Type": contentType },
  );
}

export async function listObjects(
  projectId: string,
  bucket: string,
): Promise<StorageObject[]> {
  const params = new URLSearchParams({ project_id: projectId, bucket });
  return storageRest<StorageObject[]>(
    "GET",
    `/storage/v1/dashboard/objects?${params}`,
  );
}

export async function deleteTestFile(
  projectId: string,
  bucket: string,
  path: string,
): Promise<void> {
  const params = new URLSearchParams({ project_id: projectId, bucket, path });
  await storageRest("DELETE", `/storage/v1/dashboard/delete?${params}`);
}

/* ===== Event History ===== */

interface EventItem {
  id: string;
  projectId: string;
  functionId: string;
  status: string;
  createdAt: string;
}

export async function listEventHistory(
  projectId: string,
  functionId?: string,
): Promise<EventItem[]> {
  const input: Record<string, unknown> = { projectId };
  if (functionId) input.functionId = functionId;
  const res = await rpc<{ events: EventItem[] }>(
    "etalbaas.event.v1.EventService",
    "ListEventHistory",
    input,
  );
  return res.events || [];
}

/* ===== Cleanup helper ===== */

/**
 * Delete all projects whose displayName starts with "e2e-".
 * Useful for test teardown.
 */
export async function cleanupE2eProjects(): Promise<void> {
  const projects = await listProjects();
  for (const p of projects) {
    if (p.displayName.startsWith("e2e-")) {
      try {
        await deleteTestProject(p.id);
      } catch {
        // Ignore errors during cleanup
      }
    }
  }
}

/* ===== URL helpers ===== */

export { API_URL, GOTRUE_API_URL, BASE_URL };

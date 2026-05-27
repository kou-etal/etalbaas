import { test, expect } from "@playwright/test";
import * as https from "node:https";
import { execSync } from "node:child_process";
import {
  createTestProject,
  deleteTestProject,
  API_URL,
} from "./helpers/api";

// Allow self-signed certificates in Kind cluster
process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";

const KUBECTL_BIN =
  process.env.KUBECTL_PATH ||
  "C:\\Program Files\\Docker\\Docker\\resources\\bin\\kubectl.exe";

/* ===== Helpers ===== */

/** Get project CRD phase directly via kubectl. */
function getCRDPhase(projectId: string): string {
  try {
    return execSync(
      `"${KUBECTL_BIN}" get project ${projectId} -n platform-system -o jsonpath="{.status.phase}"`,
      { encoding: "utf-8", timeout: 10_000 },
    )
      .trim()
      .replace(/^"|"$/g, "");
  } catch {
    return "";
  }
}

/** Poll CRD phase until Ready (or timeout). */
async function waitForProjectReady(
  projectId: string,
  timeoutMs = 300_000,
): Promise<void> {
  const start = Date.now();
  let lastPhase = "";
  while (Date.now() - start < timeoutMs) {
    lastPhase = getCRDPhase(projectId);
    if (lastPhase === "Ready") return;
    console.log(
      `[waitForProjectReady] project=${projectId} phase="${lastPhase}" elapsed=${Math.round((Date.now() - start) / 1000)}s`,
    );
    await new Promise((r) => setTimeout(r, 5_000));
  }
  throw new Error(
    `Project ${projectId} did not become READY within ${timeoutMs}ms (last phase: "${lastPhase}")`,
  );
}

/** Read a k8s Secret value (base64 decoded). */
function readSecretValue(
  namespace: string,
  secretName: string,
  key: string,
): string {
  const b64 = execSync(
    `"${KUBECTL_BIN}" get secret ${secretName} -n ${namespace} -o jsonpath="{.data.${key}}"`,
    { encoding: "utf-8", timeout: 10_000 },
  )
    .trim()
    .replace(/^"|"$/g, "");
  return Buffer.from(b64, "base64").toString("utf-8");
}

/** HTTPS request with custom DNS resolution for *.local.etalbaas.dev. */
async function httpsRequest(
  url: string,
  options: {
    method?: string;
    headers?: Record<string, string>;
    body?: string;
  } = {},
): Promise<{ status: number; body: unknown; text: string }> {
  const parsed = new URL(url);
  const method = options.method || "GET";
  const postData = options.body || "";

  const { status, text } = await new Promise<{ status: number; text: string }>(
    (resolve, reject) => {
      const req = https.request(
        {
          hostname: parsed.hostname,
          port: parsed.port || 443,
          path: parsed.pathname + parsed.search,
          method,
          headers: {
            ...options.headers,
            ...(postData
              ? {
                  "Content-Length": Buffer.byteLength(postData).toString(),
                }
              : {}),
          },
          rejectUnauthorized: false,
          timeout: 30_000,
          lookup: (hostname, opts, cb) => {
            if (hostname.endsWith(".local.etalbaas.dev")) {
              const result = { address: "127.0.0.1", family: 4 as const };
              if (typeof opts === "object" && opts?.all) {
                (cb as CallableFunction)(null, [result]);
              } else {
                (cb as CallableFunction)(null, result.address, result.family);
              }
              return;
            }
            import("node:dns").then((dns) =>
              dns.lookup(hostname, opts as never, cb as never),
            );
          },
        },
        (res) => {
          let data = "";
          res.on("data", (chunk: Buffer) => {
            data += chunk;
          });
          res.on("end", () => {
            resolve({ status: res.statusCode ?? 0, text: data });
          });
        },
      );
      req.on("error", reject);
      req.on("timeout", () => {
        req.destroy(new Error("request timeout"));
      });
      if (postData) req.write(postData);
      req.end();
    },
  );

  let parsed2: unknown;
  try {
    parsed2 = JSON.parse(text);
  } catch {
    parsed2 = text;
  }
  return { status, body: parsed2, text };
}

/** Decode JWT payload (no verification — just base64url decode). */
function decodeJWTPayload(jwt: string): Record<string, unknown> {
  const parts = jwt.split(".");
  if (parts.length !== 3) throw new Error("Invalid JWT format");
  const payload = Buffer.from(parts[1], "base64url").toString("utf-8");
  return JSON.parse(payload);
}

/** Decode JWT header. */
function decodeJWTHeader(jwt: string): Record<string, unknown> {
  const parts = jwt.split(".");
  if (parts.length !== 3) throw new Error("Invalid JWT format");
  const header = Buffer.from(parts[0], "base64url").toString("utf-8");
  return JSON.parse(header);
}

/* ===== Test suite ===== */

test.describe.serial("End-User BaaS Flow", () => {
  let project: { id: string; displayName: string };
  let suiteAvailable = false;

  let serviceRoleJWT = "";
  let anonJWT = "";

  // End-user credentials
  const testEmail = `testuser-${Date.now()}@example.com`;
  const testPassword = "TestPassword123!";
  let endUserAccessToken = "";
  let endUserSub = "";

  let projectApiBase = "";

  test.beforeAll(async () => {
    test.setTimeout(600_000);

    // Create project with Postgres + PostgREST
    project = await createTestProject("baas-flow", {
      postgresEnabled: true,
      postgrestEnabled: true,
    });
    console.log(`[beforeAll] Project created: ${project.id}`);

    await waitForProjectReady(project.id);
    console.log(`[beforeAll] Project ready: ${project.id}`);

    const apiUrl = new URL(API_URL);
    // Subdomain is the project ID in the CRD networking spec
    const subdomain = execSync(
      `"${KUBECTL_BIN}" get project ${project.id} -n platform-system -o jsonpath="{.spec.networking.subdomain}"`,
      { encoding: "utf-8", timeout: 10_000 },
    )
      .trim()
      .replace(/^"|"$/g, "");
    projectApiBase = `https://${subdomain}.api.${apiUrl.hostname.replace(/^api\./, "")}`;
    console.log(`[beforeAll] Project API base: ${projectApiBase}`);

    // Read pre-signed JWTs from Operator-created Secrets
    const ns = `project-${project.id}`;
    serviceRoleJWT = readSecretValue(ns, "service-role-key", "key");
    anonJWT = readSecretValue(ns, "anon-key", "key");

    console.log(
      `[beforeAll] service_role JWT length: ${serviceRoleJWT.length}`,
    );
    console.log(`[beforeAll] anon JWT length: ${anonJWT.length}`);

    if (!serviceRoleJWT || !anonJWT) {
      console.warn("[beforeAll] JWT Secrets not found, skipping suite");
      return;
    }

    suiteAvailable = true;
  });

  test.afterAll(async () => {
    test.setTimeout(120_000);
    if (project) {
      try {
        await deleteTestProject(project.id);
      } catch {
        // Ignore cleanup errors
      }
    }
  });

  function skipIfUnavailable() {
    test.skip(!suiteAvailable, "BaaS flow setup not ready");
  }

  /* ================================================================ */
  /*  1. GoTrue /health returns 200                                    */
  /* ================================================================ */
  test("1 - GoTrue /health returns 200", async () => {
    test.setTimeout(60_000);
    skipIfUnavailable();

    let lastResult: { status: number } | null = null;
    for (let attempt = 0; attempt < 12; attempt++) {
      const result = await httpsRequest(`${projectApiBase}/auth/health`);
      lastResult = result;
      console.log(
        `[test1] attempt=${attempt} status=${result.status} body=${JSON.stringify(result.body)}`,
      );
      if (result.status === 200) break;
      await new Promise((r) => setTimeout(r, 5_000));
    }

    expect(lastResult).toBeTruthy();
    expect(lastResult!.status).toBe(200);
  });

  /* ================================================================ */
  /*  2. Signup creates end-user                                       */
  /* ================================================================ */
  test("2 - signup creates end-user", async () => {
    test.setTimeout(60_000);
    skipIfUnavailable();

    const result = await httpsRequest(`${projectApiBase}/auth/signup`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        email: testEmail,
        password: testPassword,
      }),
    });

    console.log(`[test2] signup status=${result.status} body=${JSON.stringify(result.body)}`);
    expect(result.status).toBe(200);

    // GoTrue returns { user: { id, ... }, access_token, ... } when autoconfirm is enabled
    const body = result.body as Record<string, unknown>;
    const user = body.user as Record<string, unknown>;
    expect(user?.id).toBeTruthy();
    endUserSub = user.id as string;
    console.log(`[test2] user_id=${endUserSub}`);
  });

  /* ================================================================ */
  /*  3. Login returns RS256 JWT with authenticated role                */
  /* ================================================================ */
  test("3 - login returns RS256 JWT with authenticated role", async () => {
    test.setTimeout(60_000);
    skipIfUnavailable();

    const result = await httpsRequest(
      `${projectApiBase}/auth/token?grant_type=password`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          email: testEmail,
          password: testPassword,
        }),
      },
    );

    console.log(`[test3] login status=${result.status}`);
    expect(result.status).toBe(200);

    const body = result.body as Record<string, unknown>;
    expect(body.access_token).toBeTruthy();
    endUserAccessToken = body.access_token as string;

    // Verify JWT header is RS256
    const header = decodeJWTHeader(endUserAccessToken);
    console.log(`[test3] JWT header: ${JSON.stringify(header)}`);
    expect(header.alg).toBe("RS256");

    // Verify JWT claims
    const claims = decodeJWTPayload(endUserAccessToken);
    console.log(`[test3] JWT claims: ${JSON.stringify(claims)}`);
    expect(claims.role).toBe("authenticated");
    expect(claims.sub).toBe(endUserSub);
  });

  /* ================================================================ */
  /*  4. Create test table with RLS via kubectl                        */
  /* ================================================================ */
  test("4 - create test table with RLS via kubectl", async () => {
    test.setTimeout(60_000);
    skipIfUnavailable();

    const ns = `project-${project.id}`;
    // SET ROLE app so the table is owned by app (default privileges apply to PostgREST roles)
    const sql = `
      SET ROLE app;
      CREATE TABLE IF NOT EXISTS public.user_notes (
        id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
        user_id UUID DEFAULT auth.uid(),
        content TEXT NOT NULL,
        created_at TIMESTAMPTZ DEFAULT now()
      );
      ALTER TABLE public.user_notes ENABLE ROW LEVEL SECURITY;
      DROP POLICY IF EXISTS users_read_own ON public.user_notes;
      CREATE POLICY users_read_own ON public.user_notes
        FOR SELECT USING (user_id = auth.uid());
      DROP POLICY IF EXISTS users_insert_own ON public.user_notes;
      CREATE POLICY users_insert_own ON public.user_notes
        FOR INSERT WITH CHECK (user_id = auth.uid());
      DROP POLICY IF EXISTS svc_bypass ON public.user_notes;
      CREATE POLICY svc_bypass ON public.user_notes
        FOR ALL TO service_role USING (true) WITH CHECK (true);
      NOTIFY pgrst, 'reload schema';
    `;

    const result = execSync(
      `"${KUBECTL_BIN}" exec -n ${ns} db-1 -- psql -U postgres -d postgres -c "${sql.replace(/"/g, '\\"').replace(/\n/g, " ")}"`,
      { encoding: "utf-8", timeout: 30_000 },
    );
    console.log(`[test4] SQL result: ${result.trim()}`);

    // Wait a moment for PostgREST schema cache reload
    await new Promise((r) => setTimeout(r, 5_000));
  });

  /* ================================================================ */
  /*  5. Authenticated user can INSERT own note                        */
  /* ================================================================ */
  test("5 - authenticated user can INSERT own note", async () => {
    test.setTimeout(60_000);
    skipIfUnavailable();

    // Retry loop: PostgREST schema cache may need time to reload after table creation
    let result: { status: number; body: unknown } = { status: 0, body: {} };
    for (let attempt = 0; attempt < 10; attempt++) {
      result = await httpsRequest(
        `${projectApiBase}/rest/v1/user_notes`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${endUserAccessToken}`,
            Prefer: "return=representation",
            apikey: anonJWT,
          },
          body: JSON.stringify({ content: "Hello from E2E test" }),
        },
      );
      console.log(
        `[test5] attempt=${attempt} INSERT status=${result.status} body=${JSON.stringify(result.body)}`,
      );
      if (result.status === 201) break;
      // Wait for PostgREST to reload schema cache
      await new Promise((r) => setTimeout(r, 3_000));
    }

    expect(result.status).toBe(201);

    const rows = result.body as Array<Record<string, unknown>>;
    expect(rows.length).toBe(1);
    expect(rows[0].content).toBe("Hello from E2E test");
    expect(rows[0].user_id).toBe(endUserSub);
  });

  /* ================================================================ */
  /*  6. Authenticated user sees only own notes                        */
  /* ================================================================ */
  test("6 - authenticated user sees only own notes", async () => {
    test.setTimeout(60_000);
    skipIfUnavailable();

    const result = await httpsRequest(
      `${projectApiBase}/rest/v1/user_notes?select=*`,
      {
        method: "GET",
        headers: {
          Authorization: `Bearer ${endUserAccessToken}`,
          apikey: anonJWT,
        },
      },
    );

    console.log(
      `[test6] SELECT status=${result.status} body=${JSON.stringify(result.body)}`,
    );
    expect(result.status).toBe(200);

    const rows = result.body as Array<Record<string, unknown>>;
    expect(rows.length).toBeGreaterThanOrEqual(1);
    // All returned rows must belong to the authenticated user
    for (const row of rows) {
      expect(row.user_id).toBe(endUserSub);
    }
  });

  /* ================================================================ */
  /*  7. Anon key cannot see notes                                     */
  /* ================================================================ */
  test("7 - anon key cannot see notes", async () => {
    test.setTimeout(60_000);
    skipIfUnavailable();

    const result = await httpsRequest(
      `${projectApiBase}/rest/v1/user_notes?select=*`,
      {
        method: "GET",
        headers: {
          Authorization: `Bearer ${anonJWT}`,
          apikey: anonJWT,
        },
      },
    );

    console.log(
      `[test7] anon SELECT status=${result.status} body=${JSON.stringify(result.body)}`,
    );
    expect(result.status).toBe(200);

    // RLS blocks: auth.uid() is null for anon → no rows
    const rows = result.body as Array<Record<string, unknown>>;
    expect(rows.length).toBe(0);
  });

  /* ================================================================ */
  /*  8. Service_role key sees all notes (RLS bypass)                   */
  /* ================================================================ */
  test("8 - service_role key sees all notes (RLS bypass)", async () => {
    test.setTimeout(60_000);
    skipIfUnavailable();

    const result = await httpsRequest(
      `${projectApiBase}/rest/v1/user_notes?select=*`,
      {
        method: "GET",
        headers: {
          Authorization: `Bearer ${serviceRoleJWT}`,
          apikey: serviceRoleJWT,
        },
      },
    );

    console.log(
      `[test8] service_role SELECT status=${result.status} body=${JSON.stringify(result.body)}`,
    );
    expect(result.status).toBe(200);

    const rows = result.body as Array<Record<string, unknown>>;
    expect(rows.length).toBeGreaterThanOrEqual(1);
  });

  /* ================================================================ */
  /*  9. auth.uid() matches JWT sub claim                              */
  /* ================================================================ */
  test("9 - auth.uid() matches JWT sub claim", async () => {
    test.setTimeout(60_000);
    skipIfUnavailable();

    // Insert another note and verify user_id == sub
    const insertResult = await httpsRequest(
      `${projectApiBase}/rest/v1/user_notes`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${endUserAccessToken}`,
          Prefer: "return=representation",
          apikey: anonJWT,
        },
        body: JSON.stringify({ content: "auth.uid() verification" }),
      },
    );

    console.log(
      `[test9] INSERT status=${insertResult.status} body=${JSON.stringify(insertResult.body)}`,
    );
    expect(insertResult.status).toBe(201);

    const rows = insertResult.body as Array<Record<string, unknown>>;
    expect(rows.length).toBe(1);

    // user_id is set by DEFAULT auth.uid() — must match JWT sub
    const claims = decodeJWTPayload(endUserAccessToken);
    expect(rows[0].user_id).toBe(claims.sub);
  });
});

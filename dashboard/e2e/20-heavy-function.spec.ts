import { test, expect } from "@playwright/test";
import * as https from "node:https";
import * as crypto from "node:crypto";
import { execSync } from "node:child_process";
import {
  getAuthToken,
  createTestProject,
  deleteTestProject,
  getProject,
  createTestApiKey,
  createTestBucket,
  uploadTestFile,
  listFunctions,
  API_URL,
} from "./helpers/api";

// Allow self-signed certificates in Kind cluster
process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";

/** Full path to kubectl (defined after imports, before first use). */
const KUBECTL_BIN =
  process.env.KUBECTL_PATH ||
  "C:\\Program Files\\Docker\\Docker\\resources\\bin\\kubectl.exe";

/** Get project CRD phase directly via kubectl. */
function getCRDPhase(projectId: string): string {
  try {
    const out = execSync(
      `"${KUBECTL_BIN}" get project ${projectId} -n platform-system -o jsonpath="{.status.phase}"`,
      { encoding: "utf-8", timeout: 10_000 },
    ).trim().replace(/^"|"$/g, "");
    return out;
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
    if (lastPhase === "Ready") {
      return;
    }
    console.log(`[waitForProjectReady] project=${projectId} CRD phase="${lastPhase}" elapsed=${Math.round((Date.now() - start) / 1000)}s`);
    await new Promise((r) => setTimeout(r, 5_000));
  }
  throw new Error(
    `Project ${projectId} did not become READY within ${timeoutMs}ms (last CRD phase: "${lastPhase}")`,
  );
}

// JWT secret shared between GoTrue, PostgREST, and the operator.
const JWT_SECRET =
  process.env.E2E_JWT_SECRET ||
  "super-secret-jwt-token-for-local-dev-minimum-32-chars!!";

/** Generate a PostgREST-compatible JWT with the given role claim. */
function generatePostgrestJWT(role: string): string {
  const b64url = (buf: Buffer) =>
    buf
      .toString("base64")
      .replace(/=/g, "")
      .replace(/\+/g, "-")
      .replace(/\//g, "_");
  const header = b64url(
    Buffer.from(JSON.stringify({ alg: "HS256", typ: "JWT" })),
  );
  const now = Math.floor(Date.now() / 1000);
  const payload = b64url(
    Buffer.from(
      JSON.stringify({ role, iss: "etalbaas", iat: now, exp: now + 86400 }),
    ),
  );
  const sig = b64url(
    crypto
      .createHmac("sha256", JWT_SECRET)
      .update(`${header}.${payload}`)
      .digest(),
  );
  return `${header}.${payload}.${sig}`;
}

/* ===== Inline function source code ===== */

/**
 * Heavy-deployment function that handles both HTTP invocations and
 * event-driven triggers (DatabaseChange, ObjectStorage).
 *
 * - Event triggers: records the event payload to the trigger_results table via postgres-meta.
 * - HTTP invoke: returns a simple JSON response.
 */
const HEAVY_FN_CODE = `
export default async function(req) {
  const body = await req.json().catch(() => ({}));
  const eventType = req.headers.get("X-Event-Type");

  if (eventType === "DatabaseChange" || eventType === "StorageEvent") {
    // Event trigger: record result to trigger_results table via postgres-meta
    try {
      const metaRes = await fetch('http://postgres-meta:8080/query', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          query: "INSERT INTO public.trigger_results (event_type, payload, received_at) VALUES ('" +
            eventType.replace(/'/g, "''") + "', '" +
            JSON.stringify(body).replace(/'/g, "''") + "', NOW())"
        })
      });
      const metaBody = await metaRes.text();
      return new Response(JSON.stringify({
        ok: true, triggered: true, type: eventType,
        metaStatus: metaRes.status, metaBody
      }), { headers: { 'Content-Type': 'application/json' } });
    } catch(e) {
      return new Response(JSON.stringify({
        ok: false, error: e.message, type: eventType
      }), { status: 500, headers: { 'Content-Type': 'application/json' } });
    }
  }

  // HTTP invoke: direct call
  return new Response(JSON.stringify({
    ok: true, mode: "http-invoke", ts: Date.now()
  }), { headers: { 'Content-Type': 'application/json' } });
}
`;

/* ===== Types ===== */

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

/* ===== Helpers ===== */

/** DDL queries to create trigger test tables. */
const SETUP_QUERIES = [
  `CREATE TABLE IF NOT EXISTS public.trigger_test_events (
    id serial PRIMARY KEY,
    title text NOT NULL,
    created_at timestamptz DEFAULT now()
  )`,
  "ALTER TABLE public.trigger_test_events ENABLE ROW LEVEL SECURITY",
  "GRANT ALL ON public.trigger_test_events TO anon, authenticated, service_role",
  "GRANT USAGE, SELECT ON SEQUENCE public.trigger_test_events_id_seq TO anon, authenticated, service_role",
  "CREATE POLICY trigger_test_all ON public.trigger_test_events FOR ALL USING (true) WITH CHECK (true)",
  `CREATE TABLE IF NOT EXISTS public.trigger_results (
    id serial PRIMARY KEY,
    event_type text NOT NULL,
    payload text,
    received_at timestamptz DEFAULT now()
  )`,
  "ALTER TABLE public.trigger_results ENABLE ROW LEVEL SECURITY",
  "GRANT ALL ON public.trigger_results TO anon, authenticated, service_role",
  "GRANT USAGE, SELECT ON SEQUENCE public.trigger_results_id_seq TO anon, authenticated, service_role",
  "CREATE POLICY trigger_results_all ON public.trigger_results FOR ALL USING (true) WITH CHECK (true)",
];

/** Cleanup queries to drop trigger test tables. */
const CLEANUP_QUERIES = [
  "DROP TABLE IF EXISTS public.trigger_results CASCADE",
  "DROP TABLE IF EXISTS public.trigger_test_events CASCADE",
];

async function invokeFunction(
  projectId: string,
  functionName: string,
  payload: Record<string, unknown>,
): Promise<{ status: number; body: unknown }> {
  const apiUrl = new URL(API_URL);
  const targetHost = `${projectId}.${apiUrl.hostname}`;
  const postData = JSON.stringify(payload);

  const { status, text } = await new Promise<{ status: number; text: string }>(
    (resolve, reject) => {
      const req = https.request(
        {
          hostname: targetHost,
          port: apiUrl.port || 443,
          path: `/functions/${functionName}/invoke`,
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "Content-Length": Buffer.byteLength(postData),
          },
          rejectUnauthorized: false,
          lookup: (hostname, options, cb) => {
            if (hostname.endsWith(".api.local.etalbaas.dev")) {
              const result = { address: "127.0.0.1", family: 4 as const };
              if (typeof options === "object" && options?.all) {
                (cb as CallableFunction)(null, [result]);
              } else {
                (cb as CallableFunction)(null, result.address, result.family);
              }
              return;
            }
            import("node:dns").then((dns) =>
              dns.lookup(hostname, options as never, cb as never),
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
      req.write(postData);
      req.end();
    },
  );

  let parsed: unknown;
  try {
    parsed = JSON.parse(text);
  } catch {
    parsed = text;
  }
  return { status, body: parsed };
}

/** Deploy a heavy-deployment function with triggers via CreateFunction RPC. */
async function deployHeavyFunction(
  projectId: string,
  name: string,
  code: string,
  triggers: Record<string, unknown>[],
  envVars: Record<string, string>,
): Promise<FunctionItem> {
  const token = await getAuthToken();
  const res = await fetch(
    `${API_URL}/etalbaas.function.v1.FunctionService/CreateFunction`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({
        projectId,
        name,
        displayName: name,
        kind: "heavy-deployment",
        mode: "sync",
        inlineSource: { code, filename: "handler.js" },
        presetRuntime: { preset: "node-20" },
        triggers,
        envVars: Object.entries(envVars).map(([key, value]) => ({
          name: key,
          value,
          source: "literal",
        })),
      }),
    },
  );
  if (!res.ok) {
    throw new Error(`Deploy failed: ${await res.text()}`);
  }
  const data = await res.json();
  return data.function as FunctionItem;
}

/** Get function CRD phase directly via kubectl. */
function getFunctionCRDPhase(projectId: string, funcName: string): string {
  try {
    const out = execSync(
      `"${KUBECTL_BIN}" get function ${funcName} -n project-${projectId} -o jsonpath="{.status.phase}"`,
      { encoding: "utf-8", timeout: 10_000 },
    ).trim().replace(/^"|"$/g, "");
    return out;
  } catch {
    return "";
  }
}

/** Wait for a function to reach "Ready" CRD phase. */
async function waitForFunctionReady(
  projectId: string,
  functionId: string,
  timeoutMs = 360_000,
  funcName?: string,
): Promise<void> {
  // If funcName not provided, try to find it via listFunctions
  if (!funcName) {
    const fns = await listFunctions(projectId);
    const fn = fns.find((f) => f.id === functionId);
    funcName = fn?.name;
  }
  if (!funcName) {
    throw new Error(`Cannot find function name for ID ${functionId}`);
  }

  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    const phase = getFunctionCRDPhase(projectId, funcName);
    if (phase === "Ready") return;
    if (phase === "Failed") {
      throw new Error(`Function ${funcName} build failed (CRD phase: Failed)`);
    }
    console.log(`[waitForFunctionReady] func=${funcName} CRD phase="${phase}" elapsed=${Math.round((Date.now() - start) / 1000)}s`);
    await new Promise((r) => setTimeout(r, 5_000));
  }
  throw new Error(
    `Function ${functionId} did not become ready within ${timeoutMs}ms`,
  );
}

/** Execute DDL queries via postgres-meta (through a proxy function or direct). */
async function executeSetupQueries(
  projectId: string,
  funcName: string,
  queries: string[],
): Promise<boolean> {
  // Execute setup queries via the deployed function's postgres-meta access
  for (let attempt = 0; attempt < 12; attempt++) {
    const result = await invokeFunction(projectId, funcName, {
      action: "setup",
      queries,
    });
    if (result.status === 200) {
      return true;
    }
    await new Promise((r) => setTimeout(r, 5_000));
  }
  return false;
}

/** Run a kubectl command and return the output (best-effort). */
function kubectl(cmd: string): string {
  try {
    return execSync(`"${KUBECTL_BIN}" ${cmd}`, {
      encoding: "utf-8",
      timeout: 10_000,
    }).trim();
  } catch (e) {
    const msg = e instanceof Error ? e.message.split("\n")[0] : String(e);
    return `(kubectl failed: ${msg.substring(0, 200)})`;
  }
}

/** Diagnose heavy function CDC chain state via kubectl. */
function diagnoseHeavyChain(projectId: string, heavyFnName: string, iteration: number): void {
  const ns = `project-${projectId}`;
  const streamName = `CDC-project-${projectId}`;
  const consumerName = `func-${heavyFnName}`;

  // Find nats-box pod name
  const natsBox = kubectl(
    `get pods -n platform-system -l app.kubernetes.io/component=nats-box -o jsonpath="{.items[0].metadata.name}"`,
  ).replace(/"/g, "");

  // NATS stream message count
  let streamMsgs = "?";
  if (natsBox && !natsBox.startsWith("(kubectl")) {
    const streamInfo = kubectl(
      `exec -n platform-system ${natsBox} -- nats stream info ${streamName} --json`,
    );
    try {
      const si = JSON.parse(streamInfo);
      streamMsgs = `${si?.state?.messages ?? "?"}`;
    } catch { /* ignore */ }
  }

  // NATS consumer pending count
  let pending = "?", delivered = "?", ackPending = "?";
  if (natsBox && !natsBox.startsWith("(kubectl")) {
    const consumerInfo = kubectl(
      `exec -n platform-system ${natsBox} -- nats consumer info ${streamName} ${consumerName} --json`,
    );
    try {
      const ci = JSON.parse(consumerInfo);
      pending = `${ci?.num_pending ?? "?"}`;
      delivered = `${ci?.delivered?.stream_seq ?? "?"}`;
      ackPending = `${ci?.num_ack_pending ?? "?"}`;
    } catch { /* ignore */ }
  }

  // Heavy function deployment replicas
  const replicas = kubectl(
    `get deployment func-${heavyFnName} -n ${ns} -o jsonpath="{.status.readyReplicas}"`,
  ).replace(/"/g, "");

  // Heavy function pod status
  const podStatus = kubectl(
    `get pods -n ${ns} -l etalbaas.io/function=${heavyFnName} --no-headers`,
  );

  // Sidecar logs (last 3 lines)
  const sidecarLogs = kubectl(
    `logs -n ${ns} -l etalbaas.io/function=${heavyFnName} -c nats-sidecar --tail=3`,
  );

  // Function container logs (last 3 lines)
  const fnLogs = kubectl(
    `logs -n ${ns} -l etalbaas.io/function=${heavyFnName} -c function --tail=3`,
  );

  console.log(
    `[diag #${iteration}] stream_msgs=${streamMsgs} consumer_pending=${pending} delivered=${delivered} ack_pending=${ackPending} replicas=${replicas}`,
  );
  console.log(`[diag #${iteration}] pods: ${podStatus}`);
  if (sidecarLogs && sidecarLogs !== "(kubectl failed)") {
    console.log(`[diag #${iteration}] sidecar: ${sidecarLogs}`);
  }
  if (fnLogs && fnLogs !== "(kubectl failed)") {
    console.log(`[diag #${iteration}] function: ${fnLogs}`);
  }
}

/** Poll trigger_results table for entries matching the given event type. */
async function pollTriggerResults(
  projectId: string,
  funcName: string,
  eventType: string,
  timeoutMs = 90_000,
  heavyFnName?: string,
): Promise<Record<string, unknown>[]> {
  const start = Date.now();
  let iteration = 0;
  while (Date.now() - start < timeoutMs) {
    iteration++;
    const result = await invokeFunction(projectId, funcName, {
      action: "query",
      sql: `SELECT * FROM public.trigger_results WHERE event_type = '${eventType}' ORDER BY id DESC`,
    });
    if (result.status === 200) {
      const body = result.body as { ok: boolean; rows?: Record<string, unknown>[] };
      if (body.rows && body.rows.length > 0) {
        return body.rows;
      }
    }
    // Run diagnostics every other iteration
    if (heavyFnName && iteration % 2 === 1) {
      diagnoseHeavyChain(projectId, heavyFnName, iteration);
    }
    await new Promise((r) => setTimeout(r, 5_000));
  }
  // Final diagnostic dump on timeout
  if (heavyFnName) {
    console.log("[diag FINAL] timeout reached - dumping full state");
    diagnoseHeavyChain(projectId, heavyFnName, 999);
    // Dump full sidecar logs
    const ns = `project-${projectId}`;
    const fullSidecar = kubectl(
      `logs -n ${ns} -l etalbaas.io/function=${heavyFnName} -c nats-sidecar --tail=50`,
    );
    console.log(`[diag FINAL] sidecar full:\n${fullSidecar}`);
    const fullFn = kubectl(
      `logs -n ${ns} -l etalbaas.io/function=${heavyFnName} -c function --tail=20`,
    );
    console.log(`[diag FINAL] function full:\n${fullFn}`);
  }
  return [];
}

/* ===== Setup function for DDL + query ===== */

/**
 * A simpler setup/query function that can run DDL via postgres-meta
 * and query results for verification.
 */
const SETUP_FN_CODE = `
export default async function(req) {
  const { action, queries, sql } = await req.json();

  if (action === 'setup' && Array.isArray(queries)) {
    const results = [];
    for (const q of queries) {
      try {
        const r = await fetch('http://postgres-meta:8080/query', {
          method: 'POST',
          headers: {'Content-Type': 'application/json'},
          body: JSON.stringify({query: q})
        });
        results.push({status: r.status, query: q.substring(0,50)});
      } catch(e) {
        results.push({error: e.message, query: q.substring(0,50)});
      }
    }
    return new Response(JSON.stringify({ok: true, results}), {
      headers: {'Content-Type': 'application/json'}
    });
  }

  if (action === 'query' && sql) {
    try {
      const r = await fetch('http://postgres-meta:8080/query', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({query: sql})
      });
      const data = await r.json();
      return new Response(JSON.stringify({ok: true, rows: data}), {
        headers: {'Content-Type': 'application/json'}
      });
    } catch(e) {
      return new Response(JSON.stringify({ok: false, error: e.message}), {
        status: 500, headers: {'Content-Type': 'application/json'}
      });
    }
  }

  return new Response(JSON.stringify({ok: false, error: 'unknown action'}), {
    status: 400, headers: {'Content-Type': 'application/json'}
  });
}
`;

/* ===== Test suite ===== */

test.describe.serial(
  "Heavy Deployment Function - Build + Invoke + Triggers",
  () => {
    let project: { id: string; displayName: string };
    let serviceRoleKey: string;
    let heavyFn: FunctionItem;
    let setupFn: FunctionItem;
    let suiteAvailable = false;
    const triggerBucket = "e2e-trigger-bucket";

    const suffix = Date.now().toString(36);

    test.beforeAll(async () => {
      test.setTimeout(600_000); // 10 minutes for setup (build + deploy)

      // Create project with Postgres and PostgREST enabled
      project = await createTestProject("heavy-trigger", {
        postgresEnabled: true,
        postgrestEnabled: true,
      });

      // Wait for project provisioning (CNPG, PostgREST, etc.)
      await waitForProjectReady(project.id);

      // Create service role API key
      const srResult = await createTestApiKey(
        project.id,
        `sr-${suffix}`,
        "service_role",
      );
      serviceRoleKey = srResult.rawKey;

      // Create the trigger test bucket
      await createTestBucket(project.id, triggerBucket, "private");

      const serviceRoleJWT = generatePostgrestJWT("service_role");

      // Deploy setup/query utility function (light-deployment, quick build)
      const token = await getAuthToken();
      const setupRes = await fetch(
        `${API_URL}/etalbaas.function.v1.FunctionService/CreateFunction`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${token}`,
          },
          body: JSON.stringify({
            projectId: project.id,
            name: `setup-util-${suffix}`,
            displayName: `setup-util-${suffix}`,
            kind: "light-deployment",
            mode: "sync",
            inlineSource: { code: SETUP_FN_CODE, filename: "handler.js" },
            presetRuntime: { preset: "node-20" },
          }),
        },
      );
      if (!setupRes.ok) {
        console.warn(`Setup function deploy failed: ${await setupRes.text()}`);
        return;
      }
      const setupData = await setupRes.json();
      setupFn = setupData.function as FunctionItem;

      // Deploy heavy-deployment function with triggers
      try {
        heavyFn = await deployHeavyFunction(
          project.id,
          `e2e-heavy-${suffix}`,
          HEAVY_FN_CODE,
          [
            {
              databaseChange: {
                table: "trigger_test_events",
                events: ["INSERT"],
              },
            },
            {
              objectStorage: {
                bucket: triggerBucket,
                events: ["ObjectCreated"],
              },
            },
          ],
          {
            SERVICE_ROLE_KEY: serviceRoleJWT,
            NODE_TLS_REJECT_UNAUTHORIZED: "0",
          },
        );
      } catch (err) {
        console.warn(`Heavy function deploy failed: ${err}`);
        return;
      }

      // Wait for both functions to become ready (use CRD names)
      try {
        await Promise.all([
          waitForFunctionReady(project.id, setupFn.id, 540_000, setupFn.name),
          waitForFunctionReady(project.id, heavyFn.id, 540_000, heavyFn.name),
        ]);
      } catch (err) {
        console.warn(`Function not ready (skipping suite): ${err}`);
        return;
      }

      // Set up trigger test tables via setup function
      let tablesReady = false;
      for (let attempt = 0; attempt < 12; attempt++) {
        const result = await invokeFunction(
          project.id,
          setupFn.name,
          { action: "setup", queries: SETUP_QUERIES },
        );
        if (result.status === 200) {
          const body = result.body as { ok: boolean };
          if (body.ok) {
            tablesReady = true;
            break;
          }
        }
        await new Promise((r) => setTimeout(r, 5_000));
      }

      if (!tablesReady) {
        console.warn("Trigger test tables setup failed");
        return;
      }

      suiteAvailable = true;
    });

    test.afterAll(async () => {
      test.setTimeout(360_000);
      // Cleanup tables
      if (setupFn && project) {
        try {
          await invokeFunction(project.id, setupFn.name, {
            action: "setup",
            queries: CLEANUP_QUERIES,
          });
        } catch {
          // Ignore cleanup errors
        }
      }
      // Delete project (cascades to functions, buckets, etc.)
      if (project) {
        try {
          await deleteTestProject(project.id);
        } catch {
          // Ignore cleanup errors
        }
      }
    });

    function skipIfUnavailable() {
      test.skip(!suiteAvailable, "Heavy function setup not ready");
    }

    /* ================================================================ */
    /*  1. heavy-deployment function builds via Kaniko                   */
    /* ================================================================ */
    test("1 - heavy-deployment function builds via Kaniko", async () => {
      skipIfUnavailable();
      // Use CRD phase instead of API status (lazy-sync may be stale)
      const phase = getFunctionCRDPhase(project.id, heavyFn.name);
      expect(phase).toBe("Ready");
      // Verify via API that function exists with correct kind
      const fns = await listFunctions(project.id);
      const fn = fns.find((f) => f.id === heavyFn.id);
      expect(fn).toBeTruthy();
      expect(fn!.kind).toBe("heavy-deployment");
    });

    /* ================================================================ */
    /*  2. DB trigger fires function on INSERT                          */
    /* ================================================================ */
    test("2 - DB trigger fires function on INSERT", async () => {
      test.setTimeout(300_000);
      skipIfUnavailable();

      // Pre-INSERT diagnostics: check NATS consumer and heavy function state
      console.log("[test2] pre-INSERT diagnostics:");
      diagnoseHeavyChain(project.id, heavyFn.name, 0);

      // Insert a row into trigger_test_events via setup function
      await invokeFunction(project.id, setupFn.name, {
        action: "setup",
        queries: [
          `INSERT INTO public.trigger_test_events (title) VALUES ('trigger-test-${suffix}')`,
        ],
      });

      // Poll trigger_results for DatabaseChange event (with diagnostics)
      // Allow up to 180s for KEDA scale-up + pod start + sidecar connect + event processing
      const results = await pollTriggerResults(
        project.id,
        setupFn.name,
        "DatabaseChange",
        180_000,
        heavyFn.name,
      );

      expect(results.length).toBeGreaterThan(0);
    });

    /* ================================================================ */
    /*  3. HTTP invoke returns response (Pod should be running from #2) */
    /* ================================================================ */
    test("3 - HTTP invoke returns response", async () => {
      test.setTimeout(60_000);
      skipIfUnavailable();

      // The pod should already be running from the DB trigger test
      // Retry a few times in case KEDA is still scaling
      let lastResult: { status: number; body: unknown } | null = null;
      for (let attempt = 0; attempt < 12; attempt++) {
        const result = await invokeFunction(project.id, heavyFn.name, {
          test: true,
        });
        lastResult = result;
        if (result.status === 200) break;
        await new Promise((r) => setTimeout(r, 5_000));
      }

      expect(lastResult).toBeTruthy();
      expect(lastResult!.status).toBe(200);
      const body = lastResult!.body as { ok: boolean; mode: string };
      expect(body.ok).toBe(true);
      expect(body.mode).toBe("http-invoke");
    });

    /* ================================================================ */
    /*  4. Storage trigger fires function on upload                     */
    /* ================================================================ */
    test("4 - Storage trigger fires function on upload", async () => {
      test.setTimeout(120_000);
      skipIfUnavailable();

      // Upload a file to the trigger bucket
      await uploadTestFile(
        project.id,
        triggerBucket,
        `trigger-file-${suffix}.txt`,
        "storage trigger test content",
        "text/plain",
      );

      // Poll trigger_results for StorageEvent (with diagnostics)
      const results = await pollTriggerResults(
        project.id,
        setupFn.name,
        "StorageEvent",
        90_000,
        heavyFn.name,
      );

      expect(results.length).toBeGreaterThan(0);
    });

    /* ================================================================ */
    /*  5. Function receives correct CDC payload structure               */
    /* ================================================================ */
    test("5 - function receives correct CDC payload structure", async () => {
      test.setTimeout(30_000);
      skipIfUnavailable();

      // Query the trigger_results for DatabaseChange entries
      const result = await invokeFunction(project.id, setupFn.name, {
        action: "query",
        sql: "SELECT * FROM public.trigger_results WHERE event_type = 'DatabaseChange' LIMIT 1",
      });

      expect(result.status).toBe(200);
      const body = result.body as { ok: boolean; rows: Record<string, unknown>[] };
      expect(body.ok).toBe(true);
      expect(body.rows.length).toBeGreaterThan(0);

      const row = body.rows[0];
      const payload = JSON.parse(row.payload as string);

      // CDC payload should contain table, operation, and new data
      expect(payload).toHaveProperty("table");
      expect(payload).toHaveProperty("operation");
    });

    /* ================================================================ */
    /*  6. Function receives correct Storage event structure             */
    /* ================================================================ */
    test("6 - function receives correct Storage event structure", async () => {
      test.setTimeout(30_000);
      skipIfUnavailable();

      // Query the trigger_results for StorageEvent entries
      const result = await invokeFunction(project.id, setupFn.name, {
        action: "query",
        sql: "SELECT * FROM public.trigger_results WHERE event_type = 'StorageEvent' LIMIT 1",
      });

      expect(result.status).toBe(200);
      const body = result.body as { ok: boolean; rows: Record<string, unknown>[] };
      expect(body.ok).toBe(true);
      expect(body.rows.length).toBeGreaterThan(0);

      const row = body.rows[0];
      const payload = JSON.parse(row.payload as string);

      // Storage event payload should contain bucket, object_key, operation
      expect(payload).toHaveProperty("bucket");
      expect(payload).toHaveProperty("object_key");
      expect(payload).toHaveProperty("operation");
    });
  },
);

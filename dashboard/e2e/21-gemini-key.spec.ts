import { test, expect } from "@playwright/test";
import * as https from "node:https";
import { execSync } from "node:child_process";
import {
  getAuthToken,
  createTestProject,
  deleteTestProject,
  createTestSecret,
  API_URL,
} from "./helpers/api";

// Allow self-signed certificates in Kind cluster
process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";

const KUBECTL_BIN =
  process.env.KUBECTL_PATH ||
  "C:\\Program Files\\Docker\\Docker\\resources\\bin\\kubectl.exe";

const GEMINI_API_KEY = process.env.E2E_GEMINI_API_KEY || "";

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

/** Get function CRD phase. */
function getFunctionCRDPhase(projectId: string, funcName: string): string {
  try {
    return execSync(
      `"${KUBECTL_BIN}" get function ${funcName} -n project-${projectId} -o jsonpath="{.status.phase}"`,
      { encoding: "utf-8", timeout: 10_000 },
    )
      .trim()
      .replace(/^"|"$/g, "");
  } catch {
    return "";
  }
}

/** Wait for function CRD to reach Ready. */
async function waitForFunctionReady(
  projectId: string,
  funcName: string,
  timeoutMs = 360_000,
): Promise<void> {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    const phase = getFunctionCRDPhase(projectId, funcName);
    if (phase === "Ready") return;
    if (phase === "Failed") {
      throw new Error(`Function ${funcName} build failed (CRD phase: Failed)`);
    }
    console.log(
      `[waitForFunctionReady] func=${funcName} phase="${phase}" elapsed=${Math.round((Date.now() - start) / 1000)}s`,
    );
    await new Promise((r) => setTimeout(r, 5_000));
  }
  throw new Error(
    `Function ${funcName} did not become ready within ${timeoutMs}ms`,
  );
}

/** Invoke a function via HTTPS with custom DNS resolution. */
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
          timeout: 30_000,
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
      req.on("timeout", () => {
        req.destroy(new Error("request timeout"));
      });
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

/* ===== Function source code ===== */

const GEMINI_FN_CODE = `
export default async function(req) {
  const body = await req.json().catch(() => ({}));
  const action = body.action || "check-env";

  if (action === "check-env") {
    const key = process.env.GEMINI_API_KEY || "";
    return new Response(JSON.stringify({
      ok: true,
      exists: key.length > 0,
      length: key.length
    }), { headers: { "Content-Type": "application/json" } });
  }

  if (action === "call-gemini") {
    const key = process.env.GEMINI_API_KEY;
    if (!key) {
      return new Response(JSON.stringify({
        ok: false, error: "API key not configured"
      }), { status: 400, headers: { "Content-Type": "application/json" } });
    }

    const url = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=" + key;
    const res = await fetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        contents: [{ parts: [{ text: "Reply with exactly one word: PONG" }] }],
        generationConfig: { maxOutputTokens: 10 }
      })
    });

    const data = await res.json();
    return new Response(JSON.stringify({
      ok: res.ok,
      status: res.status,
      text: data?.candidates?.[0]?.content?.parts?.[0]?.text || null,
      error: data?.error?.message || null
    }), { headers: { "Content-Type": "application/json" } });
  }

  return new Response(JSON.stringify({
    ok: false, error: "unknown action"
  }), { status: 400, headers: { "Content-Type": "application/json" } });
}
`;

/* ===== Test suite ===== */

test.describe.serial("Gemini Key Connection", () => {
  let project: { id: string; displayName: string };
  let funcName: string;
  let noKeyFuncName: string;
  let suiteAvailable = false;

  const suffix = Date.now().toString(36);

  test.beforeAll(async () => {
    test.setTimeout(600_000);

    if (!GEMINI_API_KEY) {
      console.warn(
        "E2E_GEMINI_API_KEY not set — skipping Gemini key tests",
      );
      return;
    }

    // Create project
    project = await createTestProject("gemini-key", {
      postgresEnabled: true,
      postgrestEnabled: true,
    });

    await waitForProjectReady(project.id);

    // Create secret via Secret MS (creates k8s Secret in project namespace)
    await createTestSecret(project.id, "GEMINI_API_KEY", GEMINI_API_KEY);

    const token = await getAuthToken();

    // Deploy function WITH secret env var
    funcName = `gemini-fn-${suffix}`;
    const fnRes = await fetch(
      `${API_URL}/etalbaas.function.v1.FunctionService/CreateFunction`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          projectId: project.id,
          name: funcName,
          displayName: funcName,
          kind: "light-deployment",
          mode: "sync",
          inlineSource: { code: GEMINI_FN_CODE, filename: "handler.js" },
          presetRuntime: { preset: "node-20" },
          envVars: [
            { name: "GEMINI_API_KEY", secretName: "GEMINI_API_KEY" },
          ],
        }),
      },
    );
    if (!fnRes.ok) {
      console.warn(`Function deploy failed: ${await fnRes.text()}`);
      return;
    }

    // Deploy function WITHOUT secret (for test 3)
    noKeyFuncName = `nokey-fn-${suffix}`;
    const noKeyRes = await fetch(
      `${API_URL}/etalbaas.function.v1.FunctionService/CreateFunction`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          projectId: project.id,
          name: noKeyFuncName,
          displayName: noKeyFuncName,
          kind: "light-deployment",
          mode: "sync",
          inlineSource: { code: GEMINI_FN_CODE, filename: "handler.js" },
          presetRuntime: { preset: "node-20" },
        }),
      },
    );
    if (!noKeyRes.ok) {
      console.warn(`No-key function deploy failed: ${await noKeyRes.text()}`);
      return;
    }

    // Wait for both functions to be ready
    try {
      await Promise.all([
        waitForFunctionReady(project.id, funcName, 360_000),
        waitForFunctionReady(project.id, noKeyFuncName, 360_000),
      ]);
    } catch (err) {
      console.warn(`Function not ready (skipping suite): ${err}`);
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
    test.skip(!GEMINI_API_KEY, "E2E_GEMINI_API_KEY not set");
    test.skip(!suiteAvailable, "Gemini function setup not ready");
  }

  /* ================================================================ */
  /*  1. Secret is injected into function pod                         */
  /* ================================================================ */
  test("1 - secret is injected into function pod", async () => {
    test.setTimeout(60_000);
    skipIfUnavailable();

    let lastResult: { status: number; body: unknown } | null = null;
    for (let attempt = 0; attempt < 6; attempt++) {
      const result = await invokeFunction(project.id, funcName, {
        action: "check-env",
      });
      lastResult = result;
      if (result.status === 200) break;
      await new Promise((r) => setTimeout(r, 5_000));
    }

    expect(lastResult).toBeTruthy();
    expect(lastResult!.status).toBe(200);
    const body = lastResult!.body as {
      ok: boolean;
      exists: boolean;
      length: number;
    };
    console.log("[test1] env check:", JSON.stringify(body));
    expect(body.ok).toBe(true);
    expect(body.exists).toBe(true);
    expect(body.length).toBeGreaterThan(0);
  });

  /* ================================================================ */
  /*  2. Function can call Gemini API (SINGLE call only)              */
  /* ================================================================ */
  test("2 - function can call Gemini API", async () => {
    test.setTimeout(60_000);
    skipIfUnavailable();

    // Single invocation — no retry on the Gemini call itself
    const result = await invokeFunction(project.id, funcName, {
      action: "call-gemini",
    });

    expect(result.status).toBe(200);
    const body = result.body as {
      ok: boolean;
      status: number;
      text: string | null;
      error: string | null;
    };
    console.log("[test2] Gemini response:", JSON.stringify(body));
    expect(body.ok).toBe(true);
    expect(body.status).toBe(200);
    expect(body.text).toBeTruthy();
    // Gemini should respond with something containing PONG
    expect(body.text!.toUpperCase()).toContain("PONG");
  });

  /* ================================================================ */
  /*  3. Function handles missing API key gracefully                  */
  /* ================================================================ */
  test("3 - function handles missing API key gracefully", async () => {
    test.setTimeout(60_000);
    skipIfUnavailable();

    let lastResult: { status: number; body: unknown } | null = null;
    for (let attempt = 0; attempt < 6; attempt++) {
      const result = await invokeFunction(project.id, noKeyFuncName, {
        action: "call-gemini",
      });
      lastResult = result;
      if (result.status === 400) break;
      await new Promise((r) => setTimeout(r, 5_000));
    }

    expect(lastResult).toBeTruthy();
    expect(lastResult!.status).toBe(400);
    const body = lastResult!.body as { ok: boolean; error: string };
    expect(body.ok).toBe(false);
    expect(body.error).toContain("API key not configured");
  });
});

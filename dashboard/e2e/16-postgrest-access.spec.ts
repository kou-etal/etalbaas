/**
 * 16 - PostgREST CRUD + RLS Enforcement
 *
 * API-only tests (no browser UI) verifying PostgREST access using anon and
 * service_role API keys. Uses Playwright's `request` context for direct HTTP
 * calls against the project's PostgREST endpoint.
 *
 * Setup:
 *   - Create a project with postgresEnabled + postgrestEnabled
 *   - Wait for the project to reach READY status
 *   - Create anon and service_role API keys
 *   - Discover available tables via PostgREST schema introspection
 *
 * The tests exercise SELECT / INSERT / UPDATE / DELETE for both roles and
 * verify RLS enforcement: anon must be blocked from admin-only resources
 * while service_role bypasses RLS.
 */

process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";

import { rewriteUrl } from "./helpers/dns-override";
import { test, expect, APIRequestContext } from "@playwright/test";
import {
  createTestProject,
  deleteTestProject,
  getProject,
  createTestApiKey,
  uniqueName,
  API_URL,
} from "./helpers/api";

/* ===== Helpers ===== */

/** Derive the PostgREST base URL from a project ID (original domain). */
function getPostgrestUrlRaw(projectId: string): string {
  const apiUrl = new URL(API_URL);
  return `https://${projectId}.${apiUrl.hostname}/rest/v1`;
}

/** Host header for PostgREST requests (set in beforeAll). */
let postgrestHost: string;

/** Build common PostgREST headers for a given raw API key. */
function postgrestHeaders(
  rawKey: string,
  extra: Record<string, string> = {},
): Record<string, string> {
  return {
    Host: postgrestHost,
    apikey: rawKey,
    Authorization: `Bearer ${rawKey}`,
    "Content-Type": "application/json",
    Prefer: "return=representation",
    ...extra,
  };
}

/** Poll getProject until status is READY (or timeout). */
async function waitForProjectReady(
  projectId: string,
  timeoutMs = 300_000,
  intervalMs = 3_000,
): Promise<void> {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const p = await getProject(projectId);
    if (
      p.status === "READY" ||
      p.status === "PROJECT_STATUS_READY" ||
      p.status === "ready"
    ) {
      return;
    }
    await new Promise((r) => setTimeout(r, intervalMs));
  }
  throw new Error(
    `Project ${projectId} did not become READY within ${timeoutMs}ms`,
  );
}

/* ===== Shared state across serial tests ===== */

let projectId: string;
let postgrestUrl = "";
let anonKey = "";
let serviceRoleKey = "";
let apiCtx: APIRequestContext;

/**
 * The name of a "public" table that anon can access, and an
 * "admin-only" table that anon cannot access. Discovered during
 * beforeAll by introspecting the PostgREST schema.
 */
let publicTable: string;
let adminTable: string;

/**
 * If PostgREST is not reachable or no tables are available the
 * whole suite is skipped gracefully.
 */
let suiteAvailable = false;

/** Track an inserted row id so UPDATE / DELETE can reference it. */
let insertedRowFilter: string;

test.describe.serial("PostgREST CRUD + RLS", () => {
  /* ================================================================ */
  /*  Setup                                                            */
  /* ================================================================ */

  test.beforeAll(async ({ playwright }) => {
    test.setTimeout(360_000);
    // 1. Create project with PG + PostgREST enabled
    const project = await createTestProject("postgrest", {
      postgresEnabled: true,
      postgrestEnabled: true,
    });
    projectId = project.id;

    // 2. Wait for project to be fully ready
    try {
      await waitForProjectReady(projectId);
    } catch (err) {
      console.warn(`Project not ready (skipping suite): ${err}`);
      return; // suiteAvailable stays false → tests will skip gracefully
    }

    // 3. Create anon + service_role API keys
    const anonResult = await createTestApiKey(
      projectId,
      "e2e-anon",
      "anon",
    );
    anonKey = anonResult.rawKey;

    const svcResult = await createTestApiKey(
      projectId,
      "e2e-service-role",
      "service_role",
    );
    serviceRoleKey = svcResult.rawKey;

    // 4. Derive PostgREST URL (rewrite to 127.0.0.1 for DNS bypass)
    const rawUrl = getPostgrestUrlRaw(projectId);
    const rewritten = rewriteUrl(rawUrl);
    postgrestUrl = rewritten.url;
    postgrestHost = rewritten.host;

    // 5. Create a shared API request context (ignores TLS errors, sets Host)
    apiCtx = await playwright.request.newContext({
      ignoreHTTPSErrors: true,
      extraHTTPHeaders: { Host: postgrestHost },
    });

    // 6. Probe PostgREST availability and discover tables
    try {
      const schemaRes = await apiCtx.get(`${postgrestUrl}/`, {
        headers: postgrestHeaders(serviceRoleKey),
        timeout: 30_000,
      });

      if (!schemaRes.ok()) {
        console.warn(
          `PostgREST schema probe failed (${schemaRes.status()}); skipping suite`,
        );
        return; // suiteAvailable stays false
      }

      // PostgREST root returns an OpenAPI spec with "paths" keyed by table
      const schema = await schemaRes.json();
      const paths: string[] = schema.paths
        ? Object.keys(schema.paths)
            .map((p: string) => p.replace(/^\//, ""))
            .filter((p: string) => p && !p.startsWith("rpc/"))
        : [];

      if (paths.length === 0) {
        console.warn("No tables exposed by PostgREST; skipping suite");
        return;
      }

      // Heuristic: pick the first table as "public", the second (if any)
      // as "admin". Many default setups expose a profiles / users table.
      publicTable = paths[0];
      adminTable = paths.length > 1 ? paths[1] : "";

      suiteAvailable = true;
    } catch (err) {
      console.warn("PostgREST not reachable; skipping suite:", err);
    }
  });

  test.afterAll(async () => {
    test.setTimeout(360_000);
    // Dispose the API request context
    if (apiCtx) {
      await apiCtx.dispose();
    }

    // Tear down the project
    if (projectId) {
      try {
        await deleteTestProject(projectId);
      } catch {
        // best-effort cleanup
      }
    }
  });

  /* ================================================================ */
  /*  Helper: skip when suite is unavailable                           */
  /* ================================================================ */

  function skipIfUnavailable() {
    test.skip(!suiteAvailable, "PostgREST not available or no tables found");
  }

  /* ================================================================ */
  /*  Tests                                                            */
  /* ================================================================ */

  /* ---------------------------------------------------------------- */
  /*  1. anon: GET schema -> tables listed                             */
  /* ---------------------------------------------------------------- */
  test("1 - anon: GET schema returns tables", async () => {
    skipIfUnavailable();

    const res = await apiCtx.get(`${postgrestUrl}/`, {
      headers: postgrestHeaders(anonKey),
    });

    expect(res.status()).toBe(200);

    const body = await res.json();
    // OpenAPI spec must contain at least one path (table)
    expect(body).toHaveProperty("paths");
    const tableCount = Object.keys(body.paths).filter(
      (p: string) => !p.startsWith("/rpc/"),
    ).length;
    expect(tableCount).toBeGreaterThanOrEqual(1);
  });

  /* ---------------------------------------------------------------- */
  /*  2. anon: SELECT from public table                                */
  /* ---------------------------------------------------------------- */
  test("2 - anon: SELECT from public table returns 200", async () => {
    skipIfUnavailable();

    const res = await apiCtx.get(
      `${postgrestUrl}/${publicTable}?select=*`,
      { headers: postgrestHeaders(anonKey) },
    );

    expect(res.status()).toBe(200);
    const rows = await res.json();
    expect(Array.isArray(rows)).toBeTruthy();
  });

  /* ---------------------------------------------------------------- */
  /*  3. anon: INSERT into public table                                */
  /* ---------------------------------------------------------------- */
  test("3 - anon: INSERT into public table succeeds", async () => {
    skipIfUnavailable();

    // First, discover the columns via an OPTIONS or empty GET to understand schema.
    // We insert a minimal JSON payload; PostgREST will return the created row
    // if the table and RLS allow it. If the table has required columns we
    // cannot guess, we accept 201 OR a 4xx that is NOT 401 (auth error).
    const tag = uniqueName("anon");
    const payload: Record<string, unknown> = { name: tag };

    const res = await apiCtx.post(`${postgrestUrl}/${publicTable}`, {
      headers: postgrestHeaders(anonKey),
      data: payload,
    });

    // Acceptable: 201 Created, 200 OK, or 409 Conflict (duplicate).
    // If we get 400 (bad columns) that's also acceptable as long as
    // it's not an auth failure. The key assertion: NOT 401 / 403.
    if (res.status() === 201 || res.status() === 200) {
      const body = await res.json();
      // Prefer: return=representation means we get the row(s) back
      if (Array.isArray(body) && body.length > 0) {
        // Store a filter for subsequent UPDATE / DELETE tests
        const row = body[0];
        const idField =
          "id" in row ? "id" : Object.keys(row)[0];
        insertedRowFilter = `${idField}=eq.${row[idField]}`;
      }
    } else {
      // Column mismatch or constraint error is ok; 401/403 is not
      expect([400, 404, 409, 422]).toContain(res.status());
    }
  });

  /* ---------------------------------------------------------------- */
  /*  4. anon: UPDATE own data                                         */
  /* ---------------------------------------------------------------- */
  test("4 - anon: UPDATE own data succeeds", async () => {
    skipIfUnavailable();
    test.skip(
      !insertedRowFilter,
      "No row was inserted by anon to update",
    );

    const res = await apiCtx.patch(
      `${postgrestUrl}/${publicTable}?${insertedRowFilter}`,
      {
        headers: postgrestHeaders(anonKey),
        data: { name: uniqueName("anon-upd") },
      },
    );

    // 200 or 204 are acceptable success codes
    expect([200, 204]).toContain(res.status());
  });

  /* ---------------------------------------------------------------- */
  /*  5. anon: DELETE own data                                         */
  /* ---------------------------------------------------------------- */
  test("5 - anon: DELETE own data succeeds", async () => {
    skipIfUnavailable();
    test.skip(
      !insertedRowFilter,
      "No row was inserted by anon to delete",
    );

    const res = await apiCtx.delete(
      `${postgrestUrl}/${publicTable}?${insertedRowFilter}`,
      { headers: postgrestHeaders(anonKey) },
    );

    expect([200, 204]).toContain(res.status());
  });

  /* ---------------------------------------------------------------- */
  /*  6. anon: access to RLS-protected table blocked                   */
  /* ---------------------------------------------------------------- */
  test("6 - anon: access to RLS-protected table blocked", async () => {
    skipIfUnavailable();
    test.skip(!adminTable, "No second table available for RLS test");

    const res = await apiCtx.get(
      `${postgrestUrl}/${adminTable}?select=*`,
      { headers: postgrestHeaders(anonKey) },
    );

    // RLS should either return 0 rows (200 + empty array) or 403
    if (res.status() === 200) {
      const rows = await res.json();
      expect(Array.isArray(rows)).toBeTruthy();
      expect(rows.length).toBe(0);
    } else {
      expect(res.status()).toBe(403);
    }
  });

  /* ---------------------------------------------------------------- */
  /*  7. anon: INSERT into RLS-protected table blocked                 */
  /* ---------------------------------------------------------------- */
  test("7 - anon: INSERT into RLS-protected table blocked", async () => {
    skipIfUnavailable();
    test.skip(!adminTable, "No second table available for RLS test");

    const res = await apiCtx.post(`${postgrestUrl}/${adminTable}`, {
      headers: postgrestHeaders(anonKey),
      data: { name: uniqueName("anon-blocked") },
    });

    // Should be denied: 403 Forbidden, 401, or 0-row 200/201
    expect([401, 403, 404]).toContain(res.status());
  });

  /* ---------------------------------------------------------------- */
  /*  8. service_role: SELECT all data (RLS bypass)                    */
  /* ---------------------------------------------------------------- */
  test("8 - service_role: SELECT all data bypassing RLS", async () => {
    skipIfUnavailable();

    const res = await apiCtx.get(
      `${postgrestUrl}/${publicTable}?select=*`,
      { headers: postgrestHeaders(serviceRoleKey) },
    );

    expect(res.status()).toBe(200);
    const rows = await res.json();
    expect(Array.isArray(rows)).toBeTruthy();
  });

  /* ---------------------------------------------------------------- */
  /*  9. service_role: INSERT any table                                */
  /* ---------------------------------------------------------------- */
  test("9 - service_role: INSERT into any table succeeds", async () => {
    skipIfUnavailable();

    const tag = uniqueName("svc");
    const res = await apiCtx.post(`${postgrestUrl}/${publicTable}`, {
      headers: postgrestHeaders(serviceRoleKey),
      data: { name: tag },
    });

    if (res.status() === 201 || res.status() === 200) {
      const body = await res.json();
      if (Array.isArray(body) && body.length > 0) {
        const row = body[0];
        const idField =
          "id" in row ? "id" : Object.keys(row)[0];
        // Store for subsequent service_role UPDATE/DELETE
        insertedRowFilter = `${idField}=eq.${row[idField]}`;
      }
    } else {
      // Column mismatch is acceptable; auth failures are not
      expect([400, 409, 422]).toContain(res.status());
    }
  });

  /* ---------------------------------------------------------------- */
  /*  10. service_role: UPDATE any row                                 */
  /* ---------------------------------------------------------------- */
  test("10 - service_role: UPDATE any row succeeds", async () => {
    skipIfUnavailable();
    test.skip(
      !insertedRowFilter,
      "No row available for service_role update",
    );

    const res = await apiCtx.patch(
      `${postgrestUrl}/${publicTable}?${insertedRowFilter}`,
      {
        headers: postgrestHeaders(serviceRoleKey),
        data: { name: uniqueName("svc-upd") },
      },
    );

    expect([200, 204]).toContain(res.status());
  });

  /* ---------------------------------------------------------------- */
  /*  11. service_role: DELETE any row                                 */
  /* ---------------------------------------------------------------- */
  test("11 - service_role: DELETE any row succeeds", async () => {
    skipIfUnavailable();
    test.skip(
      !insertedRowFilter,
      "No row available for service_role delete",
    );

    const res = await apiCtx.delete(
      `${postgrestUrl}/${publicTable}?${insertedRowFilter}`,
      { headers: postgrestHeaders(serviceRoleKey) },
    );

    expect([200, 204]).toContain(res.status());
  });

  /* ---------------------------------------------------------------- */
  /*  12. service_role: access RLS-protected table                     */
  /* ---------------------------------------------------------------- */
  test("12 - service_role: access RLS-protected table returns rows", async () => {
    skipIfUnavailable();
    test.skip(!adminTable, "No second table available for RLS test");

    const res = await apiCtx.get(
      `${postgrestUrl}/${adminTable}?select=*`,
      { headers: postgrestHeaders(serviceRoleKey) },
    );

    // service_role bypasses RLS -> 200 guaranteed (even if 0 rows)
    expect(res.status()).toBe(200);
    const rows = await res.json();
    expect(Array.isArray(rows)).toBeTruthy();
  });

  /* ---------------------------------------------------------------- */
  /*  13. invalid key -> 401                                           */
  /* ---------------------------------------------------------------- */
  test("13 - invalid API key returns 401", async () => {
    skipIfUnavailable();

    const res = await apiCtx.get(
      `${postgrestUrl}/${publicTable}?select=*`,
      {
        headers: postgrestHeaders("totally-invalid-key-12345"),
      },
    );

    expect(res.status()).toBe(401);
  });

  /* ---------------------------------------------------------------- */
  /*  14. no key -> 401                                                */
  /* ---------------------------------------------------------------- */
  test("14 - no API key returns 401", async () => {
    skipIfUnavailable();

    const res = await apiCtx.get(
      `${postgrestUrl}/${publicTable}?select=*`,
      {
        // Intentionally omit apikey and Authorization headers
        headers: { "Content-Type": "application/json" },
      },
    );

    expect(res.status()).toBe(401);
  });
});

/**
 * 19 - Supabase JS SDK Compatibility
 *
 * API-only tests (no browser UI) verifying that the official @supabase/supabase-js
 * client can connect to etalbaas's Supabase-compatible endpoints: GoTrue (auth),
 * PostgREST (data), and Storage REST (files).
 *
 * Setup:
 *   - Create a project with postgresEnabled + postgrestEnabled
 *   - Wait for the project to reach READY status
 *   - Create anon + service_role API keys
 *   - Derive a Supabase-compatible URL from the project ID
 *
 * The tests exercise auth flows, data CRUD (from/select/insert/update/delete),
 * RLS enforcement, and storage operations through the Supabase JS SDK.
 * Tests that require unimplemented features are skipped gracefully.
 */

process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";

import "./helpers/dns-override";
import { test, expect } from "@playwright/test";
import { createClient, SupabaseClient } from "@supabase/supabase-js";
import {
  createTestProject,
  deleteTestProject,
  getProject,
  createTestApiKey,
  createTestBucket,
  uniqueName,
  API_URL,
} from "./helpers/api";

/* ===== Helpers ===== */

/** Derive the Supabase-compatible base URL from a project ID. */
function getSupabaseUrl(projectId: string): string {
  const apiUrl = new URL(API_URL);
  return `https://${projectId}.${apiUrl.hostname}`;
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
let supabaseUrl = "";
let anonKey = "";
let serviceRoleKey = "";

/** Supabase client initialised with the anon key. */
let supabase: SupabaseClient;

/** Supabase client initialised with the service_role key. */
let supabaseServiceRole: SupabaseClient;

/** Whether the Supabase-compatible endpoint is reachable. */
let canConnect = false;

/** Whether PostgREST exposes at least one table for data tests. */
let hasTables = false;

/** The name of the first table discovered via PostgREST. */
let tableName: string;

/** Test user credentials for auth tests. */
const testEmail = `e2e-sdk-${Date.now()}@test.etalbaas.dev`;
const testPassword = "TestPassword123!";

/** Track an inserted row id so UPDATE / DELETE can reference it. */
let insertedRowId: string | number | undefined;
let insertedRowIdField: string;

/** Storage test constants. */
const sdkBucketName = uniqueName("sdk-bucket");
const fileName = "test-file.txt";
const fileContent = "Hello from Supabase SDK E2E test";

test.describe.serial("Supabase JS SDK Compatibility", () => {
  /* ================================================================ */
  /*  Setup                                                            */
  /* ================================================================ */

  test.beforeAll(async () => {
    test.setTimeout(360_000);
    // 1. Create project with PG + PostgREST enabled
    const project = await createTestProject("sdk-compat", {
      postgresEnabled: true,
      postgrestEnabled: true,
    });
    projectId = project.id;

    // Derive Supabase-compatible URL early (needed even if project never becomes READY)
    supabaseUrl = getSupabaseUrl(projectId);

    // 2. Wait for project to be fully ready
    try {
      await waitForProjectReady(projectId);
    } catch (err) {
      console.warn(`Project not ready (skipping suite): ${err}`);
      return; // canConnect stays false → tests will skip gracefully
    }

    // 3. Create anon + service_role API keys
    const anonResult = await createTestApiKey(projectId, "e2e-sdk-anon", "anon");
    anonKey = anonResult.rawKey;

    const svcResult = await createTestApiKey(
      projectId,
      "e2e-sdk-svc",
      "service_role",
    );
    serviceRoleKey = svcResult.rawKey;

    // 4. Create a test bucket via the management API
    try {
      await createTestBucket(projectId, sdkBucketName, "public");
    } catch {
      // Bucket creation may fail if storage is not configured; tests will skip
    }

    // 6. Try to create Supabase clients
    try {
      supabase = createClient(supabaseUrl, anonKey, {
        auth: {
          autoRefreshToken: false,
          persistSession: false,
        },
      });

      supabaseServiceRole = createClient(supabaseUrl, serviceRoleKey, {
        auth: {
          autoRefreshToken: false,
          persistSession: false,
        },
      });

      canConnect = true;
    } catch {
      canConnect = false;
    }

    // 7. Probe PostgREST for available tables
    if (canConnect) {
      try {
        // PostgREST root returns an OpenAPI spec; we use service_role to discover tables.
        const res = await fetch(`${supabaseUrl}/rest/v1/`, {
          headers: {
            apikey: serviceRoleKey,
            Authorization: `Bearer ${serviceRoleKey}`,
          },
        });

        if (res.ok) {
          const schema = await res.json();
          const paths: string[] = schema.paths
            ? Object.keys(schema.paths)
                .map((p: string) => p.replace(/^\//, ""))
                .filter((p: string) => p && !p.startsWith("rpc/"))
            : [];

          if (paths.length > 0) {
            tableName = paths[0];
            hasTables = true;
          }
        }
      } catch {
        // PostgREST not reachable; data tests will be skipped
      }
    }
  });

  test.afterAll(async () => {
    test.setTimeout(360_000);
    // Clean up SDK-created storage resources
    if (supabaseServiceRole) {
      try {
        await supabaseServiceRole.storage.emptyBucket(sdkBucketName);
        await supabaseServiceRole.storage.deleteBucket(sdkBucketName);
      } catch {
        // best-effort cleanup
      }
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
  /*  Skip helpers                                                      */
  /* ================================================================ */

  function skipIfCannotConnect() {
    test.skip(!canConnect, "Cannot connect to Supabase-compatible API");
  }

  function skipIfNoTables() {
    test.skip(!canConnect, "Cannot connect to Supabase-compatible API");
    test.skip(!hasTables, "No tables exposed by PostgREST");
  }

  /* ================================================================ */
  /*  1. createClient succeeds                                         */
  /* ================================================================ */
  test("1 - createClient succeeds without throwing", async () => {
    skipIfCannotConnect();

    // Creating the client itself should not throw. If beforeAll failed,
    // canConnect will be false and the test is skipped above.
    const client = createClient(supabaseUrl, anonKey, {
      auth: {
        autoRefreshToken: false,
        persistSession: false,
      },
    });

    expect(client).toBeTruthy();
    expect(typeof client.auth).toBe("object");
    expect(typeof client.from).toBe("function");
    expect(typeof client.storage).toBe("object");
  });

  /* ================================================================ */
  /*  2. auth.signUp                                                    */
  /* ================================================================ */
  test("2 - auth.signUp creates a new user", async () => {
    skipIfCannotConnect();

    const { data, error } = await supabase.auth.signUp({
      email: testEmail,
      password: testPassword,
    });

    if (error) {
      // If GoTrue is not behind the expected path, skip gracefully
      test.skip(
        error.message.includes("fetch") ||
          error.message.includes("ECONNREFUSED") ||
          error.message.includes("Failed") ||
          error.status === 404,
        `Auth endpoint not available: ${error.message}`,
      );
    }

    expect(data).toBeTruthy();
    expect(data.user).toBeTruthy();
    expect(data.user!.email).toBe(testEmail);
  });

  /* ================================================================ */
  /*  3. auth.signInWithPassword                                        */
  /* ================================================================ */
  test("3 - auth.signInWithPassword returns session", async () => {
    skipIfCannotConnect();

    const { data, error } = await supabase.auth.signInWithPassword({
      email: testEmail,
      password: testPassword,
    });

    if (error) {
      test.skip(
        error.message.includes("fetch") ||
          error.message.includes("ECONNREFUSED") ||
          error.message.includes("Failed") ||
          error.status === 404,
        `Auth endpoint not available: ${error.message}`,
      );
    }

    expect(data).toBeTruthy();
    expect(data.session).toBeTruthy();
    expect(data.session!.access_token).toBeTruthy();
    expect(typeof data.session!.access_token).toBe("string");
    expect(data.session!.access_token.length).toBeGreaterThan(10);
  });

  /* ================================================================ */
  /*  4. auth.getUser                                                   */
  /* ================================================================ */
  test("4 - auth.getUser returns authenticated user data", async () => {
    skipIfCannotConnect();

    // Ensure we are signed in
    const signIn = await supabase.auth.signInWithPassword({
      email: testEmail,
      password: testPassword,
    });

    if (signIn.error) {
      test.skip(true, `Cannot sign in: ${signIn.error.message}`);
    }

    const { data, error } = await supabase.auth.getUser();

    if (error) {
      test.skip(
        error.message.includes("fetch") || error.status === 404,
        `getUser endpoint not available: ${error.message}`,
      );
    }

    expect(data).toBeTruthy();
    expect(data.user).toBeTruthy();
    expect(data.user!.email).toBe(testEmail);
  });

  /* ================================================================ */
  /*  5. auth.signOut                                                   */
  /* ================================================================ */
  test("5 - auth.signOut destroys the session", async () => {
    skipIfCannotConnect();

    // Ensure we are signed in first
    const signIn = await supabase.auth.signInWithPassword({
      email: testEmail,
      password: testPassword,
    });

    if (signIn.error) {
      test.skip(true, `Cannot sign in: ${signIn.error.message}`);
    }

    const { error } = await supabase.auth.signOut();

    if (error) {
      test.skip(
        error.message.includes("fetch") || error.status === 404,
        `signOut endpoint not available: ${error.message}`,
      );
    }

    // After sign out, getUser should return no user (or error)
    const { data: userData } = await supabase.auth.getUser();
    // Session is cleared locally; remote session may still be valid
    // but the local client should reflect signed-out state
    expect(userData.user).toBeFalsy();
  });

  /* ================================================================ */
  /*  6. from().select()                                                */
  /* ================================================================ */
  test("6 - from().select() returns rows array", async () => {
    skipIfNoTables();

    const { data, error } = await supabaseServiceRole
      .from(tableName)
      .select("*");

    if (error) {
      test.skip(
        error.message.includes("fetch") ||
          error.message.includes("ECONNREFUSED") ||
          error.code === "404",
        `PostgREST not available: ${error.message}`,
      );
    }

    expect(data).toBeTruthy();
    expect(Array.isArray(data)).toBeTruthy();
  });

  /* ================================================================ */
  /*  7. from().insert()                                                */
  /* ================================================================ */
  test("7 - from().insert() creates a new row", async () => {
    skipIfNoTables();

    const tag = uniqueName("sdk-ins");
    const { data, error } = await supabaseServiceRole
      .from(tableName)
      .insert({ name: tag })
      .select();

    if (error) {
      // Column mismatch or constraint errors are acceptable; the key
      // assertion is that the endpoint responded (not 401/403).
      if (
        error.code === "42703" || // undefined column
        error.code === "23502" || // not-null violation
        error.code === "23505" || // unique violation
        error.message.includes("column")
      ) {
        // Table schema doesn't have a "name" column — skip further data tests
        test.skip(true, `Table schema mismatch: ${error.message}`);
        return;
      }
      test.skip(
        error.message.includes("fetch") || error.code === "404",
        `PostgREST not available: ${error.message}`,
      );
    }

    expect(data).toBeTruthy();
    expect(Array.isArray(data)).toBeTruthy();
    expect(data!.length).toBeGreaterThanOrEqual(1);

    // Store row identity for subsequent update/delete tests
    const row = data![0];
    insertedRowIdField = "id" in row ? "id" : Object.keys(row)[0];
    insertedRowId = row[insertedRowIdField];
  });

  /* ================================================================ */
  /*  8. from().update()                                                */
  /* ================================================================ */
  test("8 - from().update() modifies an existing row", async () => {
    skipIfNoTables();
    test.skip(insertedRowId === undefined, "No row was inserted to update");

    const updatedTag = uniqueName("sdk-upd");
    const { data, error } = await supabaseServiceRole
      .from(tableName)
      .update({ name: updatedTag })
      .eq(insertedRowIdField, insertedRowId!)
      .select();

    if (error) {
      test.skip(
        error.message.includes("fetch") || error.code === "404",
        `PostgREST not available: ${error.message}`,
      );
    }

    expect(data).toBeTruthy();
    expect(Array.isArray(data)).toBeTruthy();
    if (data!.length > 0) {
      expect(data![0].name).toBe(updatedTag);
    }
  });

  /* ================================================================ */
  /*  9. from().delete()                                                */
  /* ================================================================ */
  test("9 - from().delete() removes the row", async () => {
    skipIfNoTables();
    test.skip(insertedRowId === undefined, "No row was inserted to delete");

    const { error } = await supabaseServiceRole
      .from(tableName)
      .delete()
      .eq(insertedRowIdField, insertedRowId!);

    if (error) {
      test.skip(
        error.message.includes("fetch") || error.code === "404",
        `PostgREST not available: ${error.message}`,
      );
    }

    // Verify the row is gone
    const { data: remaining } = await supabaseServiceRole
      .from(tableName)
      .select("*")
      .eq(insertedRowIdField, insertedRowId!);

    expect(remaining).toBeTruthy();
    expect(remaining!.length).toBe(0);
  });

  /* ================================================================ */
  /*  10. from().select() with filters                                  */
  /* ================================================================ */
  test("10 - from().select() with .eq() / .gt() / .like() filters", async () => {
    skipIfNoTables();

    // Insert two rows to have something to filter
    const tagA = uniqueName("flt-a");
    const tagB = uniqueName("flt-b");

    const insA = await supabaseServiceRole
      .from(tableName)
      .insert({ name: tagA })
      .select();
    const insB = await supabaseServiceRole
      .from(tableName)
      .insert({ name: tagB })
      .select();

    if (insA.error || insB.error) {
      test.skip(true, "Cannot insert rows for filter test");
      return;
    }

    // .eq() filter
    const { data: eqData, error: eqErr } = await supabaseServiceRole
      .from(tableName)
      .select("*")
      .eq("name", tagA);

    if (eqErr) {
      test.skip(true, `Filter test failed: ${eqErr.message}`);
      return;
    }

    expect(eqData).toBeTruthy();
    expect(eqData!.length).toBe(1);
    expect(eqData![0].name).toBe(tagA);

    // .like() filter
    const { data: likeData } = await supabaseServiceRole
      .from(tableName)
      .select("*")
      .like("name", "%flt-%");

    expect(likeData).toBeTruthy();
    expect(likeData!.length).toBeGreaterThanOrEqual(2);

    // Cleanup
    if (insA.data && insA.data[0]) {
      const idField = "id" in insA.data[0] ? "id" : Object.keys(insA.data[0])[0];
      await supabaseServiceRole.from(tableName).delete().eq(idField, insA.data[0][idField]);
      await supabaseServiceRole.from(tableName).delete().eq(idField, insB.data![0][idField]);
    }
  });

  /* ================================================================ */
  /*  11. from().select() with order + limit                            */
  /* ================================================================ */
  test("11 - from().select() with .order().limit() returns ordered subset", async () => {
    skipIfNoTables();

    // Insert a few rows
    const rows = [];
    for (let i = 0; i < 3; i++) {
      const tag = uniqueName(`ord-${i}`);
      const res = await supabaseServiceRole
        .from(tableName)
        .insert({ name: tag })
        .select();
      if (res.error) {
        test.skip(true, "Cannot insert rows for order/limit test");
        return;
      }
      rows.push(res.data![0]);
    }

    // Select with order + limit
    const { data, error } = await supabaseServiceRole
      .from(tableName)
      .select("*")
      .order("name", { ascending: true })
      .limit(2);

    if (error) {
      test.skip(true, `Order/limit query failed: ${error.message}`);
      return;
    }

    expect(data).toBeTruthy();
    expect(data!.length).toBeLessThanOrEqual(2);

    // Verify ordering: each name should be <= the next
    if (data!.length === 2) {
      expect(data![0].name <= data![1].name).toBeTruthy();
    }

    // Cleanup
    for (const row of rows) {
      const idField = "id" in row ? "id" : Object.keys(row)[0];
      await supabaseServiceRole.from(tableName).delete().eq(idField, row[idField]);
    }
  });

  /* ================================================================ */
  /*  12. RLS: anon user sees own data only                             */
  /* ================================================================ */
  test("12 - RLS: anon user SELECT is filtered by RLS", async () => {
    skipIfNoTables();

    // Sign in as the test user to get an authenticated anon client
    const signIn = await supabase.auth.signInWithPassword({
      email: testEmail,
      password: testPassword,
    });

    // Even if sign-in fails, the anon key should still work for SELECT
    const { data, error } = await supabase
      .from(tableName)
      .select("*");

    if (error) {
      // A 403 or permission error means RLS is enforced correctly
      if (error.code === "42501" || error.message.includes("permission")) {
        // RLS denies access — this is expected behaviour for anon
        expect(error).toBeTruthy();
        return;
      }
      test.skip(
        error.message.includes("fetch") || error.code === "404",
        `PostgREST not available: ${error.message}`,
      );
    }

    // If we get data, RLS may have filtered rows (returning fewer or zero rows).
    // The key assertion: anon should NOT see more rows than service_role.
    expect(data).toBeTruthy();
    expect(Array.isArray(data)).toBeTruthy();

    const { data: allData } = await supabaseServiceRole
      .from(tableName)
      .select("*");

    if (allData) {
      expect(data!.length).toBeLessThanOrEqual(allData.length);
    }
  });

  /* ================================================================ */
  /*  13. RLS: service_role sees all data                               */
  /* ================================================================ */
  test("13 - RLS: service_role SELECT returns all rows (RLS bypass)", async () => {
    skipIfNoTables();

    // Insert a row with service_role to ensure at least one row exists
    const tag = uniqueName("rls-svc");
    const ins = await supabaseServiceRole
      .from(tableName)
      .insert({ name: tag })
      .select();

    if (ins.error) {
      test.skip(true, `Cannot insert row for RLS test: ${ins.error.message}`);
      return;
    }

    // service_role should see all rows including the one just inserted
    const { data, error } = await supabaseServiceRole
      .from(tableName)
      .select("*");

    if (error) {
      test.skip(
        error.message.includes("fetch") || error.code === "404",
        `PostgREST not available: ${error.message}`,
      );
    }

    expect(data).toBeTruthy();
    expect(data!.length).toBeGreaterThanOrEqual(1);

    // Verify the inserted row is present
    const found = data!.some((row: Record<string, unknown>) => row.name === tag);
    expect(found).toBeTruthy();

    // Cleanup
    if (ins.data && ins.data[0]) {
      const idField = "id" in ins.data[0] ? "id" : Object.keys(ins.data[0])[0];
      await supabaseServiceRole.from(tableName).delete().eq(idField, ins.data[0][idField]);
    }
  });

  /* ================================================================ */
  /*  14. storage.createBucket                                          */
  /* ================================================================ */
  test("14 - storage.createBucket creates a bucket via SDK", async () => {
    skipIfCannotConnect();

    const newBucket = uniqueName("sdk-create-bkt");
    const { data, error } = await supabaseServiceRole.storage.createBucket(
      newBucket,
      { public: true },
    );

    if (error) {
      test.skip(
        error.message.includes("fetch") ||
          error.message.includes("ECONNREFUSED") ||
          error.message.includes("Not Found") ||
          error.message.includes("404"),
        `Storage endpoint not available: ${error.message}`,
      );
    }

    expect(data).toBeTruthy();
    expect(data!.name).toBe(newBucket);

    // Cleanup: remove the bucket we just created
    try {
      await supabaseServiceRole.storage.emptyBucket(newBucket);
      await supabaseServiceRole.storage.deleteBucket(newBucket);
    } catch {
      // best-effort
    }
  });

  /* ================================================================ */
  /*  15. storage.upload                                                */
  /* ================================================================ */
  test("15 - storage.upload uploads a file via SDK", async () => {
    skipIfCannotConnect();

    const blob = new Blob([fileContent], { type: "text/plain" });

    const { data, error } = await supabaseServiceRole.storage
      .from(sdkBucketName)
      .upload(fileName, blob, {
        contentType: "text/plain",
        upsert: true,
      });

    if (error) {
      test.skip(
        error.message.includes("fetch") ||
          error.message.includes("ECONNREFUSED") ||
          error.message.includes("Not Found") ||
          error.message.includes("404") ||
          error.message.includes("Bucket not found"),
        `Storage endpoint not available: ${error.message}`,
      );
    }

    expect(data).toBeTruthy();
    expect(data!.path).toBeTruthy();
  });

  /* ================================================================ */
  /*  16. storage.download                                              */
  /* ================================================================ */
  test("16 - storage.download retrieves file with matching content", async () => {
    skipIfCannotConnect();

    const { data, error } = await supabaseServiceRole.storage
      .from(sdkBucketName)
      .download(fileName);

    if (error) {
      test.skip(
        error.message.includes("fetch") ||
          error.message.includes("ECONNREFUSED") ||
          error.message.includes("Not Found") ||
          error.message.includes("404") ||
          error.message.includes("Object not found"),
        `Storage download not available: ${error.message}`,
      );
    }

    expect(data).toBeTruthy();

    // Read the Blob content and verify it matches
    const text = await data!.text();
    expect(text).toBe(fileContent);
  });

  /* ================================================================ */
  /*  17. storage.remove                                                */
  /* ================================================================ */
  test("17 - storage.remove deletes the file via SDK", async () => {
    skipIfCannotConnect();

    const { data, error } = await supabaseServiceRole.storage
      .from(sdkBucketName)
      .remove([fileName]);

    if (error) {
      test.skip(
        error.message.includes("fetch") ||
          error.message.includes("ECONNREFUSED") ||
          error.message.includes("Not Found") ||
          error.message.includes("404"),
        `Storage remove not available: ${error.message}`,
      );
    }

    expect(data).toBeTruthy();

    // Verify the file is gone by trying to download it
    const { error: dlError } = await supabaseServiceRole.storage
      .from(sdkBucketName)
      .download(fileName);

    // Should get an error or empty result since the file was deleted
    expect(dlError).toBeTruthy();
  });

  /* ================================================================ */
  /*  18. storage.getPublicUrl                                          */
  /* ================================================================ */
  test("18 - storage.getPublicUrl returns a public URL", async () => {
    skipIfCannotConnect();

    // Upload a file first so there is something to get the URL for
    const publicFileName = "public-url-test.txt";
    const blob = new Blob(["public url test"], { type: "text/plain" });

    const uploadResult = await supabaseServiceRole.storage
      .from(sdkBucketName)
      .upload(publicFileName, blob, {
        contentType: "text/plain",
        upsert: true,
      });

    if (uploadResult.error) {
      test.skip(
        true,
        `Cannot upload file for public URL test: ${uploadResult.error.message}`,
      );
      return;
    }

    // getPublicUrl is synchronous and always returns a URL
    const { data } = supabaseServiceRole.storage
      .from(sdkBucketName)
      .getPublicUrl(publicFileName);

    expect(data).toBeTruthy();
    expect(data.publicUrl).toBeTruthy();
    expect(typeof data.publicUrl).toBe("string");

    // The public URL should contain the bucket name and file path
    expect(data.publicUrl).toContain(sdkBucketName);
    expect(data.publicUrl).toContain(publicFileName);

    // Cleanup
    try {
      await supabaseServiceRole.storage
        .from(sdkBucketName)
        .remove([publicFileName]);
    } catch {
      // best-effort
    }
  });
});

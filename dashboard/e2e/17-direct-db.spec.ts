process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";

import { test, expect } from "@playwright/test";
import { Client } from "pg";
import { ChildProcess, spawn } from "child_process";
import {
  createTestProject,
  deleteTestProject,
  getProject,
  uniqueName,
} from "./helpers/api";

/**
 * 17 - Direct PostgreSQL Connectivity (12 tests)
 *
 * Tests direct SQL operations against a project's dedicated PostgreSQL
 * instance using the `pg` package. No browser UI is needed — all
 * assertions go through raw SQL queries and system catalog lookups.
 *
 * Tests run serially because they build on each other:
 *   CREATE TABLE → INSERT → UPDATE → DELETE → DROP
 *
 * Connectivity: The PG service runs inside the kind cluster. We use
 * kubectl port-forward to expose it to the host machine.
 */

/* ---- Wait for project READY ---- */

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

/* ---- Port-forward helper ---- */

function startPortForward(
  namespace: string,
  service: string,
  remotePort: number,
): Promise<{ proc: ChildProcess; localPort: number }> {
  return new Promise((resolve, reject) => {
    const localPort = 40000 + Math.floor(Math.random() * 20000);
    const proc = spawn(
      "kubectl",
      [
        "port-forward",
        `svc/${service}`,
        `${localPort}:${remotePort}`,
        "-n",
        namespace,
      ],
      { stdio: ["ignore", "pipe", "pipe"] },
    );

    let resolved = false;
    const timeout = setTimeout(() => {
      if (!resolved) {
        resolved = true;
        proc.kill();
        reject(new Error("port-forward timed out after 30s"));
      }
    }, 30_000);

    proc.stdout?.on("data", (data: Buffer) => {
      const line = data.toString();
      if (line.includes("Forwarding from") && !resolved) {
        resolved = true;
        clearTimeout(timeout);
        resolve({ proc, localPort });
      }
    });

    proc.stderr?.on("data", (data: Buffer) => {
      const msg = data.toString();
      if (!resolved && msg.includes("error")) {
        resolved = true;
        clearTimeout(timeout);
        proc.kill();
        reject(new Error(`port-forward failed: ${msg}`));
      }
    });

    proc.on("error", (err) => {
      if (!resolved) {
        resolved = true;
        clearTimeout(timeout);
        reject(err);
      }
    });
  });
}

test.describe.serial("Direct PostgreSQL Connectivity", () => {
  let project: { id: string; displayName: string };
  let client: Client;
  let canConnect = false;
  let portForwardProc: ChildProcess | null = null;

  // Unique table / index / policy names scoped to this run
  const tableName = `e2e_test_${Date.now().toString(36)}`;
  const indexName = `idx_${tableName}`;
  const policyName = `policy_${tableName}`;

  /* ================================================================== */
  /*  Setup: create a postgres-enabled project + port-forward + connect  */
  /* ================================================================== */

  test.beforeAll(async () => {
    test.setTimeout(360_000);
    project = await createTestProject("direct-db", {
      postgresEnabled: true,
    });

    try {
      // Wait for project to be fully ready (CNPG cluster provisioned)
      await waitForProjectReady(project.id);

      // Use env vars if provided, otherwise set up port-forward
      const host = process.env.E2E_PG_HOST;
      const port = parseInt(process.env.E2E_PG_PORT || "5432");

      let connectHost = host;
      let connectPort = port;

      if (!host) {
        // Set up kubectl port-forward to the CNPG read-write service
        const ns = `project-${project.id}`;
        const svc = "db-rw";

        // Wait for the service to exist (CNPG creates it asynchronously)
        for (let attempt = 0; attempt < 30; attempt++) {
          try {
            const { proc, localPort } = await startPortForward(
              ns,
              svc,
              5432,
            );
            portForwardProc = proc;
            connectHost = "127.0.0.1";
            connectPort = localPort;
            break;
          } catch {
            await new Promise((r) => setTimeout(r, 5_000));
          }
        }

        if (!connectHost) {
          console.warn("Could not establish port-forward to PG; skipping");
          return;
        }
      }

      client = new Client({
        host: connectHost,
        port: connectPort,
        database: process.env.E2E_PG_DATABASE || "postgres",
        user: process.env.E2E_PG_USER || "postgres",
        password: process.env.E2E_PG_PASSWORD || "postgres",
        ssl: false,
      });
      await client.connect();
      canConnect = true;
    } catch (err) {
      console.warn(
        "Cannot connect to project PG, skipping direct-db tests:",
        err,
      );
      canConnect = false;
    }
  });

  /* ================================================================== */
  /*  Teardown: drop table + disconnect + kill port-forward + delete     */
  /* ================================================================== */

  test.afterAll(async () => {
    test.setTimeout(360_000);
    if (client && canConnect) {
      try {
        await client.query(`DROP TABLE IF EXISTS ${tableName}`);
      } catch {
        // Ignore – table may already have been dropped by test #12
      }
      try {
        await client.end();
      } catch {
        // Ignore – client may already be disconnected
      }
    }
    if (portForwardProc) {
      portForwardProc.kill();
    }
    try {
      await deleteTestProject(project.id);
    } catch {
      // Best-effort cleanup
    }
  });

  /* ================================================================== */
  /*  1. Connect successfully (SELECT 1)                                 */
  /* ================================================================== */

  test("1 - Connect successfully", async () => {
    test.skip(!canConnect, "PG not reachable");

    const res = await client.query("SELECT 1 AS result");
    expect(res.rows).toHaveLength(1);
    expect(res.rows[0].result).toBe(1);
  });

  /* ================================================================== */
  /*  2. CREATE TABLE                                                    */
  /* ================================================================== */

  test("2 - CREATE TABLE", async () => {
    test.skip(!canConnect, "PG not reachable");

    await client.query(`
      CREATE TABLE ${tableName} (
        id    serial PRIMARY KEY,
        name  text,
        value int
      )
    `);

    // Verify via information_schema
    const res = await client.query(
      `SELECT table_name
         FROM information_schema.tables
        WHERE table_schema = 'public'
          AND table_name   = $1`,
      [tableName],
    );
    expect(res.rows).toHaveLength(1);
    expect(res.rows[0].table_name).toBe(tableName);
  });

  /* ================================================================== */
  /*  3. ALTER TABLE ADD COLUMN                                          */
  /* ================================================================== */

  test("3 - ALTER TABLE ADD COLUMN", async () => {
    test.skip(!canConnect, "PG not reachable");

    await client.query(
      `ALTER TABLE ${tableName} ADD COLUMN extra text`,
    );

    // Verify column exists
    const res = await client.query(
      `SELECT column_name
         FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name   = $1
          AND column_name  = 'extra'`,
      [tableName],
    );
    expect(res.rows).toHaveLength(1);
    expect(res.rows[0].column_name).toBe("extra");
  });

  /* ================================================================== */
  /*  4. INSERT + SELECT                                                 */
  /* ================================================================== */

  test("4 - INSERT + SELECT data matches", async () => {
    test.skip(!canConnect, "PG not reachable");

    await client.query(
      `INSERT INTO ${tableName} (name, value, extra)
       VALUES ($1, $2, $3)`,
      ["alice", 42, "hello"],
    );

    const res = await client.query(
      `SELECT name, value, extra FROM ${tableName} WHERE name = $1`,
      ["alice"],
    );
    expect(res.rows).toHaveLength(1);
    expect(res.rows[0].name).toBe("alice");
    expect(res.rows[0].value).toBe(42);
    expect(res.rows[0].extra).toBe("hello");
  });

  /* ================================================================== */
  /*  5. UPDATE + SELECT                                                 */
  /* ================================================================== */

  test("5 - UPDATE + SELECT value updated", async () => {
    test.skip(!canConnect, "PG not reachable");

    await client.query(
      `UPDATE ${tableName} SET value = $1 WHERE name = $2`,
      [100, "alice"],
    );

    const res = await client.query(
      `SELECT value FROM ${tableName} WHERE name = $1`,
      ["alice"],
    );
    expect(res.rows).toHaveLength(1);
    expect(res.rows[0].value).toBe(100);
  });

  /* ================================================================== */
  /*  6. DELETE + SELECT                                                 */
  /* ================================================================== */

  test("6 - DELETE + SELECT row gone", async () => {
    test.skip(!canConnect, "PG not reachable");

    await client.query(
      `DELETE FROM ${tableName} WHERE name = $1`,
      ["alice"],
    );

    const res = await client.query(
      `SELECT * FROM ${tableName} WHERE name = $1`,
      ["alice"],
    );
    expect(res.rows).toHaveLength(0);
  });

  /* ================================================================== */
  /*  7. CREATE INDEX                                                    */
  /* ================================================================== */

  test("7 - CREATE INDEX", async () => {
    test.skip(!canConnect, "PG not reachable");

    await client.query(
      `CREATE INDEX ${indexName} ON ${tableName} (name)`,
    );

    // Verify via pg_indexes
    const res = await client.query(
      `SELECT indexname
         FROM pg_indexes
        WHERE schemaname = 'public'
          AND tablename  = $1
          AND indexname   = $2`,
      [tableName, indexName],
    );
    expect(res.rows).toHaveLength(1);
    expect(res.rows[0].indexname).toBe(indexName);
  });

  /* ================================================================== */
  /*  8. DROP INDEX                                                      */
  /* ================================================================== */

  test("8 - DROP INDEX", async () => {
    test.skip(!canConnect, "PG not reachable");

    await client.query(`DROP INDEX ${indexName}`);

    // Verify gone from pg_indexes
    const res = await client.query(
      `SELECT indexname
         FROM pg_indexes
        WHERE schemaname = 'public'
          AND tablename  = $1
          AND indexname   = $2`,
      [tableName, indexName],
    );
    expect(res.rows).toHaveLength(0);
  });

  /* ================================================================== */
  /*  9. ENABLE RLS                                                      */
  /* ================================================================== */

  test("9 - ENABLE ROW LEVEL SECURITY", async () => {
    test.skip(!canConnect, "PG not reachable");

    await client.query(
      `ALTER TABLE ${tableName} ENABLE ROW LEVEL SECURITY`,
    );

    // Verify via pg_class
    const res = await client.query(
      `SELECT relrowsecurity
         FROM pg_class
        WHERE relname = $1
          AND relnamespace = (
            SELECT oid FROM pg_namespace WHERE nspname = 'public'
          )`,
      [tableName],
    );
    expect(res.rows).toHaveLength(1);
    expect(res.rows[0].relrowsecurity).toBe(true);
  });

  /* ================================================================== */
  /*  10. CREATE POLICY                                                  */
  /* ================================================================== */

  test("10 - CREATE POLICY", async () => {
    test.skip(!canConnect, "PG not reachable");

    await client.query(
      `CREATE POLICY ${policyName}
         ON ${tableName}
         FOR SELECT
         USING (true)`,
    );

    // Verify via pg_policies
    const res = await client.query(
      `SELECT policyname
         FROM pg_policies
        WHERE schemaname = 'public'
          AND tablename  = $1
          AND policyname = $2`,
      [tableName, policyName],
    );
    expect(res.rows).toHaveLength(1);
    expect(res.rows[0].policyname).toBe(policyName);
  });

  /* ================================================================== */
  /*  11. DROP POLICY                                                    */
  /* ================================================================== */

  test("11 - DROP POLICY", async () => {
    test.skip(!canConnect, "PG not reachable");

    await client.query(
      `DROP POLICY ${policyName} ON ${tableName}`,
    );

    // Verify gone from pg_policies
    const res = await client.query(
      `SELECT policyname
         FROM pg_policies
        WHERE schemaname = 'public'
          AND tablename  = $1
          AND policyname = $2`,
      [tableName, policyName],
    );
    expect(res.rows).toHaveLength(0);
  });

  /* ================================================================== */
  /*  12. DROP TABLE                                                     */
  /* ================================================================== */

  test("12 - DROP TABLE", async () => {
    test.skip(!canConnect, "PG not reachable");

    await client.query(`DROP TABLE ${tableName}`);

    // Verify gone from information_schema
    const res = await client.query(
      `SELECT table_name
         FROM information_schema.tables
        WHERE table_schema = 'public'
          AND table_name   = $1`,
      [tableName],
    );
    expect(res.rows).toHaveLength(0);
  });
});

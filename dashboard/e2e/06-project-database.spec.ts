import { test, expect } from "@playwright/test";
import { createTestProject, deleteTestProject, getAuthToken } from "./helpers/api";

/**
 * 06 - Project Database Tab
 *
 * Part A (#1-28): UI element visibility checks — sub-tabs, schema sidebar,
 * table actions, mini-tabs, SQL editor, connection info.
 *
 * Part B (#29-43): Real CRUD operations via postgres-meta + PostgREST:
 * create/drop table, add/delete column, insert/delete row, run SQL,
 * toggle RLS, create/drop policy.
 */

const BASE_URL =
  process.env.E2E_BASE_URL || "https://dashboard.local.etalbaas.dev";

/* ===== Helper: wait for postgres-meta to be ready ===== */
async function waitForDbReady(
  projectId: string,
  timeoutMs = 300_000,
): Promise<void> {
  const token = await getAuthToken();
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    try {
      const res = await fetch(
        `${BASE_URL}/api/database/tables?project_id=${projectId}`,
        {
          headers: { Authorization: `Bearer ${token}` },
        },
      );
      if (res.ok) {
        const data = await res.json();
        if (Array.isArray(data)) return;
      }
    } catch {
      // not ready yet
    }
    await new Promise((r) => setTimeout(r, 5_000));
  }
  throw new Error(`Database not ready after ${timeoutMs / 1000}s`);
}

/* ===== Helper: seed test data via postgres-meta /query ===== */
async function seedTestData(projectId: string): Promise<void> {
  const token = await getAuthToken();
  const statements = [
    `CREATE TABLE IF NOT EXISTS e2e_items (
       id serial PRIMARY KEY,
       name text NOT NULL,
       value text
     )`,
    `INSERT INTO e2e_items (name, value) VALUES ('item1', 'val1'), ('item2', 'val2')`,
    `CREATE TABLE IF NOT EXISTS profiles (
       id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
       email text,
       name text
     )`,
    `INSERT INTO profiles (email, name) VALUES ('test@test.com', 'Test User')`,
    `ALTER TABLE e2e_items ENABLE ROW LEVEL SECURITY`,
    `CREATE POLICY e2e_select ON e2e_items FOR SELECT USING (true)`,
    // Notify PostgREST to reload schema cache after DDL changes.
    `NOTIFY pgrst, 'reload schema'`,
  ];

  for (const sql of statements) {
    const res = await fetch(
      `${BASE_URL}/api/database/query?project_id=${projectId}`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ query: sql }),
      },
    );
    if (!res.ok) {
      const body = await res.text();
      // Ignore "already exists" errors for idempotency
      if (!body.includes("already exists")) {
        throw new Error(`Seed SQL failed: ${res.status} ${body}\nSQL: ${sql}`);
      }
    }
  }
}

/* ===== Helper: wait for PostgREST to be reachable ===== */
async function waitForPostgrestReady(
  projectId: string,
  table: string,
  timeoutMs = 120_000,
): Promise<void> {
  const token = await getAuthToken();
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    try {
      const res = await fetch(
        `${BASE_URL}/api/postgrest/${table}?project_id=${projectId}&limit=1`,
        { headers: { Authorization: `Bearer ${token}` } },
      );
      if (res.ok) return;
    } catch {
      // not ready yet
    }
    await new Promise((r) => setTimeout(r, 5_000));
  }
  throw new Error(`PostgREST not ready after ${timeoutMs / 1000}s`);
}

test.describe.serial("Project Database Tab", () => {
  let project: { id: string; displayName: string };

  /** If CNPG or postgres-meta isn't ready, tests skip gracefully. */
  let suiteAvailable = false;

  /* ---- Setup: create project with postgres enabled, seed data ---- */
  test.beforeAll(async () => {
    test.setTimeout(360_000); // 6 min for CNPG startup + seeding
    project = await createTestProject("db-tab", {
      postgresEnabled: true,
      postgrestEnabled: true,
    });

    // Wait for postgres-meta to respond (CNPG needs to start first)
    try {
      await waitForDbReady(project.id);
    } catch (err) {
      console.warn(`Database not ready (skipping suite): ${err}`);
      return; // suiteAvailable stays false
    }

    // Seed test tables, columns, data, and RLS policy
    try {
      await seedTestData(project.id);
    } catch (err) {
      console.warn(`Seed data failed (skipping suite): ${err}`);
      return;
    }

    // Wait for PostgREST to be ready and serving the seeded table
    try {
      await waitForPostgrestReady(project.id, "e2e_items");
    } catch (err) {
      console.warn(`PostgREST not ready (skipping suite): ${err}`);
      return;
    }

    suiteAvailable = true;
  });

  /* ---- Teardown ---- */
  test.afterAll(async () => {
    try {
      await deleteTestProject(project.id);
    } catch {}
  });

  /* ---- Navigate to Database tab before each test ---- */
  test.beforeEach(async ({ page }) => {
    test.skip(!suiteAvailable, "Database not available");
    await page.goto(`/projects/${project.id}`, { waitUntil: "networkidle" });
    await page.getByRole("tab", { name: "Database" }).click({ timeout: 60_000 });
    // Wait for sub-tabs to appear
    await expect(page.getByRole("tab", { name: "Tables" })).toBeVisible({
      timeout: 15_000,
    });
  });

  /* ================================================================== */
  /*  Part A: Sub-tab navigation                                        */
  /* ================================================================== */

  test("1 - Tables sub-tab is default active", async ({ page }) => {
    const tablesTab = page.getByRole("tab", { name: "Tables" });
    await expect(tablesTab).toHaveAttribute("aria-selected", "true");
  });

  test("2 - clicking SQL Editor sub-tab shows SQL editor", async ({
    page,
  }) => {
    const sqlTab = page.getByRole("tab", { name: "SQL Editor" });
    await sqlTab.click();
    await expect(sqlTab).toHaveAttribute("aria-selected", "true");

    const editorArea = page.locator("textarea.sql-code");
    await expect(editorArea).toBeVisible();
  });

  test("3 - clicking Connection sub-tab shows connection panels", async ({
    page,
  }) => {
    const connTab = page.getByRole("tab", { name: "Connection" });
    await connTab.click();
    await expect(connTab).toHaveAttribute("aria-selected", "true");

    const copyButtons = page.locator("button[aria-label='Copy']");
    await expect(copyButtons.first()).toBeVisible();
  });

  /* ================================================================== */
  /*  Schema sidebar                                                    */
  /* ================================================================== */

  test("4 - public schema group is visible and collapsible", async ({
    page,
  }) => {
    // Wait for tables to load
    const publicHeader = page.locator("button.schema-group-head", {
      hasText: "public",
    });
    await expect(publicHeader).toBeVisible({ timeout: 15_000 });

    // Tables should be visible initially
    await expect(
      page.locator("button.schema-table", { hasText: "e2e_items" }),
    ).toBeVisible();

    // Click to collapse
    await publicHeader.click();
    await expect(
      page.locator("button.schema-table", { hasText: "e2e_items" }),
    ).not.toBeVisible();

    // Click again to expand
    await publicHeader.click();
    await expect(
      page.locator("button.schema-table", { hasText: "e2e_items" }),
    ).toBeVisible();
  });

  test("5 - clicking different table names updates detail view", async ({
    page,
  }) => {
    // Wait for tables to load
    const firstTable = page.locator("button.schema-table", {
      hasText: "e2e_items",
    });
    await expect(firstTable).toBeVisible({ timeout: 15_000 });
    await firstTable.click();

    // Detail header should show table name
    await expect(page.getByText("e2e_items").first()).toBeVisible();

    // Click profiles table
    const secondTable = page.locator("button.schema-table", {
      hasText: "profiles",
    });
    await secondTable.click();
    await expect(page.getByText("profiles").first()).toBeVisible();
  });

  /* ================================================================== */
  /*  Table action buttons                                              */
  /* ================================================================== */

  test("6 - New Table button is visible and enabled", async ({ page }) => {
    const newTableBtn = page.getByRole("button", { name: "New Table" });
    await expect(newTableBtn).toBeVisible();
    await expect(newTableBtn).toBeEnabled();
  });

  test("7 - Add Column button is visible", async ({ page }) => {
    // Select a table first
    const tableItem = page.locator("button.schema-table", {
      hasText: "e2e_items",
    });
    await expect(tableItem).toBeVisible({ timeout: 15_000 });
    await tableItem.click();

    const addColBtn = page.getByRole("button", { name: "Add Column" });
    await expect(addColBtn).toBeVisible();
  });

  test("8 - table menu shows Drop Table and other items", async ({ page }) => {
    // Select a table first
    const tableItem = page.locator("button.schema-table", {
      hasText: "e2e_items",
    });
    await expect(tableItem).toBeVisible({ timeout: 15_000 });
    await tableItem.click();

    // Open the kebab menu
    const menuBtn = page.locator(".icon-menu-btn").first();
    await expect(menuBtn).toBeVisible();
    await menuBtn.click();

    // Verify menu items
    await expect(page.getByText("Export as SQL")).toBeVisible();
    await expect(page.getByText("Truncate")).toBeVisible();
    await expect(page.getByText("Drop Table")).toBeVisible();
  });

  /* ================================================================== */
  /*  Mini-tabs for table details                                       */
  /* ================================================================== */

  test("9 - Columns mini-tab is default active", async ({ page }) => {
    const tableItem = page.locator("button.schema-table", {
      hasText: "e2e_items",
    });
    await expect(tableItem).toBeVisible({ timeout: 15_000 });
    await tableItem.click();

    const columnsTab = page.getByRole("tab", { name: "Columns" });
    await expect(columnsTab).toBeVisible();
    await expect(columnsTab).toHaveAttribute("aria-selected", "true");
  });

  test("10 - clicking Data mini-tab shows data table", async ({ page }) => {
    const tableItem = page.locator("button.schema-table", {
      hasText: "e2e_items",
    });
    await expect(tableItem).toBeVisible({ timeout: 15_000 });
    await tableItem.click();

    const dataTab = page.getByRole("tab", { name: "Data", exact: true });
    await dataTab.click();
    await expect(dataTab).toHaveAttribute("aria-selected", "true");

    // Data table should be visible
    const dataTable = page.locator("table").first();
    await expect(dataTable).toBeVisible();
  });

  test("11 - clicking RLS mini-tab shows RLS toggle", async ({ page }) => {
    const tableItem = page.locator("button.schema-table", {
      hasText: "e2e_items",
    });
    await expect(tableItem).toBeVisible({ timeout: 15_000 });
    await tableItem.click();

    const rlsTab = page.getByRole("tab", { name: "RLS" });
    await rlsTab.click();
    await expect(rlsTab).toHaveAttribute("aria-selected", "true");

    const rlsToggle = page
      .locator("input[type='checkbox'], [role='switch']")
      .first();
    await expect(rlsToggle).toBeVisible();
  });

  test("12 - clicking Indexes mini-tab shows index content", async ({
    page,
  }) => {
    const tableItem = page.locator("button.schema-table", {
      hasText: "e2e_items",
    });
    await expect(tableItem).toBeVisible({ timeout: 15_000 });
    await tableItem.click();

    const indexesTab = page.getByRole("tab", { name: "Indexes" });
    await indexesTab.click();
    await expect(indexesTab).toHaveAttribute("aria-selected", "true");

    // Index content should be visible (e2e_items has a pkey index)
    await expect(page.locator(".idx-type", { hasText: "btree" }).first()).toBeVisible();
  });

  /* ================================================================== */
  /*  Columns view details                                              */
  /* ================================================================== */

  test("13 - column rows are displayed from postgres-meta", async ({
    page,
  }) => {
    const tableItem = page.locator("button.schema-table", {
      hasText: "e2e_items",
    });
    await expect(tableItem).toBeVisible({ timeout: 15_000 });
    await tableItem.click();

    // Columns tab should be active by default — check for column names
    await expect(page.locator(".col-name", { hasText: "id" })).toBeVisible({
      timeout: 10_000,
    });
    await expect(
      page.locator(".col-name", { hasText: "name" }),
    ).toBeVisible();
  });

  /* ================================================================== */
  /*  Data view details                                                 */
  /* ================================================================== */

  test("14 - Data view: Insert Row button is visible", async ({ page }) => {
    const tableItem = page.locator("button.schema-table", {
      hasText: "e2e_items",
    });
    await expect(tableItem).toBeVisible({ timeout: 15_000 });
    await tableItem.click();

    const dataTab = page.getByRole("tab", { name: "Data", exact: true });
    await dataTab.click();

    const insertBtn = page.getByRole("button", { name: "Insert Row" });
    await expect(insertBtn).toBeVisible();
  });

  test("15 - Data view: rows from PostgREST are displayed", async ({
    page,
  }) => {
    const tableItem = page.locator("button.schema-table", {
      hasText: "e2e_items",
    });
    await expect(tableItem).toBeVisible({ timeout: 15_000 });
    await tableItem.click();

    const dataTab = page.getByRole("tab", { name: "Data", exact: true });
    await dataTab.click();

    // Seeded rows should be visible
    await expect(page.locator("td", { hasText: "item1" })).toBeVisible({
      timeout: 10_000,
    });
  });

  test("16 - Data view: pagination buttons exist", async ({ page }) => {
    const tableItem = page.locator("button.schema-table", {
      hasText: "e2e_items",
    });
    await expect(tableItem).toBeVisible({ timeout: 15_000 });
    await tableItem.click();

    const dataTab = page.getByRole("tab", { name: "Data", exact: true });
    await dataTab.click();

    // Pagination controls should exist
    const pagination = page.locator(".pagination-bar");
    await expect(pagination).toBeVisible();
  });

  /* ================================================================== */
  /*  RLS view details                                                  */
  /* ================================================================== */

  test("17 - RLS view: toggle and New Policy button visible", async ({
    page,
  }) => {
    const tableItem = page.locator("button.schema-table", {
      hasText: "e2e_items",
    });
    await expect(tableItem).toBeVisible({ timeout: 15_000 });
    await tableItem.click();

    const rlsTab = page.getByRole("tab", { name: "RLS" });
    await rlsTab.click();

    const toggle = page
      .locator("input[type='checkbox'], [role='switch']")
      .first();
    await expect(toggle).toBeVisible();

    const newPolicyBtn = page.getByRole("button", { name: "New Policy" });
    await expect(newPolicyBtn).toBeVisible();
  });

  test("18 - RLS view: seeded policy card visible", async ({ page }) => {
    const tableItem = page.locator("button.schema-table", {
      hasText: "e2e_items",
    });
    await expect(tableItem).toBeVisible({ timeout: 15_000 });
    await tableItem.click();

    const rlsTab = page.getByRole("tab", { name: "RLS" });
    await rlsTab.click();

    // Policy from seed should be visible
    await expect(
      page.locator(".policy-card", { hasText: "e2e_select" }),
    ).toBeVisible({ timeout: 10_000 });
  });

  /* ================================================================== */
  /*  SQL Editor sub-tab                                                */
  /* ================================================================== */

  test("19 - SQL Editor: Run and Clear buttons are visible", async ({
    page,
  }) => {
    const sqlTab = page.getByRole("tab", { name: "SQL Editor" });
    await sqlTab.click();

    await expect(page.getByRole("button", { name: /Run/ })).toBeVisible();
    await expect(page.getByRole("button", { name: "Clear" })).toBeVisible();
  });

  test("20 - SQL Editor: textarea area exists", async ({ page }) => {
    const sqlTab = page.getByRole("tab", { name: "SQL Editor" });
    await sqlTab.click();

    await expect(page.locator("textarea.sql-code")).toBeVisible();
  });

  test("21 - SQL Editor: Results and Messages tab toggle exists", async ({
    page,
  }) => {
    const sqlTab = page.getByRole("tab", { name: "SQL Editor" });
    await sqlTab.click();

    const resultsTab = page.getByRole("tab", { name: "Results" });
    const messagesTab = page.getByRole("tab", { name: "Messages" });

    await expect(resultsTab).toBeVisible();
    await expect(messagesTab).toBeVisible();

    await messagesTab.click();
    await expect(messagesTab).toHaveAttribute("aria-selected", "true");

    await resultsTab.click();
    await expect(resultsTab).toHaveAttribute("aria-selected", "true");
  });

  /* ================================================================== */
  /*  Connection sub-tab                                                */
  /* ================================================================== */

  test("22 - Connection: copy buttons are visible", async ({ page }) => {
    const connTab = page.getByRole("tab", { name: "Connection" });
    await connTab.click();

    const copyButtons = page.locator("button[aria-label='Copy']");
    const count = await copyButtons.count();
    expect(count).toBeGreaterThanOrEqual(5);
  });

  test("23 - Connection: show/hide toggles are visible", async ({ page }) => {
    const connTab = page.getByRole("tab", { name: "Connection" });
    await connTab.click();

    const toggleButtons = page.locator(
      "button[aria-label='Toggle visibility']",
    );
    const count = await toggleButtons.count();
    expect(count).toBeGreaterThanOrEqual(2);
  });

  /* ================================================================== */
  /*  Part B: Real CRUD operations (postgres-meta + PostgREST)          */
  /* ================================================================== */

  test("24 - create table via New Table modal", async ({ page }) => {
    const newTableBtn = page.getByRole("button", { name: "New Table" });
    await newTableBtn.click();

    const modal = page.locator(".modal-overlay");
    await expect(modal).toBeVisible();

    const nameInput = modal.locator("input[type='text']");
    await nameInput.fill("e2e_test_crud");

    const createBtn = modal.getByRole("button", { name: /Create Table/ });
    await createBtn.click();

    await expect(modal).not.toBeVisible({ timeout: 10_000 });

    await expect(
      page.locator("button.schema-table", { hasText: "e2e_test_crud" }),
    ).toBeVisible({ timeout: 10_000 });
  });

  test("25 - created table is auto-selected", async ({ page }) => {
    const tableBtn = page.locator("button.schema-table", {
      hasText: "e2e_test_crud",
    });
    await expect(tableBtn).toBeVisible({ timeout: 10_000 });
    await tableBtn.click();

    const header = page.locator(".table-main-head h2");
    await expect(header).toContainText("e2e_test_crud");
  });

  test("26 - add column via Add Column modal", async ({ page }) => {
    const tableBtn = page.locator("button.schema-table", {
      hasText: "e2e_test_crud",
    });
    await expect(tableBtn).toBeVisible({ timeout: 10_000 });
    await tableBtn.click();

    const addColBtn = page.getByRole("button", { name: "Add Column" });
    await addColBtn.click();

    const modal = page.locator(".modal-overlay");
    await expect(modal).toBeVisible();

    const nameInput = modal.locator("input[type='text']").first();
    await nameInput.fill("test_name");

    const typeSelect = modal.locator("select");
    await expect(typeSelect).toHaveValue("text");

    const addBtn = modal.getByRole("button", { name: /Add Column/ });
    await addBtn.click();

    await expect(modal).not.toBeVisible({ timeout: 10_000 });

    await expect(
      page.locator(".col-name", { hasText: "test_name" }),
    ).toBeVisible({ timeout: 10_000 });
  });

  test("27 - insert row via Insert Row modal", async ({ page }) => {
    const tableBtn = page.locator("button.schema-table", {
      hasText: "e2e_test_crud",
    });
    await expect(tableBtn).toBeVisible({ timeout: 10_000 });
    await tableBtn.click();

    const dataTab = page.locator("button.mini-tab", { hasText: "Data" });
    await dataTab.click();

    const insertBtn = page.getByRole("button", { name: "Insert Row" });
    await insertBtn.click();

    const modal = page.locator(".modal-overlay");
    await expect(modal).toBeVisible();

    const nameInput = modal.locator("input[type='text']").first();
    await nameInput.fill("e2e_row_value");

    const confirmBtn = modal.getByRole("button", { name: /Insert Row/ });
    await confirmBtn.click();

    await expect(modal).not.toBeVisible({ timeout: 10_000 });

    await expect(
      page.locator("td", { hasText: "e2e_row_value" }),
    ).toBeVisible({ timeout: 10_000 });
  });

  test("28 - row count updates after insert", async ({ page }) => {
    const tableBtn = page.locator("button.schema-table", {
      hasText: "e2e_test_crud",
    });
    await expect(tableBtn).toBeVisible({ timeout: 10_000 });
    await tableBtn.click();

    const dataTab = page.locator("button.mini-tab", { hasText: "Data" });
    await dataTab.click();

    // Wait for row count to update (PostgREST HEAD may take a moment)
    await expect(page.locator(".row-count")).toContainText(/[1-9]/, {
      timeout: 15_000,
    });
  });

  test("29 - delete row via delete button", async ({ page }) => {
    const tableBtn = page.locator("button.schema-table", {
      hasText: "e2e_test_crud",
    });
    await expect(tableBtn).toBeVisible({ timeout: 10_000 });
    await tableBtn.click();

    const dataTab = page.locator("button.mini-tab", { hasText: "Data" });
    await dataTab.click();

    await expect(
      page.locator("td", { hasText: "e2e_row_value" }),
    ).toBeVisible({ timeout: 10_000 });

    page.on("dialog", async (dialog) => {
      await dialog.accept();
    });

    const row = page.locator("table tbody tr", { hasText: "e2e_row_value" });
    const deleteBtn = row.locator("button.del");
    await deleteBtn.click();

    // Wait for mutation to complete, then reload to verify deletion
    await page.waitForTimeout(2_000);
    await page.reload({ waitUntil: "networkidle" });
    await page.getByRole("tab", { name: "Database" }).click({ timeout: 30_000 });
    const reloadedTable = page.locator("button.schema-table", {
      hasText: "e2e_test_crud",
    });
    await expect(reloadedTable).toBeVisible({ timeout: 15_000 });
    await reloadedTable.click();
    await page.locator("button.mini-tab", { hasText: "Data" }).click();

    await expect(
      page.locator("td", { hasText: "e2e_row_value" }),
    ).not.toBeVisible({ timeout: 10_000 });
  });

  test("30 - SQL Editor: run query returns results", async ({ page }) => {
    const sqlTab = page.getByRole("tab", { name: "SQL Editor" });
    await sqlTab.click();

    const textarea = page.locator("textarea.sql-code");
    await textarea.fill("SELECT 1 AS result;");

    const runBtn = page.getByRole("button", { name: /Run/ });
    await runBtn.click();

    await expect(page.locator("th", { hasText: "result" })).toBeVisible({
      timeout: 15_000,
    });
    await expect(page.locator("td", { hasText: "1" })).toBeVisible();
  });

  test("31 - SQL Editor: invalid query shows error in Messages", async ({
    page,
  }) => {
    const sqlTab = page.getByRole("tab", { name: "SQL Editor" });
    await sqlTab.click();

    const textarea = page.locator("textarea.sql-code");
    await textarea.fill("SELECTT INVALID;");

    const runBtn = page.getByRole("button", { name: /Run/ });
    await runBtn.click();

    const messagesTab = page.getByRole("tab", { name: "Messages" });
    await messagesTab.click();
    // Target the Messages pane (second .sql-results-pane) to avoid matching the hidden Results pane
    const messagesPane = page.locator(".sql-results-pane").nth(1);
    await expect(
      messagesPane.locator(".sql-message", { hasText: "ERROR" }),
    ).toBeVisible({ timeout: 15_000 });
  });

  test("32 - SQL Editor: Clear button resets editor", async ({ page }) => {
    const sqlTab = page.getByRole("tab", { name: "SQL Editor" });
    await sqlTab.click();

    const textarea = page.locator("textarea.sql-code");
    await textarea.fill("SELECT 42;");

    const clearBtn = page.getByRole("button", { name: "Clear" });
    await clearBtn.click();

    await expect(textarea).toHaveValue("");
  });

  test("33 - SQL Editor: Ctrl+Enter shortcut runs query", async ({ page }) => {
    const sqlTab = page.getByRole("tab", { name: "SQL Editor" });
    await sqlTab.click();

    const textarea = page.locator("textarea.sql-code");
    await textarea.fill("SELECT 42 AS answer;");

    await textarea.press("Control+Enter");

    await expect(page.locator("th", { hasText: "answer" })).toBeVisible({
      timeout: 15_000,
    });
    await expect(page.locator("td", { hasText: "42" })).toBeVisible();
  });

  test("34 - toggle RLS on table", async ({ page }) => {
    const tableBtn = page.locator("button.schema-table", {
      hasText: "e2e_test_crud",
    });
    await expect(tableBtn).toBeVisible({ timeout: 10_000 });
    await tableBtn.click();

    const rlsTab = page.locator("button.mini-tab", {
      hasText: "RLS Policies",
    });
    await rlsTab.click();

    // RLS should initially be disabled for the newly created table
    await expect(page.locator(".rls-toolbar")).toContainText("Disabled", {
      timeout: 10_000,
    });

    const rlsSwitch = page.locator("input.rls-switch");
    await rlsSwitch.click();

    // Wait for the mutation to complete and UI to update
    await expect(page.locator(".rls-toolbar")).toContainText("Enabled", {
      timeout: 15_000,
    });
  });

  test("35 - create RLS policy via New Policy modal", async ({ page }) => {
    const tableBtn = page.locator("button.schema-table", {
      hasText: "e2e_test_crud",
    });
    await expect(tableBtn).toBeVisible({ timeout: 10_000 });
    await tableBtn.click();

    const rlsTab = page.locator("button.mini-tab", {
      hasText: "RLS Policies",
    });
    await rlsTab.click();

    const newPolicyBtn = page.getByRole("button", { name: "New Policy" });
    await newPolicyBtn.click();

    const modal = page.locator(".modal-overlay");
    await expect(modal).toBeVisible();

    const nameInput = modal.locator("input[type='text']").first();
    await nameInput.fill("e2e_test_policy");

    const defInput = modal.locator("input[type='text']").nth(1);
    await expect(defInput).toHaveValue("true");

    const createBtn = modal.getByRole("button", { name: /Create Policy/ });
    await createBtn.click();

    await expect(modal).not.toBeVisible({ timeout: 10_000 });

    await expect(
      page.locator(".policy-card .pc-name", { hasText: "e2e_test_policy" }),
    ).toBeVisible({ timeout: 10_000 });
  });

  test("36 - drop RLS policy via Drop button", async ({ page }) => {
    const tableBtn = page.locator("button.schema-table", {
      hasText: "e2e_test_crud",
    });
    await expect(tableBtn).toBeVisible({ timeout: 10_000 });
    await tableBtn.click();

    const rlsTab = page.locator("button.mini-tab", {
      hasText: "RLS Policies",
    });
    await rlsTab.click();

    const policyCard = page.locator(".policy-card", {
      hasText: "e2e_test_policy",
    });
    await expect(policyCard).toBeVisible({ timeout: 10_000 });

    page.on("dialog", async (dialog) => {
      await dialog.accept();
    });

    const dropBtn = policyCard.getByRole("button", { name: "Drop" });
    await dropBtn.click();

    await expect(policyCard).not.toBeVisible({ timeout: 10_000 });
  });

  test("37 - delete column via delete button", async ({ page }) => {
    const tableBtn = page.locator("button.schema-table", {
      hasText: "e2e_test_crud",
    });
    await expect(tableBtn).toBeVisible({ timeout: 10_000 });
    await tableBtn.click();

    const columnsTab = page.locator("button.mini-tab", {
      hasText: "Columns",
    });
    await columnsTab.click();

    await expect(
      page.locator(".col-name", { hasText: "test_name" }),
    ).toBeVisible({ timeout: 10_000 });

    page.on("dialog", async (dialog) => {
      await dialog.accept();
    });

    const colRow = page.locator(".row.body", { hasText: "test_name" });
    const deleteBtn = colRow.locator("button.del");
    await deleteBtn.click();

    await expect(
      page.locator(".col-name", { hasText: "test_name" }),
    ).not.toBeVisible({ timeout: 10_000 });
  });

  test("38 - drop table via Drop Table menu item", async ({ page }) => {
    const tableBtn = page.locator("button.schema-table", {
      hasText: "e2e_test_crud",
    });
    await expect(tableBtn).toBeVisible({ timeout: 10_000 });
    await tableBtn.click();

    page.on("dialog", async (dialog) => {
      await dialog.accept();
    });

    const kebab = page.locator(".icon-menu-btn").first();
    await kebab.click();
    const dropBtn = page.locator(".menu-dropdown button", {
      hasText: "Drop Table",
    });
    await dropBtn.click();

    await expect(
      page.locator("button.schema-table", { hasText: "e2e_test_crud" }),
    ).not.toBeVisible({ timeout: 10_000 });
  });
});

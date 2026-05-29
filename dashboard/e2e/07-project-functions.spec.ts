import { test, expect } from "@playwright/test";
import {
  createTestProject,
  deleteTestProject,
  createTestFunction,
  deleteTestFunction,
  listFunctions,
  uniqueName,
} from "./helpers/api";

/**
 * 07 - Project Functions Tab
 *
 * Covers the /projects/{id} page Functions tab:
 * function list, search/filter, table interactions, and the Deploy Function modal
 * with all its sections (basic, source, runtime, timeout/GPU, triggers, env vars).
 *
 * Setup creates 1 project + 3 functions with different kinds via API.
 */

test.describe.serial("Project Functions Tab", () => {
  let project: { id: string; displayName: string };
  let functions: Array<{ id: string; name: string; kind: string }>;
  /** IDs of functions created via the Deploy modal, collected for cleanup */
  const deployedFnIds: string[] = [];

  test.beforeAll(async () => {
    project = await createTestProject("fn-tab");
    functions = [];
    const f1 = await createTestFunction(project.id, "heavyjob", {
      kind: "heavy-job",
      mode: "async",
    });
    const f2 = await createTestFunction(project.id, "heavydeploy", {
      kind: "heavy-deployment",
      mode: "sync",
    });
    const f3 = await createTestFunction(project.id, "lightfn", {
      kind: "light-deployment",
      mode: "sync",
    });
    functions.push(f1, f2, f3);
  });

  test.afterAll(async () => {
    // Clean up deployed functions created via modal
    for (const id of deployedFnIds) {
      try {
        await deleteTestFunction(project.id, id);
      } catch {}
    }
    // Clean up API-seeded functions
    for (const f of functions) {
      try {
        await deleteTestFunction(project.id, f.id);
      } catch {}
    }
    try {
      await deleteTestProject(project.id);
    } catch {}
  });

  test.beforeEach(async ({ page }) => {
    await page.goto(`/projects/${project.id}`);
    // Click the Functions tab — use regex to match with or without count badge
    const fnTab = page.locator('[role="tab"]').filter({ hasText: /^Functions/ });
    await fnTab.click();
    await expect(fnTab).toHaveAttribute("aria-selected", "true");
    // Wait for the function table rows to load (not just "Deploy Function" which shows in empty state too)
    await expect(
      page.locator('input[placeholder="Search functions\u2026"]'),
    ).toBeVisible({ timeout: 15000 });
  });

  /* ================================================================== */
  /*  Section A: List operation tests (1-12)                             */
  /* ================================================================== */

  /* ------------------------------------------------------------------ */
  /*  1. Search filters by name                                          */
  /* ------------------------------------------------------------------ */
  test("1 - search filters functions by name", async ({ page }) => {
    const searchInput = page.locator(
      'input[placeholder="Search functions\u2026"]',
    );
    await expect(searchInput).toBeVisible();

    // All 3 rows should be visible initially
    const rows = page.locator(".fn-table .body-row");
    await expect(rows).toHaveCount(3);

    // Type the name of the first function to filter
    await searchInput.fill(functions[0].name);
    // Only 1 row should match
    await expect(rows).toHaveCount(1);
    await expect(rows.first()).toContainText(functions[0].name);

    // Clear search — all 3 rows return
    await searchInput.clear();
    await expect(rows).toHaveCount(3);
  });

  /* ------------------------------------------------------------------ */
  /*  2. Kind filter                                                      */
  /* ------------------------------------------------------------------ */
  test("2 - kind filter shows matching rows only", async ({ page }) => {
    const rows = page.locator(".fn-table .body-row");
    await expect(rows).toHaveCount(3);

    // Select "Heavy Job" from the Kind filter dropdown
    const kindSelect = page.locator('select[aria-label="Kind filter"]');
    await expect(kindSelect).toBeVisible();
    await kindSelect.selectOption("Heavy Job");

    // Should show only the heavy-job function
    await expect(rows).toHaveCount(1);
    await expect(rows.first()).toContainText("Heavy Job");

    // Reset to "All kinds"
    await kindSelect.selectOption("All kinds");
    await expect(rows).toHaveCount(3);
  });

  /* ------------------------------------------------------------------ */
  /*  3. Status filter                                                    */
  /* ------------------------------------------------------------------ */
  test("3 - status filter shows matching rows", async ({ page }) => {
    const rows = page.locator(".fn-table .body-row");
    const statusSelect = page.locator('select[aria-label="Status filter"]');
    await expect(statusSelect).toBeVisible();

    // Select a specific status — the count may be 0 or more
    await statusSelect.selectOption("Ready");
    const readyCount = await rows.count();
    // Each visible row should have the "Ready" status badge text
    for (let i = 0; i < readyCount; i++) {
      await expect(rows.nth(i).locator(".status-cell")).toContainText("Ready");
    }

    // Reset
    await statusSelect.selectOption("All statuses");
    await expect(rows).toHaveCount(3);
  });

  /* ------------------------------------------------------------------ */
  /*  4. Name header sort                                                 */
  /* ------------------------------------------------------------------ */
  test("4 - name header sort toggles ascending/descending", async ({
    page,
  }) => {
    const sortBtn = page.locator(".fn-table .header .sortable");
    await expect(sortBtn).toBeVisible();

    // Click sort — get names
    await sortBtn.click();
    const rows = page.locator(".fn-table .body-row .cell-name .nm");
    const count = await rows.count();
    const namesAfterFirst: string[] = [];
    for (let i = 0; i < count; i++) {
      namesAfterFirst.push((await rows.nth(i).textContent()) || "");
    }

    // Click sort again — order should reverse
    await sortBtn.click();
    const namesAfterSecond: string[] = [];
    for (let i = 0; i < count; i++) {
      namesAfterSecond.push((await rows.nth(i).textContent()) || "");
    }

    // The two orderings should be reversed
    expect(namesAfterSecond).toEqual([...namesAfterFirst].reverse());
  });

  /* ------------------------------------------------------------------ */
  /*  5. Row click navigates to function detail                           */
  /* ------------------------------------------------------------------ */
  test("5 - row click navigates to function detail page", async ({ page }) => {
    const firstRow = page.locator(".fn-table .body-row").first();
    await expect(firstRow).toBeVisible();

    // Click the row (which has an onClick that pushes to /functions/{fnId})
    await firstRow.click();

    // URL should contain /functions/ and the function id
    await expect(page).toHaveURL(/\/projects\/[^/]+\/functions\/[^/]+/, {
      timeout: 10000,
    });
  });

  /* ------------------------------------------------------------------ */
  /*  6. Row rebuild button                                               */
  /* ------------------------------------------------------------------ */
  test("6 - row rebuild button is clickable", async ({ page }) => {
    const firstRow = page.locator(".fn-table .body-row").first();
    await expect(firstRow).toBeVisible();

    // Find the Rebuild button (action-btn with aria-label)
    const rebuildBtn = firstRow.locator('button[aria-label="Rebuild"]');
    await expect(rebuildBtn).toBeVisible();

    // Click it — stop propagation so it does not navigate
    await rebuildBtn.click({ force: true });

    // The page should still be on the Functions tab (not navigated)
    await expect(
      page.getByRole("tab", { name: "Functions" }),
    ).toHaveAttribute("aria-selected", "true");
  });

  /* ------------------------------------------------------------------ */
  /*  7. Row delete button removes function                               */
  /* ------------------------------------------------------------------ */
  test("7 - row delete button removes function after confirm", async ({
    page,
  }) => {
    // Create a temporary function to delete
    const tempFn = await createTestFunction(project.id, "to-delete");

    // Navigate to refresh the list
    await page.goto(`/projects/${project.id}`);
    await page.getByRole("tab", { name: "Functions" }).click();
    await expect(page.getByText("Deploy Function").first()).toBeVisible({
      timeout: 15000,
    });

    // Find the row for the temp function
    const targetRow = page.locator(".fn-table .body-row", {
      hasText: tempFn.name,
    });
    await expect(targetRow).toBeVisible();

    // Click the Delete button inside the row
    const deleteBtn = targetRow.locator('button[aria-label="Delete"]');
    await deleteBtn.click({ force: true });

    // A confirm dialog should appear
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Click confirm Delete in the dialog
    await dialog.getByRole("button", { name: "Delete" }).click();

    // Wait for the dialog to close
    await expect(dialog).not.toBeVisible({ timeout: 10000 });

    // The row should no longer exist
    await expect(targetRow).not.toBeVisible({ timeout: 10000 });
  });

  /* ------------------------------------------------------------------ */
  /*  8. Select all checkbox                                              */
  /* ------------------------------------------------------------------ */
  test("8 - select all checkbox checks/unchecks all rows", async ({ page }) => {
    const selectAll = page.locator(
      '.fn-table .header input[aria-label="Select all"]',
    );
    await expect(selectAll).toBeVisible();

    // Check the select-all checkbox
    await selectAll.check({ force: true });

    // All row checkboxes should be checked
    const rowChecks = page.locator(
      '.fn-table .body-row input[aria-label="Select function"]',
    );
    const count = await rowChecks.count();
    for (let i = 0; i < count; i++) {
      await expect(rowChecks.nth(i)).toBeChecked();
    }

    // Uncheck select-all
    await selectAll.uncheck({ force: true });
    for (let i = 0; i < count; i++) {
      await expect(rowChecks.nth(i)).not.toBeChecked();
    }
  });

  /* ------------------------------------------------------------------ */
  /*  9. Individual row checkbox                                          */
  /* ------------------------------------------------------------------ */
  test("9 - individual row checkbox checks only one row", async ({ page }) => {
    const rowChecks = page.locator(
      '.fn-table .body-row input[aria-label="Select function"]',
    );
    const count = await rowChecks.count();
    expect(count).toBeGreaterThanOrEqual(2);

    // Check only the first row
    await rowChecks.first().check({ force: true });
    await expect(rowChecks.first()).toBeChecked();

    // Other rows should remain unchecked
    for (let i = 1; i < count; i++) {
      await expect(rowChecks.nth(i)).not.toBeChecked();
    }

    // Clean up — skip if already unchecked (avoids flaky timeout)
    if (await rowChecks.first().isChecked()) {
      await rowChecks.first().uncheck({ force: true });
    }
  });

  /* ------------------------------------------------------------------ */
  /*  10. Deploy modal dismiss — Cancel / X / scrim                       */
  /* ------------------------------------------------------------------ */
  test("10 - deploy modal dismisses via Cancel, X button, and scrim click", async ({
    page,
  }) => {
    const deployBtn = page.locator("button.btn-primary", {
      hasText: "Deploy Function",
    });
    const dialog = page.locator('.dfm[role="dialog"]');

    // --- Cancel button ---
    await deployBtn.click();
    await expect(dialog).toBeVisible();
    await dialog.locator("button.btn-ghost", { hasText: "Cancel" }).click();
    await expect(dialog).not.toBeVisible();

    // --- X (close) button ---
    await deployBtn.click();
    await expect(dialog).toBeVisible();
    await dialog.locator('button.close[aria-label="Close"]').click();
    await expect(dialog).not.toBeVisible();

    // --- Scrim click ---
    await deployBtn.click();
    await expect(dialog).toBeVisible();
    // Click the scrim (the dfm overlay, not the inner .modal)
    const scrimBox = await dialog.boundingBox();
    expect(scrimBox).toBeTruthy();
    // Click top-left corner which is outside the centered .modal
    await page.mouse.click(scrimBox!.x + 5, scrimBox!.y + 5);
    await expect(dialog).not.toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  11. Deploy modal: name empty → Deploy disabled                      */
  /* ------------------------------------------------------------------ */
  test("11 - deploy modal: empty name disables Deploy button", async ({
    page,
  }) => {
    const deployBtn = page.locator("button.btn-primary", {
      hasText: "Deploy Function",
    });
    await deployBtn.click();

    const dialog = page.locator('.dfm[role="dialog"]');
    await expect(dialog).toBeVisible();

    const submitBtn = dialog.locator("button.btn-primary", {
      hasText: "Deploy",
    });
    const nameInput = dialog.locator("#dfm-fn-name");

    // Initially the name is empty — Deploy should be disabled
    await expect(submitBtn).toBeDisabled();

    // Type a valid name
    await nameInput.fill("test-valid-name");
    await expect(submitBtn).toBeEnabled();

    // Clear the name
    await nameInput.clear();
    await expect(submitBtn).toBeDisabled();

    // Close
    await dialog.locator("button.btn-ghost", { hasText: "Cancel" }).click();
  });

  /* ------------------------------------------------------------------ */
  /*  12. Empty state                                                     */
  /* ------------------------------------------------------------------ */
  test("12 - empty state shows when no functions exist", async ({ page }) => {
    // This test only applies if functions list is empty.
    // Since we have seeded functions, we skip.
    test.skip(
      functions.length > 0,
      "Skipped: seeded functions exist; empty state not reachable without deleting all",
    );
  });

  /* ================================================================== */
  /*  Section B: Deploy modal operation tests (13-30)                    */
  /* ================================================================== */

  /**
   * Helper: open the Deploy Function modal and fill in the name field.
   * Returns a locator for the modal dialog.
   */
  async function openDeployModal(
    page: import("@playwright/test").Page,
    name: string,
  ) {
    const deployBtn = page.locator("button.btn-primary", {
      hasText: "Deploy Function",
    });
    await deployBtn.click();
    const dialog = page.locator('.dfm[role="dialog"]');
    await expect(dialog).toBeVisible();
    await dialog.locator("#dfm-fn-name").fill(name);
    return dialog;
  }

  /**
   * Helper: after a deploy, collect the new function ID for cleanup.
   */
  async function collectNewFunctionId() {
    const allFns = await listFunctions(project.id);
    for (const fn of allFns) {
      if (
        !functions.some((f) => f.id === fn.id) &&
        !deployedFnIds.includes(fn.id)
      ) {
        deployedFnIds.push(fn.id);
      }
    }
  }

  /* ------------------------------------------------------------------ */
  /*  13. Deploy Inline + Light Deploy                                    */
  /* ------------------------------------------------------------------ */
  test("13 - deploy: Inline source + Light Deployment", async ({ page }) => {
    const name = uniqueName("light-inline");
    const dialog = await openDeployModal(page, name);

    // Select Light Deployment kind
    await dialog
      .locator("label.radio-card.light-deploy")
      .click();

    // Source tab: Inline should be default; type some code
    const inlinePane = dialog.locator(".pane.active textarea.code-editor");
    await expect(inlinePane).toBeVisible();
    await inlinePane.fill('def handler(e, c): return {"ok": True}');

    // Deploy
    await dialog.locator("button.btn-primary", { hasText: "Deploy" }).click();
    await expect(dialog).not.toBeVisible({ timeout: 20000 });

    // Verify row appears in table
    await page.waitForTimeout(1000);
    // Re-navigate to refresh
    await page.goto(`/projects/${project.id}`);
    await page.getByRole("tab", { name: "Functions" }).click();
    await expect(page.getByText("Deploy Function").first()).toBeVisible({
      timeout: 15000,
    });
    const newRow = page.locator(".fn-table .body-row", { hasText: name });
    await expect(newRow).toBeVisible({ timeout: 15000 });

    await collectNewFunctionId();
  });

  /* ------------------------------------------------------------------ */
  /*  14. Deploy Git + Heavy Deploy                                       */
  /* ------------------------------------------------------------------ */
  test("14 - deploy: Git source + Heavy Deployment", async ({ page }) => {
    const name = uniqueName("heavy-git");
    const dialog = await openDeployModal(page, name);

    // Select Heavy Deployment kind
    await dialog
      .locator("label.radio-card.heavy-deploy")
      .click();

    // Switch to Git source tab
    const gitTabBtn = dialog.locator(".segmented button", { hasText: "Git" });
    await gitTabBtn.click();

    // Fill Git fields
    await dialog
      .locator("#dfm-git-url")
      .fill("https://github.com/acme/functions.git");
    await dialog.locator("#dfm-git-branch").clear();
    await dialog.locator("#dfm-git-branch").fill("main");

    // Deploy
    await dialog.locator("button.btn-primary", { hasText: "Deploy" }).click();
    await expect(dialog).not.toBeVisible({ timeout: 20000 });

    // Verify row appears
    await page.goto(`/projects/${project.id}`);
    await page.getByRole("tab", { name: "Functions" }).click();
    await expect(page.getByText("Deploy Function").first()).toBeVisible({
      timeout: 15000,
    });
    const newRow = page.locator(".fn-table .body-row", { hasText: name });
    await expect(newRow).toBeVisible({ timeout: 15000 });

    await collectNewFunctionId();
  });

  /* ------------------------------------------------------------------ */
  /*  15. Deploy Zip + Heavy Job                                          */
  /* ------------------------------------------------------------------ */
  test("15 - deploy: Zip source + Heavy Job", async ({ page }) => {
    const name = uniqueName("heavy-zip");
    const dialog = await openDeployModal(page, name);

    // Select Heavy Job kind
    await dialog
      .locator("label.radio-card.heavy-job")
      .click();

    // Switch to Zip source tab
    const zipTabBtn = dialog.locator(".segmented button", { hasText: "Zip" });
    await zipTabBtn.click();

    // Fill zip path
    await dialog
      .locator("#dfm-zip-path")
      .fill("s3://builds/my-function-v1.zip");

    // Deploy
    await dialog.locator("button.btn-primary", { hasText: "Deploy" }).click();
    await expect(dialog).not.toBeVisible({ timeout: 20000 });

    // Verify row
    await page.goto(`/projects/${project.id}`);
    await page.getByRole("tab", { name: "Functions" }).click();
    await expect(page.getByText("Deploy Function").first()).toBeVisible({
      timeout: 15000,
    });
    const newRow = page.locator(".fn-table .body-row", { hasText: name });
    await expect(newRow).toBeVisible({ timeout: 15000 });

    await collectNewFunctionId();
  });

  /* ------------------------------------------------------------------ */
  /*  16. Deploy Async mode                                               */
  /* ------------------------------------------------------------------ */
  test("16 - deploy: Async mode appears in table", async ({ page }) => {
    const name = uniqueName("async-fn");
    const dialog = await openDeployModal(page, name);

    // Select Async mode via the segmented control
    await dialog
      .locator('.segmented button', { hasText: "Async" })
      .first()
      .click();

    // Deploy
    await dialog.locator("button.btn-primary", { hasText: "Deploy" }).click();
    await expect(dialog).not.toBeVisible({ timeout: 20000 });

    // Verify mode in table
    await page.goto(`/projects/${project.id}`);
    await page.getByRole("tab", { name: "Functions" }).click();
    await expect(page.getByText("Deploy Function").first()).toBeVisible({
      timeout: 15000,
    });
    const newRow = page.locator(".fn-table .body-row", { hasText: name });
    await expect(newRow).toBeVisible({ timeout: 15000 });
    await expect(newRow.locator(".mode-cell")).toContainText("async");

    await collectNewFunctionId();
  });

  /* ------------------------------------------------------------------ */
  /*  17. Deploy Stream mode                                              */
  /* ------------------------------------------------------------------ */
  test("17 - deploy: Stream mode appears in table", async ({ page }) => {
    const name = uniqueName("stream-fn");
    const dialog = await openDeployModal(page, name);

    // Select Stream mode
    await dialog
      .locator('.segmented button', { hasText: "Stream" })
      .first()
      .click();

    // Deploy
    await dialog.locator("button.btn-primary", { hasText: "Deploy" }).click();
    await expect(dialog).not.toBeVisible({ timeout: 20000 });

    // Verify mode in table
    await page.goto(`/projects/${project.id}`);
    await page.getByRole("tab", { name: "Functions" }).click();
    await expect(page.getByText("Deploy Function").first()).toBeVisible({
      timeout: 15000,
    });
    const newRow = page.locator(".fn-table .body-row", { hasText: name });
    await expect(newRow).toBeVisible({ timeout: 15000 });
    await expect(newRow.locator(".mode-cell")).toContainText("stream");

    await collectNewFunctionId();
  });

  /* ------------------------------------------------------------------ */
  /*  18. Deploy Custom Runtime                                           */
  /* ------------------------------------------------------------------ */
  test("18 - deploy: Custom Dockerfile runtime", async ({ page }) => {
    const name = uniqueName("custom-rt");
    const dialog = await openDeployModal(page, name);

    // Switch runtime tab to Custom Dockerfile
    const customBtn = dialog.locator(".segmented button", {
      hasText: "Custom Dockerfile",
    });
    await customBtn.click();

    // Fill the Dockerfile textarea
    await dialog
      .locator("#dfm-custom-dockerfile")
      .fill('FROM python:3.12-slim\nCMD ["python", "main.py"]');

    // Deploy
    await dialog.locator("button.btn-primary", { hasText: "Deploy" }).click();
    await expect(dialog).not.toBeVisible({ timeout: 20000 });

    // Verify row appears
    await page.goto(`/projects/${project.id}`);
    await page.getByRole("tab", { name: "Functions" }).click();
    await expect(page.getByText("Deploy Function").first()).toBeVisible({
      timeout: 15000,
    });
    const newRow = page.locator(".fn-table .body-row", { hasText: name });
    await expect(newRow).toBeVisible({ timeout: 15000 });

    await collectNewFunctionId();
  });

  /* ------------------------------------------------------------------ */
  /*  19. Deploy with timeout change                                      */
  /* ------------------------------------------------------------------ */
  test("19 - deploy: custom timeout value", async ({ page }) => {
    const name = uniqueName("timeout-fn");
    const dialog = await openDeployModal(page, name);

    // Select Heavy Job (allows larger timeout)
    await dialog
      .locator("label.radio-card.heavy-job")
      .click();

    // Change timeout
    const timeoutInput = dialog.locator("#dfm-timeout");
    await timeoutInput.clear();
    await timeoutInput.fill("120");

    // Deploy
    await dialog.locator("button.btn-primary", { hasText: "Deploy" }).click();
    await expect(dialog).not.toBeVisible({ timeout: 20000 });

    // Verify row
    await page.goto(`/projects/${project.id}`);
    await page.getByRole("tab", { name: "Functions" }).click();
    await expect(page.getByText("Deploy Function").first()).toBeVisible({
      timeout: 15000,
    });
    const newRow = page.locator(".fn-table .body-row", { hasText: name });
    await expect(newRow).toBeVisible({ timeout: 15000 });

    await collectNewFunctionId();
  });

  /* ------------------------------------------------------------------ */
  /*  20. Deploy with GPU enabled                                         */
  /* ------------------------------------------------------------------ */
  test("20 - deploy: GPU acceleration enabled", async ({ page }) => {
    const name = uniqueName("gpu-fn");
    const dialog = await openDeployModal(page, name);

    // Select Heavy Deployment (GPU-compatible)
    await dialog
      .locator("label.radio-card.heavy-deploy")
      .click();

    // Enable GPU toggle
    const gpuSwitch = dialog.locator(
      '.section:has(h3:text("Timeout")) input.switch',
    );
    await gpuSwitch.check({ force: true });

    // GPU config should appear
    const gpuConfig = dialog.locator(".gpu-config.open");
    await expect(gpuConfig).toBeVisible();

    // Select GPU type
    await dialog.locator("#dfm-gpu-type").selectOption("A100");
    await dialog.locator("#dfm-gpu-provider").selectOption("runpod");

    // Deploy
    await dialog.locator("button.btn-primary", { hasText: "Deploy" }).click();
    await expect(dialog).not.toBeVisible({ timeout: 20000 });

    // Verify row
    await page.goto(`/projects/${project.id}`);
    await page.getByRole("tab", { name: "Functions" }).click();
    await expect(page.getByText("Deploy Function").first()).toBeVisible({
      timeout: 15000,
    });
    const newRow = page.locator(".fn-table .body-row", { hasText: name });
    await expect(newRow).toBeVisible({ timeout: 15000 });

    await collectNewFunctionId();
  });

  /* ------------------------------------------------------------------ */
  /*  21. Deploy with DB trigger                                          */
  /* ------------------------------------------------------------------ */
  test("21 - deploy: with Database trigger", async ({ page }) => {
    const name = uniqueName("db-trig-fn");
    const dialog = await openDeployModal(page, name);

    // Add a trigger
    await dialog.locator("button.add-btn", { hasText: "Add trigger" }).click();

    // A trigger card should appear, default type is "Database change"
    const triggerCard = dialog.locator(".trigger-card").first();
    await expect(triggerCard).toBeVisible();

    // Ensure DB type is selected
    await triggerCard
      .locator(".segmented button", { hasText: "Database change" })
      .click();

    // Fill table name
    await triggerCard.locator('input[placeholder="public.orders"]').fill("public.users");

    // Select INSERT event
    await triggerCard
      .locator('.check-pill', { hasText: "INSERT" })
      .click();

    // Deploy
    await dialog.locator("button.btn-primary", { hasText: "Deploy" }).click();
    await expect(dialog).not.toBeVisible({ timeout: 20000 });

    // Verify row
    await page.goto(`/projects/${project.id}`);
    await page.getByRole("tab", { name: "Functions" }).click();
    await expect(page.getByText("Deploy Function").first()).toBeVisible({
      timeout: 15000,
    });
    const newRow = page.locator(".fn-table .body-row", { hasText: name });
    await expect(newRow).toBeVisible({ timeout: 15000 });

    await collectNewFunctionId();
  });

  /* ------------------------------------------------------------------ */
  /*  22. Deploy with Storage trigger                                     */
  /* ------------------------------------------------------------------ */
  test("22 - deploy: with Object Storage trigger", async ({ page }) => {
    const name = uniqueName("stor-trig-fn");
    const dialog = await openDeployModal(page, name);

    // Add a trigger
    await dialog.locator("button.add-btn", { hasText: "Add trigger" }).click();
    const triggerCard = dialog.locator(".trigger-card").first();
    await expect(triggerCard).toBeVisible();

    // Switch to Object storage type
    await triggerCard
      .locator(".segmented button", { hasText: "Object storage" })
      .click();

    // Fill bucket
    await triggerCard.locator('input[placeholder="assets"]').fill("uploads");

    // Select ObjectCreated event
    await triggerCard
      .locator('.check-pill', { hasText: "ObjectCreated" })
      .click();

    // Fill prefix
    await triggerCard.locator('input[placeholder="uploads/"]').fill("images/");

    // Deploy
    await dialog.locator("button.btn-primary", { hasText: "Deploy" }).click();
    await expect(dialog).not.toBeVisible({ timeout: 20000 });

    // Verify row
    await page.goto(`/projects/${project.id}`);
    await page.getByRole("tab", { name: "Functions" }).click();
    await expect(page.getByText("Deploy Function").first()).toBeVisible({
      timeout: 15000,
    });
    const newRow = page.locator(".fn-table .body-row", { hasText: name });
    await expect(newRow).toBeVisible({ timeout: 15000 });

    await collectNewFunctionId();
  });

  /* ------------------------------------------------------------------ */
  /*  23. Deploy with multiple triggers                                   */
  /* ------------------------------------------------------------------ */
  test("23 - deploy: with DB + Storage triggers", async ({ page }) => {
    const name = uniqueName("multi-trig");
    const dialog = await openDeployModal(page, name);

    // Add first trigger (DB)
    await dialog.locator("button.add-btn", { hasText: "Add trigger" }).click();
    const trigger1 = dialog.locator(".trigger-card").nth(0);
    await trigger1
      .locator(".segmented button", { hasText: "Database change" })
      .click();
    await trigger1.locator('input[placeholder="public.orders"]').fill("public.items");
    await trigger1.locator('.check-pill', { hasText: "INSERT" }).click();

    // Add second trigger (Storage)
    await dialog.locator("button.add-btn", { hasText: "Add trigger" }).click();
    const trigger2 = dialog.locator(".trigger-card").nth(1);
    await trigger2
      .locator(".segmented button", { hasText: "Object storage" })
      .click();
    await trigger2.locator('input[placeholder="assets"]').fill("docs");
    await trigger2.locator('.check-pill', { hasText: "ObjectCreated" }).click();

    // Verify 2 trigger cards visible
    await expect(dialog.locator(".trigger-card")).toHaveCount(2);

    // Deploy
    await dialog.locator("button.btn-primary", { hasText: "Deploy" }).click();
    await expect(dialog).not.toBeVisible({ timeout: 20000 });

    // Verify row
    await page.goto(`/projects/${project.id}`);
    await page.getByRole("tab", { name: "Functions" }).click();
    await expect(page.getByText("Deploy Function").first()).toBeVisible({
      timeout: 15000,
    });
    const newRow = page.locator(".fn-table .body-row", { hasText: name });
    await expect(newRow).toBeVisible({ timeout: 15000 });

    await collectNewFunctionId();
  });

  /* ------------------------------------------------------------------ */
  /*  24. Trigger add → remove → re-add                                   */
  /* ------------------------------------------------------------------ */
  test("24 - trigger: add, remove, and re-add changes count", async ({
    page,
  }) => {
    const deployBtn = page.locator("button.btn-primary", {
      hasText: "Deploy Function",
    });
    await deployBtn.click();
    const dialog = page.locator('.dfm[role="dialog"]');
    await expect(dialog).toBeVisible();

    const triggerCards = dialog.locator(".trigger-card");

    // Initially: 0 triggers
    await expect(triggerCards).toHaveCount(0);

    // Add a trigger
    await dialog.locator("button.add-btn", { hasText: "Add trigger" }).click();
    await expect(triggerCards).toHaveCount(1);

    // Remove it
    await triggerCards
      .first()
      .locator('button[aria-label="Remove trigger"]')
      .click();
    await expect(triggerCards).toHaveCount(0);

    // Re-add
    await dialog.locator("button.add-btn", { hasText: "Add trigger" }).click();
    await expect(triggerCards).toHaveCount(1);

    // Close modal
    await dialog.locator("button.btn-ghost", { hasText: "Cancel" }).click();
  });

  /* ------------------------------------------------------------------ */
  /*  25. Deploy with literal env var                                     */
  /* ------------------------------------------------------------------ */
  test("25 - deploy: with literal environment variable", async ({ page }) => {
    const name = uniqueName("env-lit-fn");
    const dialog = await openDeployModal(page, name);

    // Add an environment variable
    await dialog.locator("button.add-btn", { hasText: "Add variable" }).click();
    const envRow = dialog.locator(".env-row").first();
    await expect(envRow).toBeVisible();

    // Fill name and value
    await envRow.locator('input[placeholder="MY_VAR"]').fill("APP_ENV");
    await envRow.locator('input[placeholder="my-value"]').fill("production");

    // Deploy
    await dialog.locator("button.btn-primary", { hasText: "Deploy" }).click();
    await expect(dialog).not.toBeVisible({ timeout: 20000 });

    // Verify row
    await page.goto(`/projects/${project.id}`);
    await page.getByRole("tab", { name: "Functions" }).click();
    await expect(page.getByText("Deploy Function").first()).toBeVisible({
      timeout: 15000,
    });
    const newRow = page.locator(".fn-table .body-row", { hasText: name });
    await expect(newRow).toBeVisible({ timeout: 15000 });

    await collectNewFunctionId();
  });

  /* ------------------------------------------------------------------ */
  /*  26. Deploy with secret ref env var                                  */
  /* ------------------------------------------------------------------ */
  test("26 - deploy: with secret-referenced environment variable", async ({
    page,
  }) => {
    const name = uniqueName("env-sec-fn");
    const dialog = await openDeployModal(page, name);

    // Add env var
    await dialog.locator("button.add-btn", { hasText: "Add variable" }).click();
    const envRow = dialog.locator(".env-row").first();
    await expect(envRow).toBeVisible();

    // Set name
    await envRow.locator('input[placeholder="MY_VAR"]').fill("DB_PASS");

    // Switch source to Secret
    await envRow.locator(".source-seg button", { hasText: "Secret" }).click();

    // Select a secret reference
    const secretSelect = envRow.locator("select");
    await expect(secretSelect).toBeVisible();
    await secretSelect.selectOption("DATABASE_URL");

    // Deploy
    await dialog.locator("button.btn-primary", { hasText: "Deploy" }).click();
    await expect(dialog).not.toBeVisible({ timeout: 20000 });

    // Verify row
    await page.goto(`/projects/${project.id}`);
    await page.getByRole("tab", { name: "Functions" }).click();
    await expect(page.getByText("Deploy Function").first()).toBeVisible({
      timeout: 15000,
    });
    const newRow = page.locator(".fn-table .body-row", { hasText: name });
    await expect(newRow).toBeVisible({ timeout: 15000 });

    await collectNewFunctionId();
  });

  /* ------------------------------------------------------------------ */
  /*  27. Deploy with multiple env vars                                   */
  /* ------------------------------------------------------------------ */
  test("27 - deploy: with multiple environment variables", async ({ page }) => {
    const name = uniqueName("env-multi");
    const dialog = await openDeployModal(page, name);

    // Add 3 environment variables
    for (let i = 0; i < 3; i++) {
      await dialog
        .locator("button.add-btn", { hasText: "Add variable" })
        .click();
    }
    await expect(dialog.locator(".env-row")).toHaveCount(3);

    // Fill each one
    const envRows = dialog.locator(".env-row");
    await envRows.nth(0).locator('input[placeholder="MY_VAR"]').fill("VAR_A");
    await envRows.nth(0).locator('input[placeholder="my-value"]').fill("alpha");
    await envRows.nth(1).locator('input[placeholder="MY_VAR"]').fill("VAR_B");
    await envRows.nth(1).locator('input[placeholder="my-value"]').fill("beta");
    await envRows.nth(2).locator('input[placeholder="MY_VAR"]').fill("VAR_C");
    await envRows.nth(2).locator('input[placeholder="my-value"]').fill("gamma");

    // Deploy
    await dialog.locator("button.btn-primary", { hasText: "Deploy" }).click();
    await expect(dialog).not.toBeVisible({ timeout: 20000 });

    // Verify row
    await page.goto(`/projects/${project.id}`);
    await page.getByRole("tab", { name: "Functions" }).click();
    await expect(page.getByText("Deploy Function").first()).toBeVisible({
      timeout: 15000,
    });
    const newRow = page.locator(".fn-table .body-row", { hasText: name });
    await expect(newRow).toBeVisible({ timeout: 15000 });

    await collectNewFunctionId();
  });

  /* ------------------------------------------------------------------ */
  /*  28. Env var add → remove changes count                              */
  /* ------------------------------------------------------------------ */
  test("28 - env var: add and remove changes count", async ({ page }) => {
    const deployBtn = page.locator("button.btn-primary", {
      hasText: "Deploy Function",
    });
    await deployBtn.click();
    const dialog = page.locator('.dfm[role="dialog"]');
    await expect(dialog).toBeVisible();

    const envRows = dialog.locator(".env-row");

    // Initially: 0 env vars
    await expect(envRows).toHaveCount(0);

    // Add two
    await dialog.locator("button.add-btn", { hasText: "Add variable" }).click();
    await dialog.locator("button.add-btn", { hasText: "Add variable" }).click();
    await expect(envRows).toHaveCount(2);

    // Remove the first
    await envRows
      .first()
      .locator('button[aria-label="Remove variable"]')
      .click();
    await expect(envRows).toHaveCount(1);

    // Remove the last
    await envRows
      .first()
      .locator('button[aria-label="Remove variable"]')
      .click();
    await expect(envRows).toHaveCount(0);

    // Close modal
    await dialog.locator("button.btn-ghost", { hasText: "Cancel" }).click();
  });

  /* ------------------------------------------------------------------ */
  /*  29. Deploy with all fields filled                                   */
  /* ------------------------------------------------------------------ */
  test("29 - deploy: full configuration with all fields", async ({ page }) => {
    const name = uniqueName("full-cfg");
    const dialog = await openDeployModal(page, name);

    // Display name
    await dialog.locator("#dfm-fn-display").fill("Full Config Test");

    // Kind: Heavy Deployment
    await dialog
      .locator("label.radio-card.heavy-deploy")
      .click();

    // Mode: Async
    await dialog
      .locator('.segmented button', { hasText: "Async" })
      .first()
      .click();

    // Source: Git
    await dialog.locator(".segmented button", { hasText: "Git" }).first().click();
    await dialog
      .locator("#dfm-git-url")
      .fill("https://github.com/acme/full-test.git");
    await dialog.locator("#dfm-git-branch").clear();
    await dialog.locator("#dfm-git-branch").fill("develop");

    // Runtime: Custom Dockerfile
    await dialog
      .locator(".segmented button", { hasText: "Custom Dockerfile" })
      .click();
    await dialog
      .locator("#dfm-custom-dockerfile")
      .fill('FROM node:20\nCMD ["node", "index.js"]');

    // Timeout: 120s
    const timeoutInput = dialog.locator("#dfm-timeout");
    await timeoutInput.clear();
    await timeoutInput.fill("120");

    // GPU: enable
    const gpuSwitch = dialog.locator(
      '.section:has(h3:text("Timeout")) input.switch',
    );
    await gpuSwitch.check({ force: true });
    await dialog.locator("#dfm-gpu-type").selectOption("H100");

    // Trigger: DB
    await dialog.locator("button.add-btn", { hasText: "Add trigger" }).click();
    const triggerCard = dialog.locator(".trigger-card").first();
    await triggerCard
      .locator(".segmented button", { hasText: "Database change" })
      .click();
    await triggerCard
      .locator('input[placeholder="public.orders"]')
      .fill("public.events");
    await triggerCard.locator('.check-pill', { hasText: "INSERT" }).click();
    await triggerCard.locator('.check-pill', { hasText: "UPDATE" }).click();

    // Env vars: 2
    await dialog.locator("button.add-btn", { hasText: "Add variable" }).click();
    await dialog.locator("button.add-btn", { hasText: "Add variable" }).click();
    const envRows = dialog.locator(".env-row");
    await envRows.nth(0).locator('input[placeholder="MY_VAR"]').fill("NODE_ENV");
    await envRows
      .nth(0)
      .locator('input[placeholder="my-value"]')
      .fill("production");
    await envRows.nth(1).locator('input[placeholder="MY_VAR"]').fill("API_KEY");
    await envRows
      .nth(1)
      .locator(".source-seg button", { hasText: "Secret" })
      .click();

    // Deploy
    await dialog.locator("button.btn-primary", { hasText: "Deploy" }).click();
    await expect(dialog).not.toBeVisible({ timeout: 20000 });

    // Verify row
    await page.goto(`/projects/${project.id}`);
    await page.getByRole("tab", { name: "Functions" }).click();
    await expect(page.getByText("Deploy Function").first()).toBeVisible({
      timeout: 15000,
    });
    const newRow = page.locator(".fn-table .body-row", { hasText: name });
    await expect(newRow).toBeVisible({ timeout: 15000 });

    // Verify kind and mode in the row
    await expect(newRow.locator(".kind-badge")).toContainText("Heavy");
    await expect(newRow.locator(".mode-cell")).toContainText("async");

    await collectNewFunctionId();
  });

  /* ------------------------------------------------------------------ */
  /*  30. Kind radio UI change — description text updates                 */
  /* ------------------------------------------------------------------ */
  test("30 - kind radio cards change description text on selection", async ({
    page,
  }) => {
    const deployBtn = page.locator("button.btn-primary", {
      hasText: "Deploy Function",
    });
    await deployBtn.click();
    const dialog = page.locator('.dfm[role="dialog"]');
    await expect(dialog).toBeVisible();

    // Click Heavy Job
    await dialog.locator("label.radio-card.heavy-job").click();
    const heavyJobRadio = dialog.locator(
      'input[type="radio"][value="heavy-job"]',
    );
    await expect(heavyJobRadio).toBeChecked();
    // Description should mention batch
    await expect(
      dialog.locator("label.radio-card.heavy-job .desc"),
    ).toContainText("Batch");

    // Click Heavy Deployment
    await dialog.locator("label.radio-card.heavy-deploy").click();
    const heavyDeployRadio = dialog.locator(
      'input[type="radio"][value="heavy-deployment"]',
    );
    await expect(heavyDeployRadio).toBeChecked();
    // Description should mention GPU or ML
    await expect(
      dialog.locator("label.radio-card.heavy-deploy .desc"),
    ).toContainText("GPU");

    // Click Light Deployment
    await dialog.locator("label.radio-card.light-deploy").click();
    const lightRadio = dialog.locator(
      'input[type="radio"][value="light-deployment"]',
    );
    await expect(lightRadio).toBeChecked();
    // Description should mention webhooks or fast
    await expect(
      dialog.locator("label.radio-card.light-deploy .desc"),
    ).toContainText("Webhooks");

    // Close
    await dialog.locator("button.btn-ghost", { hasText: "Cancel" }).click();
  });
});

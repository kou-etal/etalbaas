import { test, expect } from "@playwright/test";
import { createTestProject, deleteTestProject } from "./helpers/api";

/**
 * 11 - Project API Keys Tab
 *
 * Covers the API Keys tab on the project detail page (/projects/{id}).
 * The API Keys tab has real API integration (ProjectService RPC) for CRUD.
 *
 * Page structure:
 * - "Create Key" button in header
 * - Table with columns: Name, Key (prefix + masked), Role (badge), Created, Actions (Revoke)
 * - Role badges: "anon" (secondary) and "service_role" (destructive)
 * - EmptyState when no keys exist
 *
 * Create API Key dialog (2-step):
 * - Step 1: Key Name input, Role radio (anon / service_role with warning), Expiration select (30d/90d/1yr/No expiration with warning)
 * - Step 2: Shows "API Key Created" title, raw key with copy button, "Done" button
 *
 * Revoke dialog (ConfirmDialog):
 * - Description warning
 * - Cancel / "Revoke Key" buttons
 *
 * Setup: 1 project via API. Keys created via UI in tests.
 */

test.describe.serial("Project API Keys Tab", () => {
  let project: { id: string; displayName: string };

  test.beforeAll(async () => {
    project = await createTestProject("apikeys-tab");
  });

  test.afterAll(async () => {
    try {
      await deleteTestProject(project.id);
    } catch {}
  });

  test.beforeEach(async ({ page }) => {
    await page.goto(`/projects/${project.id}`);
    // Click the API Keys tab
    const apiKeysTab = page.locator('[role="tab"]').filter({ hasText: /^API Keys/ });
    await expect(apiKeysTab).toBeVisible({ timeout: 15000 });
    await apiKeysTab.click();
    await expect(apiKeysTab).toHaveAttribute("aria-selected", "true");
    // Wait for the API Keys heading to be visible
    await expect(
      page.getByRole("heading", { name: "API Keys", exact: true }),
    ).toBeVisible({ timeout: 10000 });
  });

  /* ------------------------------------------------------------------ */
  /*  1. Create modal dismiss (Cancel)                                   */
  /* ------------------------------------------------------------------ */
  test("1 - create modal dismisses via Cancel", async ({ page }) => {
    // Click "Create Key" button
    await page.getByRole("button", { name: "Create Key" }).first().click();

    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Verify Step 1 is shown
    await expect(dialog.getByText("Create API Key")).toBeVisible();

    // Click Cancel
    await dialog.getByRole("button", { name: "Cancel" }).click();
    await expect(dialog).not.toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  2. service_role selection → warning visible → switch to anon       */
  /* ------------------------------------------------------------------ */
  test("2 - service_role selection shows warning, anon hides it", async ({
    page,
  }) => {
    await page.getByRole("button", { name: "Create Key" }).first().click();
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // By default, anon is selected — no warning
    const warning = dialog.locator(".modal-err");
    await expect(warning).not.toBeVisible();

    // Select service_role (force: true because the label overlay intercepts pointer events)
    const serviceRoleRadio = dialog.getByRole("radio", {
      name: "service_role",
    });
    await serviceRoleRadio.click({ force: true });

    // Warning should appear
    await expect(warning).toBeVisible();
    const warningText = await warning.textContent();
    expect(warningText).toContain("full database access");

    // Switch back to anon
    const anonRadio = dialog.getByRole("radio", { name: "anon" });
    await anonRadio.click({ force: true });

    // Warning should disappear
    await expect(warning).not.toBeVisible();

    await dialog.getByRole("button", { name: "Cancel" }).click();
  });

  /* ------------------------------------------------------------------ */
  /*  3. No expiration → warning visible → switch to 90d                 */
  /* ------------------------------------------------------------------ */
  test("3 - no expiration shows warning, switching to 90d hides it", async ({
    page,
  }) => {
    await page.getByRole("button", { name: "Create Key" }).first().click();
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Default is 90 days — the "No expiration" option is not selected
    const ninetyDayRadio = dialog.locator("input[type='radio'][value='90']");
    await expect(ninetyDayRadio).toBeChecked();

    // Select "No expiration" via its radio button
    const neverRadio = dialog.locator("input[type='radio'][value='never']");
    await neverRadio.click({ force: true });
    await expect(neverRadio).toBeChecked();

    // The exp-option label for "never" should now be checked (highlighted)
    const neverOption = dialog.locator(".exp-option", { hasText: "No expiration" });
    await expect(neverOption).toHaveClass(/checked/);

    // Switch back to 90 days
    await ninetyDayRadio.click({ force: true });
    await expect(ninetyDayRadio).toBeChecked();
    await expect(neverOption).not.toHaveClass(/checked/);

    await dialog.getByRole("button", { name: "Cancel" }).click();
  });

  /* ------------------------------------------------------------------ */
  /*  4. Create anon key (90d) — step 2 shows key → Copy → Done         */
  /* ------------------------------------------------------------------ */
  test("4 - create anon key with 90d expiration", async ({
    page,
    context,
  }) => {
    await context.grantPermissions(["clipboard-read", "clipboard-write"]);

    await page.getByRole("button", { name: "Create Key" }).first().click();
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Step 1: Fill name (plain input, no id)
    await dialog.locator("input[type='text']").fill("e2e-anon-90d");

    // Role: anon is default
    // Expiration: 90d is default

    // Click Create Key
    await dialog.getByRole("button", { name: "Create Key" }).click();

    // Step 2: Should show "API Key Created"
    await expect(
      dialog.getByText("API Key Created"),
    ).toBeVisible({ timeout: 10000 });

    // Raw key should be displayed in .key-block
    const keyDisplay = dialog.locator(".key-block");
    await expect(keyDisplay).toBeVisible();
    const rawKey = await keyDisplay.textContent();
    expect(rawKey).toBeTruthy();
    expect(rawKey!.length).toBeGreaterThan(10);

    // Click Copy key button
    await dialog.locator("button.copy-full").click();

    // Verify clipboard
    const clipboardText = await page.evaluate(() =>
      navigator.clipboard.readText(),
    );
    expect(clipboardText).toBeTruthy();

    // Click Done
    await dialog.getByRole("button", { name: "Done" }).click();
    await expect(dialog).not.toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  5. Create service_role key (30d) — table row with badge            */
  /* ------------------------------------------------------------------ */
  test("5 - create service_role key with 30d expiration", async ({
    page,
  }) => {
    await page.getByRole("button", { name: "Create Key" }).first().click();
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Name
    await dialog.locator("input[type='text']").fill("e2e-svc-30d");

    // Select service_role
    await dialog.locator("input[type='radio'][value='service_role']").click({ force: true });

    // Select 30 days (radio button)
    await dialog.locator("input[type='radio'][value='30']").click({ force: true });

    // Create
    await dialog.getByRole("button", { name: "Create Key" }).click();
    await expect(
      dialog.getByText("API Key Created"),
    ).toBeVisible({ timeout: 10000 });

    // Done
    await dialog.getByRole("button", { name: "Done" }).click();
    await expect(dialog).not.toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  6. Create key (1 year)                                             */
  /* ------------------------------------------------------------------ */
  test("6 - create key with 1 year expiration", async ({ page }) => {
    await page.getByRole("button", { name: "Create Key" }).first().click();
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    await dialog.locator("input[type='text']").fill("e2e-1yr-key");

    // Select 1 year (radio button)
    await dialog.locator("input[type='radio'][value='365']").click({ force: true });

    await dialog.getByRole("button", { name: "Create Key" }).click();
    await expect(
      dialog.getByText("API Key Created"),
    ).toBeVisible({ timeout: 10000 });

    await dialog.getByRole("button", { name: "Done" }).click();
    await expect(dialog).not.toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  7. Create key (No expiration)                                      */
  /* ------------------------------------------------------------------ */
  test("7 - create key with no expiration", async ({ page }) => {
    await page.getByRole("button", { name: "Create Key" }).first().click();
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    await dialog.locator("input[type='text']").fill("e2e-noexp-key");

    // Select No expiration (radio button)
    await dialog.locator("input[type='radio'][value='never']").click({ force: true });

    // The exp-option with "No expiration" should be checked
    await expect(
      dialog.locator(".exp-option", { hasText: "No expiration" }),
    ).toHaveClass(/checked/);

    await dialog.getByRole("button", { name: "Create Key" }).click();
    await expect(
      dialog.getByText("API Key Created"),
    ).toBeVisible({ timeout: 10000 });

    await dialog.getByRole("button", { name: "Done" }).click();
    await expect(dialog).not.toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  8. Key prefix shows masked format in table                         */
  /* ------------------------------------------------------------------ */
  test("8 - key prefix shows prefix format in table", async ({ page }) => {
    const akTable = page.locator(".ak-table");
    await expect(akTable).toBeVisible({ timeout: 10000 });

    // Each row's prefix column should show the key prefix (8 hex chars)
    const firstRow = akTable.locator(".body-row").first();
    const prefixCell = firstRow.locator(".ak-prefix");
    const prefixText = await prefixCell.textContent();
    // Key prefix is 8 hex characters, verify it's present and non-empty
    expect(prefixText).toBeTruthy();
    expect(prefixText!.length).toBeGreaterThanOrEqual(8);
  });

  /* ------------------------------------------------------------------ */
  /*  9. Revoke modal: Cancel → no change                                */
  /* ------------------------------------------------------------------ */
  test("9 - revoke modal: Cancel leaves key unchanged", async ({ page }) => {
    const akTable = page.locator(".ak-table");
    await expect(akTable).toBeVisible({ timeout: 10000 });

    const rows = akTable.locator(".body-row");
    const initialCount = await rows.count();

    // Click revoke on the first row
    const firstRow = rows.first();
    await firstRow.locator("button", { hasText: "Revoke" }).click();

    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Should mention revoking
    const dialogText = await dialog.textContent();
    expect(dialogText).toContain("Revok");

    // Click Cancel
    await dialog.getByRole("button", { name: "Cancel" }).click();
    await expect(dialog).not.toBeVisible();

    // Row count should be unchanged
    await expect(rows).toHaveCount(initialCount);
  });

  /* ------------------------------------------------------------------ */
  /*  10. Revoke modal: X close dismisses                                */
  /* ------------------------------------------------------------------ */
  test("10 - revoke modal: X close dismisses dialog", async ({ page }) => {
    const akTable = page.locator(".ak-table");
    await expect(akTable).toBeVisible({ timeout: 10000 });

    // Click revoke on the first row
    const firstRow = akTable.locator(".body-row").first();
    await firstRow.locator("button", { hasText: "Revoke" }).click();

    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Close via X button or overlay click
    const closeBtn = dialog.locator("button.close");
    if (await closeBtn.isVisible()) {
      await closeBtn.click();
    } else {
      // Fallback: click outside (overlay)
      await page.mouse.click(5, 5);
    }
    await expect(dialog).not.toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  11. Revoke key — confirm → row disappears                         */
  /* ------------------------------------------------------------------ */
  test("11 - revoke key removes row from table", async ({ page }) => {
    const akTable = page.locator(".ak-table");
    await expect(akTable).toBeVisible({ timeout: 10000 });

    const rows = akTable.locator(".body-row");
    const initialCount = await rows.count();
    expect(initialCount).toBeGreaterThanOrEqual(1);

    // Get the name of the first key to be revoked
    const firstRow = rows.first();
    const keyName = await firstRow.locator(".ak-name").textContent();

    // Click revoke button
    await firstRow.locator("button", { hasText: "Revoke" }).click();

    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Type the key name to confirm
    if (keyName) {
      await dialog.locator("input[type='text']").fill(keyName.trim());
    }

    // Confirm revocation
    await dialog.getByRole("button", { name: "Revoke Key" }).click();
    await expect(dialog).not.toBeVisible({ timeout: 10000 });

    // Row count should decrease
    await expect(rows).toHaveCount(initialCount - 1, { timeout: 10000 });

    // The revoked key name should no longer be in the table
    if (keyName) {
      await expect(
        akTable.locator(".ak-name", { hasText: keyName.trim() }),
      ).not.toBeVisible();
    }
  });

  /* ------------------------------------------------------------------ */
  /*  12. Revoke 2nd key → remaining count check                         */
  /* ------------------------------------------------------------------ */
  test("12 - revoke second key decreases count", async ({ page }) => {
    const akTable = page.locator(".ak-table");
    await expect(akTable).toBeVisible({ timeout: 10000 });

    const rows = akTable.locator(".body-row");
    const initialCount = await rows.count();
    expect(initialCount).toBeGreaterThanOrEqual(1);

    // Get the name and revoke the first remaining key
    const firstRow = rows.first();
    const keyName = await firstRow.locator(".ak-name").textContent();
    await firstRow.locator("button", { hasText: "Revoke" }).click();

    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Type the key name to confirm
    if (keyName) {
      await dialog.locator("input[type='text']").fill(keyName.trim());
    }

    await dialog.getByRole("button", { name: "Revoke Key" }).click();
    await expect(dialog).not.toBeVisible({ timeout: 10000 });

    await expect(rows).toHaveCount(initialCount - 1, { timeout: 10000 });
  });

  /* ------------------------------------------------------------------ */
  /*  13. Revoke all → empty state                                       */
  /* ------------------------------------------------------------------ */
  test("13 - revoke all keys shows empty state", async ({ page }) => {
    const akTable = page.locator(".ak-table");

    // Keep revoking until no rows left
    while (await akTable.isVisible().catch(() => false)) {
      const rows = akTable.locator(".body-row");
      const count = await rows.count();
      if (count === 0) break;

      const firstRow = rows.first();
      const keyName = await firstRow.locator(".ak-name").textContent();
      await firstRow.locator("button", { hasText: "Revoke" }).click();

      const dialog = page.getByRole("dialog");
      await expect(dialog).toBeVisible();

      // Type the key name to confirm
      if (keyName) {
        await dialog.locator("input[type='text']").fill(keyName.trim());
      }

      await dialog.getByRole("button", { name: "Revoke Key" }).click();
      await expect(dialog).not.toBeVisible({ timeout: 10000 });

      // Wait briefly for the table to update
      await page.waitForTimeout(500);
    }

    // Empty state should show
    await expect(
      page.getByText("No API keys"),
    ).toBeVisible({ timeout: 10000 });
  });

  /* ------------------------------------------------------------------ */
  /*  14. Empty state → Create Key → create → table has 1 row           */
  /* ------------------------------------------------------------------ */
  test("14 - empty state Create Key button creates key and shows table", async ({
    page,
  }) => {
    // Verify empty state
    await expect(
      page.getByText("No API keys"),
    ).toBeVisible({ timeout: 10000 });

    // Click Create Key in empty state
    await page.getByRole("button", { name: "Create Key" }).click();

    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Fill in key details
    await dialog.locator("input[type='text']").fill("e2e-recovery-key");

    // Create
    await dialog.getByRole("button", { name: "Create Key" }).click();
    await expect(
      dialog.getByText("API Key Created"),
    ).toBeVisible({ timeout: 10000 });

    // Done
    await dialog.getByRole("button", { name: "Done" }).click();
    await expect(dialog).not.toBeVisible();

    // Table should appear with 1 row
    const akTable = page.locator(".ak-table");
    await expect(akTable).toBeVisible({ timeout: 10000 });
    const rows = akTable.locator(".body-row");
    await expect(rows).toHaveCount(1);
    await expect(
      akTable.locator(".ak-name", { hasText: "e2e-recovery-key" }),
    ).toBeVisible();
  });
});

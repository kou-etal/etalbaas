import { test, expect } from "@playwright/test";
import { createTestProject, deleteTestProject } from "./helpers/api";

/**
 * 10 - Project Secrets Tab
 *
 * Covers the Secrets tab on the project detail page (/projects/{id}).
 * The Secrets tab has real API integration (SecretService RPC) for CRUD.
 *
 * Page structure:
 * - Info banner at top (static, not dismissible)
 * - "Add Secret" button in header
 * - Table with columns: Name, Description, Value (masked), Last rotated, Created, Actions (Rotate / Delete)
 * - EmptyState when no secrets exist
 *
 * Add Secret dialog (Radix Dialog):
 * - Name input (auto-uppercased, A-Z0-9_ only)
 * - Value input (type=password)
 * - Description input (optional)
 * - Cancel / Add Secret buttons
 * - X close button (sr-only "Close")
 *
 * Rotate Secret dialog:
 * - Shows current secret name
 * - New value input (type=password)
 * - Cancel / Rotate Value buttons
 *
 * Delete Secret dialog (ConfirmDialog):
 * - Description mentioning secret name
 * - Cancel / Delete Secret buttons
 *
 * Setup: 1 project via API. Secrets created via UI in tests.
 */

test.describe.serial("Project Secrets Tab", () => {
  let project: { id: string; displayName: string };

  test.beforeAll(async () => {
    project = await createTestProject("secrets-tab");
  });

  test.afterAll(async () => {
    try {
      await deleteTestProject(project.id);
    } catch {}
  });

  test.beforeEach(async ({ page }) => {
    await page.goto(`/projects/${project.id}`);
    // Click the Secrets tab
    const secretsTab = page.locator('[role="tab"]').filter({ hasText: /^Secrets/ });
    await expect(secretsTab).toBeVisible({ timeout: 15000 });
    await secretsTab.click();
    await expect(secretsTab).toHaveAttribute("aria-selected", "true");
    // Wait for the secrets heading to be visible
    await expect(
      page.getByRole("heading", { name: "Secrets", exact: true }),
    ).toBeVisible({ timeout: 10000 });
  });

  /* ------------------------------------------------------------------ */
  /*  1. Info banner is visible                                          */
  /* ------------------------------------------------------------------ */
  test("1 - info banner is visible with encryption message", async ({
    page,
  }) => {
    // The info banner is a div.info-banner with Lock icon and encryption info
    const banner = page.locator(".info-banner");
    await expect(banner).toBeVisible();

    const bannerText = await banner.textContent();
    expect(bannerText).toContain("encrypted");
    expect(bannerText).toContain("Kubernetes");
  });

  /* ------------------------------------------------------------------ */
  /*  2. Add modal dismiss methods (Cancel / X / overlay)                */
  /* ------------------------------------------------------------------ */
  test("2 - add modal dismisses via Cancel, X, and overlay click", async ({
    page,
  }) => {
    const dialog = page.getByRole("dialog");

    // --- Cancel button ---
    await page.getByRole("button", { name: "Add Secret" }).first().click();
    await expect(dialog).toBeVisible();
    await dialog.getByRole("button", { name: "Cancel" }).click();
    await expect(dialog).not.toBeVisible();

    // --- X (close) button ---
    await page.getByRole("button", { name: "Add Secret" }).first().click();
    await expect(dialog).toBeVisible();
    const closeBtn = dialog.getByRole("button", { name: "Close" });
    await expect(closeBtn).toBeVisible();
    await closeBtn.click();
    await expect(dialog).not.toBeVisible();

    // --- Overlay click ---
    await page.getByRole("button", { name: "Add Secret" }).first().click();
    await expect(dialog).toBeVisible();
    // Radix Dialog overlay is a sibling of the content; click top-left of viewport
    await page.mouse.click(5, 5);
    await expect(dialog).not.toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  3. Add modal: show/hide toggle for value field                     */
  /* ------------------------------------------------------------------ */
  test("3 - add modal: value field is password type by default", async ({
    page,
  }) => {
    await page.getByRole("button", { name: "Add Secret" }).first().click();
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // The value input should have type=password
    const valueInput = dialog.locator("#secret-value");
    await expect(valueInput).toBeVisible();
    await expect(valueInput).toHaveAttribute("type", "password");

    // Type something and verify the value is captured
    await valueInput.fill("my-secret-value");
    await expect(valueInput).toHaveValue("my-secret-value");

    // Close
    await dialog.getByRole("button", { name: "Cancel" }).click();
  });

  /* ------------------------------------------------------------------ */
  /*  4. Create secret — name + value + description                      */
  /* ------------------------------------------------------------------ */
  test("4 - create secret with name, value, and description", async ({
    page,
  }) => {
    await page.getByRole("button", { name: "Add Secret" }).first().click();
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Fill in the form
    await dialog.locator("#secret-name").fill("MY_FIRST_SECRET");
    await dialog.locator("#secret-value").fill("super-secret-value-1");
    await dialog.locator("#secret-desc").fill("First test secret");

    // Verify name is uppercased (input enforces uppercase)
    await expect(dialog.locator("#secret-name")).toHaveValue("MY_FIRST_SECRET");

    // Submit
    await dialog.getByRole("button", { name: "Add Secret" }).click();

    // Dialog should close
    await expect(dialog).not.toBeVisible({ timeout: 10000 });

    // Table row should appear with the secret name
    const table = page.locator("table");
    await expect(table).toBeVisible({ timeout: 10000 });
    await expect(table.locator("code", { hasText: "MY_FIRST_SECRET" })).toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  5. Create 2nd secret — table has 2 rows                            */
  /* ------------------------------------------------------------------ */
  test("5 - create second secret — table has 2 rows", async ({ page }) => {
    await page.getByRole("button", { name: "Add Secret" }).first().click();
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    await dialog.locator("#secret-name").fill("MY_SECOND_SECRET");
    await dialog.locator("#secret-value").fill("super-secret-value-2");
    await dialog.locator("#secret-desc").fill("Second test secret");

    await dialog.getByRole("button", { name: "Add Secret" }).click();
    await expect(dialog).not.toBeVisible({ timeout: 10000 });

    // Table should now have 2 body rows
    const rows = page.locator("table tbody tr");
    await expect(rows).toHaveCount(2, { timeout: 10000 });
  });

  /* ------------------------------------------------------------------ */
  /*  6. Rotate secret — Rotate button → new value → rotated             */
  /* ------------------------------------------------------------------ */
  test("6 - rotate secret updates Last rotated", async ({ page }) => {
    // Wait for table
    const table = page.locator("table");
    await expect(table).toBeVisible({ timeout: 10000 });

    // Find the row with MY_FIRST_SECRET
    const targetRow = table.locator("tbody tr", {
      has: page.locator("code", { hasText: "MY_FIRST_SECRET" }),
    });
    await expect(targetRow).toBeVisible();

    // Get the initial "Last rotated" text
    const rotatedCell = targetRow.locator("td").nth(3);
    const initialText = await rotatedCell.textContent();
    expect(initialText).toContain("Never rotated");

    // Click the Rotate button (RotateCw icon button, first in the actions cell)
    const rotateBtn = targetRow.locator("button").first();
    await rotateBtn.click();

    // Rotate dialog should appear
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();
    await expect(dialog.locator("code", { hasText: "MY_FIRST_SECRET" })).toBeVisible();

    // Fill new value
    await dialog.locator("input[type='password']").fill("rotated-value-new");

    // Click Rotate Value
    await dialog.getByRole("button", { name: "Rotate Value" }).click();
    await expect(dialog).not.toBeVisible({ timeout: 10000 });

    // The "Last rotated" column should no longer say "Never rotated"
    await expect(rotatedCell).not.toHaveText("Never rotated", { timeout: 10000 });
  });

  /* ------------------------------------------------------------------ */
  /*  7. Rotate modal: Cancel → no change                                */
  /* ------------------------------------------------------------------ */
  test("7 - rotate modal: Cancel does not change anything", async ({
    page,
  }) => {
    const table = page.locator("table");
    await expect(table).toBeVisible({ timeout: 10000 });

    const targetRow = table.locator("tbody tr", {
      has: page.locator("code", { hasText: "MY_SECOND_SECRET" }),
    });
    await expect(targetRow).toBeVisible();

    // Get current "Last rotated" text
    const rotatedCell = targetRow.locator("td").nth(3);
    const beforeText = await rotatedCell.textContent();

    // Open rotate dialog
    const rotateBtn = targetRow.locator("button").first();
    await rotateBtn.click();

    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Fill a value but cancel
    await dialog.locator("input[type='password']").fill("should-not-apply");
    await dialog.getByRole("button", { name: "Cancel" }).click();
    await expect(dialog).not.toBeVisible();

    // "Last rotated" should be unchanged
    const afterText = await rotatedCell.textContent();
    expect(afterText).toBe(beforeText);
  });

  /* ------------------------------------------------------------------ */
  /*  8. Delete dialog: confirmation via ConfirmDialog                   */
  /* ------------------------------------------------------------------ */
  test("8 - delete dialog shows confirmation with secret name", async ({
    page,
  }) => {
    const table = page.locator("table");
    await expect(table).toBeVisible({ timeout: 10000 });

    const targetRow = table.locator("tbody tr", {
      has: page.locator("code", { hasText: "MY_FIRST_SECRET" }),
    });
    await expect(targetRow).toBeVisible();

    // Click the delete button (Trash2 icon, second button in actions cell)
    const deleteBtn = targetRow.locator("button").nth(1);
    await deleteBtn.click();

    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Should mention the secret name in the description
    const dialogText = await dialog.textContent();
    expect(dialogText).toContain("MY_FIRST_SECRET");

    // Cancel should dismiss
    await dialog.getByRole("button", { name: "Cancel" }).click();
    await expect(dialog).not.toBeVisible();

    // Row should still exist
    await expect(targetRow).toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  9. Delete secret — confirm → row disappears → 1 row left          */
  /* ------------------------------------------------------------------ */
  test("9 - delete secret removes row from table", async ({ page }) => {
    const table = page.locator("table");
    await expect(table).toBeVisible({ timeout: 10000 });

    // Verify we start with 2 rows
    const rows = page.locator("table tbody tr");
    await expect(rows).toHaveCount(2);

    const targetRow = table.locator("tbody tr", {
      has: page.locator("code", { hasText: "MY_FIRST_SECRET" }),
    });
    await expect(targetRow).toBeVisible();

    // Click delete button
    const deleteBtn = targetRow.locator("button").nth(1);
    await deleteBtn.click();

    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Type the secret name to confirm
    await dialog.locator("input[type='text']").fill("MY_FIRST_SECRET");

    // Click "Delete Secret" to confirm
    await dialog.getByRole("button", { name: "Delete Secret" }).click();
    await expect(dialog).not.toBeVisible({ timeout: 10000 });

    // Should now have 1 row
    await expect(rows).toHaveCount(1, { timeout: 10000 });

    // The deleted secret should not be visible
    await expect(
      table.locator("code", { hasText: "MY_FIRST_SECRET" }),
    ).not.toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  10. Delete 2nd secret → empty state                                */
  /* ------------------------------------------------------------------ */
  test("10 - delete last secret shows empty state", async ({ page }) => {
    const table = page.locator("table");
    await expect(table).toBeVisible({ timeout: 10000 });

    const targetRow = table.locator("tbody tr", {
      has: page.locator("code", { hasText: "MY_SECOND_SECRET" }),
    });
    await expect(targetRow).toBeVisible();

    // Delete
    const deleteBtn = targetRow.locator("button").nth(1);
    await deleteBtn.click();

    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Type the secret name to confirm
    await dialog.locator("input[type='text']").fill("MY_SECOND_SECRET");

    await dialog.getByRole("button", { name: "Delete Secret" }).click();
    await expect(dialog).not.toBeVisible({ timeout: 10000 });

    // Table should no longer be visible
    await expect(table).not.toBeVisible({ timeout: 10000 });

    // Empty state should appear
    await expect(
      page.getByText("No secrets configured"),
    ).toBeVisible({ timeout: 10000 });
  });

  /* ------------------------------------------------------------------ */
  /*  11. Empty state → Add Secret → create → table has 1 row           */
  /* ------------------------------------------------------------------ */
  test("11 - empty state Add Secret button creates secret and shows table", async ({
    page,
  }) => {
    // Verify empty state is shown
    await expect(
      page.getByText("No secrets configured"),
    ).toBeVisible({ timeout: 10000 });

    // Click the Add Secret button (use first() since both header and empty state have this button)
    await page.getByRole("button", { name: "Add Secret" }).first().click();

    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    await dialog.locator("#secret-name").fill("RECOVERY_SECRET");
    await dialog.locator("#secret-value").fill("recovery-value");
    await dialog.locator("#secret-desc").fill("Created from empty state");

    await dialog.getByRole("button", { name: "Add Secret" }).click();
    await expect(dialog).not.toBeVisible({ timeout: 10000 });

    // Table should now be visible with 1 row
    const table = page.locator("table");
    await expect(table).toBeVisible({ timeout: 10000 });
    const rows = page.locator("table tbody tr");
    await expect(rows).toHaveCount(1);
    await expect(
      table.locator("code", { hasText: "RECOVERY_SECRET" }),
    ).toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  12. Name column header is "Name"                                   */
  /* ------------------------------------------------------------------ */
  test("12 - table has correct column headers", async ({ page }) => {
    const table = page.locator("table");
    await expect(table).toBeVisible({ timeout: 10000 });

    const headers = table.locator("thead th");
    // Expected columns: Name, Description, Value, Last rotated, Created, (actions)
    const headerTexts = await headers.allTextContents();
    expect(headerTexts).toContain("Name");
    expect(headerTexts).toContain("Description");
    expect(headerTexts).toContain("Last rotated");
    expect(headerTexts).toContain("Created");
  });

  /* ------------------------------------------------------------------ */
  /*  13. Secret value is always masked in table                         */
  /* ------------------------------------------------------------------ */
  test("13 - secret value column shows masked dots", async ({ page }) => {
    const table = page.locator("table");
    await expect(table).toBeVisible({ timeout: 10000 });

    // The Value column (3rd column) should show masked dots
    const valueCell = table.locator("tbody tr").first().locator("td").nth(2);
    const valueText = await valueCell.textContent();
    expect(valueText).toContain("••••");
  });

  /* ------------------------------------------------------------------ */
  /*  14. Name input enforces uppercase and valid characters              */
  /* ------------------------------------------------------------------ */
  test("14 - name input enforces uppercase and strips invalid characters", async ({
    page,
  }) => {
    await page.getByRole("button", { name: "Add Secret" }).first().click();
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    const nameInput = dialog.locator("#secret-name");

    // Type lowercase — should be converted to uppercase
    await nameInput.fill("my_key_name");
    await expect(nameInput).toHaveValue("MY_KEY_NAME");

    // Type with invalid characters — should strip them
    await nameInput.fill("");
    await nameInput.pressSequentially("test-key.name!@#");
    const val = await nameInput.inputValue();
    // Only uppercase letters, digits, underscores allowed
    expect(val).toMatch(/^[A-Z0-9_]*$/);

    await dialog.getByRole("button", { name: "Cancel" }).click();
  });

  /* ------------------------------------------------------------------ */
  /*  15. Add Secret button is disabled when name or value is empty      */
  /* ------------------------------------------------------------------ */
  test("15 - Add Secret button is disabled when name or value is empty", async ({
    page,
  }) => {
    await page.getByRole("button", { name: "Add Secret" }).first().click();
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    const submitBtn = dialog.getByRole("button", { name: "Add Secret" });

    // Initially disabled (both fields empty)
    await expect(submitBtn).toBeDisabled();

    // Fill only name — still disabled
    await dialog.locator("#secret-name").fill("SOME_KEY");
    await expect(submitBtn).toBeDisabled();

    // Fill value — now enabled
    await dialog.locator("#secret-value").fill("some-value");
    await expect(submitBtn).toBeEnabled();

    // Clear name — disabled again
    await dialog.locator("#secret-name").fill("");
    await expect(submitBtn).toBeDisabled();

    await dialog.getByRole("button", { name: "Cancel" }).click();
  });
});

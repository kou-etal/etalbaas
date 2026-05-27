import { test, expect } from "@playwright/test";
import {
  createTestProject,
  deleteTestProject,
  createTestFunction,
  deleteTestFunction,
} from "./helpers/api";

/**
 * 14 - Function Delete Flow
 *
 * Covers the delete confirmation modal on the function detail page
 * (/projects/{id}/functions/{fnId}).
 *
 * Delete flow: Click Delete button (topbar or Settings danger zone) -> modal opens:
 * - Modal title: "Delete function"
 * - Warning message with function name in bold/mono
 * - Input: "Type {fn.name} to confirm"
 * - Delete Function button (disabled until name matches exactly)
 * - Cancel button
 *
 * Setup: 1 project + 2 functions via API.
 *   - Function A: survives all tests (used for post-delete verification)
 *   - Function B: deleted in test 7
 */

test.describe.serial("Function Delete Flow", () => {
  let project: { id: string; displayName: string };
  let fnA: { id: string; name: string };
  let fnB: { id: string; name: string };

  test.beforeAll(async () => {
    project = await createTestProject("fn-del");
    fnA = await createTestFunction(project.id, "survivor");
    fnB = await createTestFunction(project.id, "to-delete");
  });

  test.afterAll(async () => {
    // Clean up function A (function B is deleted in test 7)
    try {
      await deleteTestFunction(project.id, fnA.id);
    } catch {}
    try {
      await deleteTestFunction(project.id, fnB.id);
    } catch {}
    try {
      await deleteTestProject(project.id);
    } catch {}
  });

  /* Tests 1-7 navigate to function B detail page */
  test.beforeEach(async ({ page }, testInfo) => {
    // Test 8 navigates to function A instead
    if (testInfo.title.startsWith("8")) return;

    await page.goto(`/projects/${project.id}/functions/${fnB.id}`);
    // Wait for function detail to load (function name visible in header)
    await expect(page.locator(".fn-header h1")).toBeVisible({ timeout: 15000 });
  });

  /* ------------------------------------------------------------------ */
  /*  1. Function name in modal prompt matches correctly                 */
  /* ------------------------------------------------------------------ */
  test("1 - function name in modal prompt matches correctly", async ({
    page,
  }) => {
    // Open the delete modal via the topbar Delete button
    await page.locator("button.btn-danger-ghost", { hasText: "Delete" }).click();

    // Wait for the modal scrim to be visible
    const scrim = page.locator(".modal-scrim.open");
    await expect(scrim).toBeVisible();

    // Modal title should be "Delete function"
    await expect(scrim.locator("h3.err")).toHaveText("Delete function");

    // Warning message should contain the function name in bold/mono
    const warning = scrim.locator(".modal-body .modal-err p");
    await expect(warning).toBeVisible();
    const strongText = scrim.locator(".modal-body .modal-err strong.mono");
    await expect(strongText).toHaveText(fnB.name);

    // Label should instruct to type the function name
    const label = scrim.locator('label[for="delete-fn-input"]');
    await expect(label).toContainText(fnB.name);

    // Close
    await scrim.locator("button.btn-ghost", { hasText: "Cancel" }).click();
    await expect(scrim).not.toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  2. Delete disabled when input empty                                */
  /* ------------------------------------------------------------------ */
  test("2 - delete button disabled when input is empty", async ({ page }) => {
    await page.locator("button.btn-danger-ghost", { hasText: "Delete" }).click();

    const scrim = page.locator(".modal-scrim.open");
    await expect(scrim).toBeVisible();

    const deleteBtn = scrim.locator("button.btn-danger", {
      hasText: "Delete Function",
    });
    const input = scrim.locator("#delete-fn-input");

    // Input should be empty by default
    await expect(input).toHaveValue("");

    // Delete Function button should be disabled
    await expect(deleteBtn).toBeDisabled();

    // Close
    await scrim.locator("button.btn-ghost", { hasText: "Cancel" }).click();
  });

  /* ------------------------------------------------------------------ */
  /*  3. Delete disabled when partial name entered                       */
  /* ------------------------------------------------------------------ */
  test("3 - delete button disabled when partial name entered", async ({
    page,
  }) => {
    await page.locator("button.btn-danger-ghost", { hasText: "Delete" }).click();

    const scrim = page.locator(".modal-scrim.open");
    await expect(scrim).toBeVisible();

    const deleteBtn = scrim.locator("button.btn-danger", {
      hasText: "Delete Function",
    });
    const input = scrim.locator("#delete-fn-input");

    // Type a partial name (first half of the function name)
    const partialName = fnB.name.slice(0, Math.floor(fnB.name.length / 2));
    await input.fill(partialName);

    // Delete button should still be disabled
    await expect(deleteBtn).toBeDisabled();

    // Close
    await scrim.locator("button.btn-ghost", { hasText: "Cancel" }).click();
  });

  /* ------------------------------------------------------------------ */
  /*  4. Delete enabled when full name matches                           */
  /* ------------------------------------------------------------------ */
  test("4 - delete button enabled when full name matches", async ({
    page,
  }) => {
    await page.locator("button.btn-danger-ghost", { hasText: "Delete" }).click();

    const scrim = page.locator(".modal-scrim.open");
    await expect(scrim).toBeVisible();

    const deleteBtn = scrim.locator("button.btn-danger", {
      hasText: "Delete Function",
    });
    const input = scrim.locator("#delete-fn-input");

    // Type the exact function name
    await input.fill(fnB.name);

    // Delete button should now be enabled
    await expect(deleteBtn).toBeEnabled();

    // Close without deleting
    await scrim.locator("button.btn-ghost", { hasText: "Cancel" }).click();
  });

  /* ------------------------------------------------------------------ */
  /*  5. Cancel -> function still in list                                */
  /* ------------------------------------------------------------------ */
  test("5 - cancel closes modal and function remains in list", async ({
    page,
  }) => {
    await page.locator("button.btn-danger-ghost", { hasText: "Delete" }).click();

    const scrim = page.locator(".modal-scrim.open");
    await expect(scrim).toBeVisible();

    // Type the full name to enable the button
    await scrim.locator("#delete-fn-input").fill(fnB.name);
    await expect(
      scrim.locator("button.btn-danger", { hasText: "Delete Function" }),
    ).toBeEnabled();

    // Click Cancel
    await scrim.locator("button.btn-ghost", { hasText: "Cancel" }).click();
    await expect(scrim).not.toBeVisible();

    // Navigate to the project page to verify function is still listed
    await page.goto(`/projects/${project.id}`);
    const fnTab = page.getByRole("tab", { name: "Functions" });
    await fnTab.click();
    await expect(fnTab).toHaveAttribute("aria-selected", "true");
    await expect(
      page.locator(".fn-table .body-row", { hasText: fnB.name }),
    ).toBeVisible({ timeout: 15000 });
  });

  /* ------------------------------------------------------------------ */
  /*  6. Scrim click -> function still in list                           */
  /* ------------------------------------------------------------------ */
  test("6 - scrim click closes modal and function remains in list", async ({
    page,
  }) => {
    await page.locator("button.btn-danger-ghost", { hasText: "Delete" }).click();

    const scrim = page.locator(".modal-scrim.open");
    await expect(scrim).toBeVisible();

    // Click the scrim (outside the modal) to close
    const scrimBox = await scrim.boundingBox();
    expect(scrimBox).toBeTruthy();
    // Click top-left corner which is outside the centered .modal
    await page.mouse.click(scrimBox!.x + 5, scrimBox!.y + 5);
    await expect(scrim).not.toBeVisible();

    // Navigate to project page to verify function still exists
    await page.goto(`/projects/${project.id}`);
    const fnTab = page.getByRole("tab", { name: "Functions" });
    await fnTab.click();
    await expect(fnTab).toHaveAttribute("aria-selected", "true");
    await expect(
      page.locator(".fn-table .body-row", { hasText: fnB.name }),
    ).toBeVisible({ timeout: 15000 });
  });

  /* ------------------------------------------------------------------ */
  /*  7. Delete execution -> redirect -> function removed from list      */
  /* ------------------------------------------------------------------ */
  test("7 - delete execution redirects and removes function from list", async ({
    page,
  }) => {
    await page.locator("button.btn-danger-ghost", { hasText: "Delete" }).click();

    const scrim = page.locator(".modal-scrim.open");
    await expect(scrim).toBeVisible();

    // Type the exact function name
    await scrim.locator("#delete-fn-input").fill(fnB.name);

    // Click the Delete Function button
    const deleteBtn = scrim.locator("button.btn-danger", {
      hasText: "Delete Function",
    });
    await expect(deleteBtn).toBeEnabled();
    await deleteBtn.click();

    // Should redirect to /projects/{id}
    await expect(page).toHaveURL(
      new RegExp(`/projects/${project.id}$`),
      { timeout: 15000 },
    );

    // Click the Functions tab
    const fnTab = page.getByRole("tab", { name: "Functions" });
    await fnTab.click();
    await expect(fnTab).toHaveAttribute("aria-selected", "true");

    // Wait for the function list to load
    await page.waitForTimeout(2000);

    // Function B should NOT be in the list
    await expect(
      page.locator(".fn-table .body-row", { hasText: fnB.name }),
    ).not.toBeVisible({ timeout: 10000 });
  });

  /* ------------------------------------------------------------------ */
  /*  8. After delete: remaining function still accessible               */
  /* ------------------------------------------------------------------ */
  test("8 - remaining function is still accessible after delete", async ({
    page,
  }) => {
    // Navigate to function A detail page
    await page.goto(`/projects/${project.id}/functions/${fnA.id}`);

    // Wait for function detail to load
    await expect(page.locator(".fn-header h1")).toBeVisible({ timeout: 15000 });

    // Verify function A name is displayed
    await expect(page.locator(".fn-header h1")).toHaveText(fnA.name);

    // Verify the page has tabs (overview, triggers, invocations, etc.)
    const tablist = page.locator('.fd-tabs[role="tablist"]');
    await expect(tablist).toBeVisible();
    await expect(
      tablist.locator('button[role="tab"]', { hasText: "Overview" }),
    ).toBeVisible();
    await expect(
      tablist.locator('button[role="tab"]', { hasText: "Settings" }),
    ).toBeVisible();

    // Verify the kind badge and status badge are present
    await expect(page.locator(".fn-header .kind-badge")).toBeVisible();
    await expect(page.locator(".fn-header .status-badge")).toBeVisible();

    // Verify the Invoke and Delete buttons are present in the topbar
    await expect(
      page.locator("button.btn-primary", { hasText: "Invoke" }),
    ).toBeVisible();
    await expect(
      page.locator("button.btn-danger-ghost", { hasText: "Delete" }),
    ).toBeVisible();
  });
});

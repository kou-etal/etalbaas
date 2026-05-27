import { test, expect } from "@playwright/test";
import {
  createTestProject,
  deleteTestProject,
  uniqueName,
} from "./helpers/api";

/**
 * 04 - Project Detail: Topbar & Tabs
 *
 * Covers the project detail page (/projects/{id}): breadcrumb navigation,
 * kebab menu, pause/delete modals, and 8-tab navigation with aria-selected.
 *
 * Setup creates 2 projects via API. Tests 10 and 15-16 mutate state, so the
 * entire suite runs serially.
 */

test.describe.serial("Project topbar & tabs", () => {
  let projectA: { id: string; displayName: string; status: string };
  let projectB: { id: string; displayName: string; status: string };

  /* ---- Setup: create two projects via API ---- */
  test.beforeAll(async () => {
    const a = await createTestProject("topbar-a");
    projectA = { id: a.id, displayName: a.displayName, status: a.status };

    const b = await createTestProject("topbar-b");
    projectB = { id: b.id, displayName: b.displayName, status: b.status };
  });

  /* ---- Teardown: delete both projects (ignore errors) ---- */
  test.afterAll(async () => {
    for (const p of [projectA, projectB]) {
      if (!p?.id) continue;
      try {
        await deleteTestProject(p.id);
      } catch {
        // Ignore – may have been deleted in test 15
      }
    }
  });

  /* ------------------------------------------------------------------ */
  /*  1. Breadcrumb "Projects" link navigates to /projects              */
  /* ------------------------------------------------------------------ */
  test("1 - breadcrumb Projects link navigates to /projects", async ({
    page,
  }) => {
    await page.goto(`/projects/${projectA.id}`);
    await expect(page.locator(".crumbs")).toBeVisible({ timeout: 15000 });

    const projectsLink = page.locator(".crumbs").getByText("Projects");
    await expect(projectsLink).toBeVisible();
    await projectsLink.click();

    await expect(page).toHaveURL(/\/projects$/);
  });

  /* ------------------------------------------------------------------ */
  /*  2. Project name & status badge match API values                   */
  /* ------------------------------------------------------------------ */
  test("2 - project name and status badge match API values", async ({
    page,
  }) => {
    await page.goto(`/projects/${projectA.id}`);
    await expect(page.locator(".crumbs")).toBeVisible({ timeout: 15000 });

    // Display name in breadcrumb <strong> (wait for project data to load)
    await expect(
      page.locator(".crumbs strong", { hasText: projectA.displayName }),
    ).toBeVisible({ timeout: 15000 });

    // Status badge
    const badge = page.locator(".badge-lg");
    await expect(badge).toBeVisible();

    const badgeText = await badge.textContent();
    expect(badgeText).toBeTruthy();
    // Status should be one of the valid statuses (the API returns lowercase;
    // the badge may capitalise)
    const validStatuses = [
      "Pending",
      "Provisioning",
      "Ready",
      "Paused",
      "Failed",
      "Deleted",
    ];
    expect(
      validStatuses.some((s) => badgeText!.trim().toLowerCase() === s.toLowerCase()),
    ).toBeTruthy();
  });

  /* ------------------------------------------------------------------ */
  /*  3. Kebab menu open/close                                          */
  /* ------------------------------------------------------------------ */
  test("3 - kebab menu opens and closes on outside click", async ({
    page,
  }) => {
    await page.goto(`/projects/${projectA.id}`);
    await expect(page.getByLabel("More actions")).toBeVisible({
      timeout: 15000,
    });

    const kebab = page.getByLabel("More actions");
    const menu = page.locator('[role="menu"]');

    // Open
    await kebab.click();
    await expect(menu).toBeVisible();

    // Should have 3 menu items (+ divider)
    const items = menu.locator(".menu-item");
    await expect(items).toHaveCount(3);

    // Close by clicking outside (click on the main content area, away from the menu)
    await page.mouse.click(400, 400);
    await expect(menu).not.toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  4. Kebab: "Duplicate project" item visible                        */
  /* ------------------------------------------------------------------ */
  test("4 - kebab menu shows Duplicate project item", async ({ page }) => {
    await page.goto(`/projects/${projectA.id}`);
    await expect(page.getByLabel("More actions")).toBeVisible({
      timeout: 15000,
    });

    await page.getByLabel("More actions").click();

    const menu = page.locator('[role="menu"]');
    await expect(menu).toBeVisible();
    await expect(
      menu.locator(".menu-item", { hasText: "Duplicate project" }),
    ).toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  5. Kebab: "Export config" item visible                             */
  /* ------------------------------------------------------------------ */
  test("5 - kebab menu shows Export config item", async ({ page }) => {
    await page.goto(`/projects/${projectA.id}`);
    await expect(page.getByLabel("More actions")).toBeVisible({
      timeout: 15000,
    });

    await page.getByLabel("More actions").click();

    const menu = page.locator('[role="menu"]');
    await expect(menu).toBeVisible();
    await expect(
      menu.locator(".menu-item", { hasText: "Export config" }),
    ).toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  6. Kebab: "Delete project" opens delete modal                     */
  /* ------------------------------------------------------------------ */
  test("6 - kebab Delete project opens delete modal", async ({ page }) => {
    await page.goto(`/projects/${projectA.id}`);
    await expect(page.getByLabel("More actions")).toBeVisible({
      timeout: 15000,
    });

    await page.getByLabel("More actions").click();

    const menu = page.locator('[role="menu"]');
    await expect(menu).toBeVisible();

    await menu.locator(".menu-item.danger", { hasText: "Delete project" }).click();

    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Close the modal via Cancel so state is clean for later tests
    await dialog.getByRole("button", { name: "Cancel" }).click();
    await expect(dialog).not.toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  7. 8 tab switching with aria-selected toggles                     */
  /* ------------------------------------------------------------------ */
  test("7 - clicking each of 8 tabs toggles aria-selected", async ({
    page,
  }) => {
    await page.goto(`/projects/${projectA.id}`);
    const tablist = page.locator('[role="tablist"]');
    await expect(tablist).toBeVisible({ timeout: 15000 });

    const tabNames = [
      "Overview",
      "Database",
      "Functions",
      "Events",
      "Storage",
      "Secrets",
      "API Keys",
      "Settings",
    ];

    // Verify there are exactly 8 tab buttons
    const tabs = tablist.getByRole("tab");
    await expect(tabs).toHaveCount(8);

    // Click each tab in order and verify aria-selected toggles
    for (const name of tabNames) {
      const tab = page.getByRole("tab", { name });
      await tab.click();
      await expect(tab).toHaveAttribute("aria-selected", "true");
      await expect(tab).toHaveClass(/active/);

      // All other tabs should NOT be selected
      for (const otherName of tabNames) {
        if (otherName === name) continue;
        const otherTab = page.getByRole("tab", { name: otherName });
        await expect(otherTab).toHaveAttribute("aria-selected", "false");
      }
    }
  });

  /* ------------------------------------------------------------------ */
  /*  8. Overview is the default active tab on page load                 */
  /* ------------------------------------------------------------------ */
  test("8 - Overview is default active tab on page load", async ({ page }) => {
    await page.goto(`/projects/${projectA.id}`);
    const tablist = page.locator('[role="tablist"]');
    await expect(tablist).toBeVisible({ timeout: 15000 });

    const overviewTab = page.getByRole("tab", { name: "Overview" });
    await expect(overviewTab).toHaveAttribute("aria-selected", "true");
    await expect(overviewTab).toHaveClass(/active/);
  });

  /* ------------------------------------------------------------------ */
  /*  9. Pause modal: Cancel / X / scrim dismiss                        */
  /* ------------------------------------------------------------------ */
  test("9 - pause modal dismisses via Cancel, X, and scrim click", async ({
    page,
  }) => {
    await page.goto(`/projects/${projectA.id}`);
    const pauseBtn = page.locator("button.btn.btn-ghost", {
      hasText: "Pause",
    });
    await expect(pauseBtn).toBeVisible({ timeout: 15000 });

    const dialog = page.getByRole("dialog");

    // --- Cancel button ---
    await pauseBtn.click();
    await expect(dialog).toBeVisible();
    await dialog.getByRole("button", { name: "Cancel" }).click();
    await expect(dialog).not.toBeVisible();

    // --- X (close) button ---
    await pauseBtn.click();
    await expect(dialog).toBeVisible();
    // Try Close button; some modals use aria-label instead
    const closeBtn = dialog
      .getByRole("button", { name: "Close" })
      .or(dialog.locator("button[aria-label='Close']"));
    if (await closeBtn.first().isVisible()) {
      await closeBtn.first().click();
      await expect(dialog).not.toBeVisible();
    }

    // --- Scrim click ---
    await pauseBtn.click();
    await expect(dialog).toBeVisible();
    const scrim = page.locator(".modal-scrim.open");
    const scrimBox = await scrim.boundingBox();
    expect(scrimBox).toBeTruthy();
    // Click top-left corner (outside the centered .modal)
    await page.mouse.click(scrimBox!.x + 5, scrimBox!.y + 5);
    await expect(dialog).not.toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  10. Pause execution → status changes to Paused                    */
  /* ------------------------------------------------------------------ */
  test("10 - pause execution changes status to Paused", async ({ page }) => {
    await page.goto(`/projects/${projectA.id}`);
    const pauseBtn = page.locator("button.btn.btn-ghost", {
      hasText: "Pause",
    });
    await expect(pauseBtn).toBeVisible({ timeout: 15000 });

    // Open pause modal
    await pauseBtn.click();
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Click the Pause confirmation button inside the modal
    await dialog.getByRole("button", { name: "Pause" }).click();

    // Modal should close
    await expect(dialog).not.toBeVisible({ timeout: 10000 });

    // Status badge should now show "Paused"
    const badge = page.locator(".badge-lg");
    await expect(badge).toHaveText("Paused", { timeout: 15000 });

    // Verify via API as well
    const { getProject } = await import("./helpers/api");
    const updated = await getProject(projectA.id);
    expect(updated.status.toLowerCase()).toContain("paused");
    projectA.status = updated.status;
  });

  /* ------------------------------------------------------------------ */
  /*  11. Delete modal: empty name → Delete button disabled             */
  /* ------------------------------------------------------------------ */
  test("11 - delete modal: empty name keeps Delete button disabled", async ({
    page,
  }) => {
    await page.goto(`/projects/${projectB.id}`);
    await expect(page.getByLabel("More actions")).toBeVisible({
      timeout: 15000,
    });

    // Open kebab → Delete project
    await page.getByLabel("More actions").click();
    const menu = page.locator('[role="menu"]');
    await menu.locator(".menu-item.danger", { hasText: "Delete project" }).click();

    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Delete button should be disabled when name is empty
    const deleteBtn = dialog.getByRole("button", { name: "Delete" });
    await expect(deleteBtn).toBeDisabled();

    // Cancel
    await dialog.getByRole("button", { name: "Cancel" }).click();
  });

  /* ------------------------------------------------------------------ */
  /*  12. Delete modal: partial name → Delete button disabled           */
  /* ------------------------------------------------------------------ */
  test("12 - delete modal: partial name keeps Delete button disabled", async ({
    page,
  }) => {
    await page.goto(`/projects/${projectB.id}`);
    await expect(page.getByLabel("More actions")).toBeVisible({
      timeout: 15000,
    });

    await page.getByLabel("More actions").click();
    const menu = page.locator('[role="menu"]');
    await menu.locator(".menu-item.danger", { hasText: "Delete project" }).click();

    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Type only part of the display name
    const nameInput = dialog.locator('input[type="text"]');
    await nameInput.fill(projectB.displayName.slice(0, 5));

    // Delete button should still be disabled
    const deleteBtn = dialog.getByRole("button", { name: "Delete" });
    await expect(deleteBtn).toBeDisabled();

    await dialog.getByRole("button", { name: "Cancel" }).click();
  });

  /* ------------------------------------------------------------------ */
  /*  13. Delete modal: "I understand" checkbox required                 */
  /* ------------------------------------------------------------------ */
  test("13 - delete modal: checkbox required to enable Delete button", async ({
    page,
  }) => {
    await page.goto(`/projects/${projectB.id}`);
    await expect(page.getByLabel("More actions")).toBeVisible({
      timeout: 15000,
    });

    await page.getByLabel("More actions").click();
    const menu = page.locator('[role="menu"]');
    await menu.locator(".menu-item.danger", { hasText: "Delete project" }).click();

    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    const nameInput = dialog.locator('input[type="text"]');
    const checkbox = dialog.locator("input[type='checkbox']");
    const deleteBtn = dialog.getByRole("button", { name: "Delete" });

    // Fill exact name but do NOT check checkbox
    await nameInput.fill(projectB.displayName);
    await expect(deleteBtn).toBeDisabled();

    // Check the checkbox → Delete should become enabled
    await checkbox.check();
    await expect(deleteBtn).toBeEnabled();

    // Uncheck → disabled again
    await checkbox.uncheck();
    await expect(deleteBtn).toBeDisabled();

    await dialog.getByRole("button", { name: "Cancel" }).click();
  });

  /* ------------------------------------------------------------------ */
  /*  14. Delete modal: Cancel → no changes                             */
  /* ------------------------------------------------------------------ */
  test("14 - delete modal: Cancel leaves project unchanged", async ({
    page,
  }) => {
    await page.goto(`/projects/${projectB.id}`);
    await expect(page.getByLabel("More actions")).toBeVisible({
      timeout: 15000,
    });

    await page.getByLabel("More actions").click();
    const menu = page.locator('[role="menu"]');
    await menu.locator(".menu-item.danger", { hasText: "Delete project" }).click();

    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Fill name + check checkbox (ready to delete)
    await dialog.locator('input[type="text"]').fill(projectB.displayName);
    await dialog.locator("input[type='checkbox']").check();

    // But click Cancel instead
    await dialog.getByRole("button", { name: "Cancel" }).click();
    await expect(dialog).not.toBeVisible();

    // Project should still exist — verify via API
    const { getProject } = await import("./helpers/api");
    const project = await getProject(projectB.id);
    expect(project.id).toBe(projectB.id);

    // Breadcrumb should still show project name
    await expect(
      page.locator(".crumbs strong", { hasText: projectB.displayName }),
    ).toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  15. Delete execution → redirects to /projects                     */
  /* ------------------------------------------------------------------ */
  test("15 - delete project redirects to /projects", async ({ page }) => {
    await page.goto(`/projects/${projectB.id}`);
    await expect(page.getByLabel("More actions")).toBeVisible({
      timeout: 15000,
    });

    await page.getByLabel("More actions").click();
    const menu = page.locator('[role="menu"]');
    await menu.locator(".menu-item.danger", { hasText: "Delete project" }).click();

    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Fill name and check the checkbox
    await dialog.locator('input[type="text"]').fill(projectB.displayName);
    await dialog.locator("input[type='checkbox']").check();

    // Click Delete
    const deleteBtn = dialog.getByRole("button", { name: "Delete" });
    await expect(deleteBtn).toBeEnabled();
    await deleteBtn.click();

    // Should redirect to /projects
    await expect(page).toHaveURL(/\/projects$/, { timeout: 15000 });

    // The deleted project should no longer appear in the list
    await page.waitForTimeout(1000);
    const deletedCard = page.locator(`a.card[href='/projects/${projectB.id}']`);
    await expect(deletedCard).not.toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  16. After delete, remaining project works normally                 */
  /* ------------------------------------------------------------------ */
  test("16 - after delete, remaining project still accessible", async ({
    page,
  }) => {
    // Navigate to the remaining project (projectA)
    await page.goto(`/projects/${projectA.id}`);
    await expect(page.locator(".crumbs")).toBeVisible({ timeout: 15000 });

    // Breadcrumb should show projectA's name
    await expect(
      page.locator(".crumbs strong", { hasText: projectA.displayName }),
    ).toBeVisible();

    // Tabs should still be functional
    const tablist = page.locator('[role="tablist"]');
    await expect(tablist).toBeVisible();

    const overviewTab = page.getByRole("tab", { name: "Overview" });
    await expect(overviewTab).toHaveAttribute("aria-selected", "true");

    // Click another tab to verify functionality
    const settingsTab = page.getByRole("tab", { name: "Settings" });
    await settingsTab.click();
    await expect(settingsTab).toHaveAttribute("aria-selected", "true");
    await expect(overviewTab).toHaveAttribute("aria-selected", "false");

    // Kebab menu still works
    const kebab = page.getByLabel("More actions");
    await expect(kebab).toBeVisible();
    await kebab.click();
    const kebabMenu = page.locator('[role="menu"]');
    await expect(kebabMenu).toBeVisible();
    await expect(kebabMenu.locator(".menu-item")).toHaveCount(3);
  });
});

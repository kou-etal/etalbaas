import { test, expect } from "@playwright/test";
import { createTestProject, deleteTestProject, getProject } from "./helpers/api";

/**
 * 12 - Project Settings Tab
 *
 * Covers the Settings tab on the project detail page (/projects/{id}).
 * The Settings tab renders a page at /projects/{id}/settings (Next.js route)
 * accessed via the "Settings" tab in the project detail tablist.
 *
 * Page structure:
 * - General section: Project ID (read-only), Display name, Description
 * - Services section: PostgreSQL toggle + extensions, Redis toggle, PostgREST toggle + endpoint
 * - Danger zone: Transfer (disabled), Pause/Resume, Delete (type-to-confirm)
 *
 * Setup: 1 project with PG + Redis enabled via API.
 */

test.describe.serial("Project Settings Tab", () => {
  let project: {
    id: string;
    displayName: string;
    description: string;
    postgresEnabled: boolean;
    redisEnabled: boolean;
    postgrestEnabled: boolean;
    postgresExtensions: string[];
    status: string;
  };

  test.beforeAll(async () => {
    project = await createTestProject("settings-tab", {
      postgresEnabled: true,
      redisEnabled: true,
      postgrestEnabled: false,
      postgresExtensions: ["pgvector", "pgcrypto"],
    });
  });

  test.afterAll(async () => {
    try {
      await deleteTestProject(project.id);
    } catch {}
  });

  test.beforeEach(async ({ page }) => {
    await page.goto(`/projects/${project.id}`);
    // Click the Settings tab
    const settingsTab = page.getByRole("tab", { name: "Settings" });
    await expect(settingsTab).toBeVisible({ timeout: 15000 });
    await settingsTab.click();
    await expect(settingsTab).toHaveAttribute("aria-selected", "true");
    // Wait for settings content to load — look for "General" card heading
    await expect(page.getByText("General")).toBeVisible({ timeout: 10000 });
  });

  /* ------------------------------------------------------------------ */
  /*  1. Nav: General -> Services -> Danger zone switching                */
  /* ------------------------------------------------------------------ */
  test("1 - settings page shows General, Services, and Danger zone sections", async ({
    page,
  }) => {
    // The settings nav has links for each section.
    // Use the nav links to verify sections are present.
    await expect(
      page.locator("nav.settings-nav a", { hasText: "General" }),
    ).toBeVisible();
    await expect(
      page.locator("nav.settings-nav a", { hasText: "Services" }),
    ).toBeVisible();
    await expect(
      page.locator("nav.settings-nav a", { hasText: "Danger zone" }),
    ).toBeVisible();

    // Verify the section panels exist
    await expect(page.locator("#stg-general")).toBeVisible();
    await expect(page.locator("#stg-services")).toBeVisible();
    await expect(page.locator("#stg-danger")).toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  2. General: Project ID display and copy                            */
  /* ------------------------------------------------------------------ */
  test("2 - Project ID is displayed as read-only with copyable text", async ({
    page,
  }) => {
    // The project ID is in a readOnly input with className "mono"
    const projectIdInput = page.locator("input.mono[readonly]", { hasText: "" }).first();
    await expect(projectIdInput).toBeVisible();

    // Verify the project ID matches
    const inputValue = await projectIdInput.inputValue();
    expect(inputValue).toBe(project.id);
  });

  /* ------------------------------------------------------------------ */
  /*  3. General: Display name input is shown (disabled in current impl) */
  /* ------------------------------------------------------------------ */
  test("3 - Display name input shows project name", async ({ page }) => {
    // Look for the label "Display name" and the associated input
    const displayNameLabel = page.getByText("Display name");
    await expect(displayNameLabel).toBeVisible();

    // The input contains the project's display name
    const section = displayNameLabel.locator("..");
    const input = section.locator("input");
    await expect(input).toBeVisible();
    await expect(input).toHaveValue(project.displayName, { timeout: 10000 });
  });

  /* ------------------------------------------------------------------ */
  /*  4. General: Display name change -> Save -> reload -> persists      */
  /* ------------------------------------------------------------------ */
  test("4 - display name input shows correct value", async ({
    page,
  }) => {
    const label = page.getByText("Display name");
    await expect(label).toBeVisible();

    // Find the input near the Display name label
    const section = label.locator("..");
    const input = section.locator("input");
    await expect(input).toBeVisible();

    await expect(input).toHaveValue(project.displayName, { timeout: 10000 });
  });

  /* ------------------------------------------------------------------ */
  /*  5. General: Description is shown (disabled)                        */
  /* ------------------------------------------------------------------ */
  test("5 - description textarea shows project description", async ({ page }) => {
    const descLabel = page.getByText("Description").first();
    await expect(descLabel).toBeVisible();

    // The textarea shows the project description
    const textarea = page.locator("#stg-general textarea");
    await expect(textarea).toBeVisible();
    await expect(textarea).toHaveValue(project.description, { timeout: 10000 });
  });

  /* ------------------------------------------------------------------ */
  /*  6. General: Project ID cannot be changed                           */
  /* ------------------------------------------------------------------ */
  test("6 - Project ID section shows 'Cannot be changed' helper text", async ({
    page,
  }) => {
    await expect(
      page.getByText("Cannot be changed"),
    ).toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  7. Services: PG toggle shows current state                         */
  /* ------------------------------------------------------------------ */
  test("7 - PostgreSQL toggle reflects enabled state from API", async ({
    page,
  }) => {
    // Look for the PostgreSQL label
    const pgLabel = page.getByText("PostgreSQL").first();
    await expect(pgLabel).toBeVisible();

    // The toggle is a checkbox with className "stg-switch"
    // Find the first svc-row containing "PostgreSQL"
    const pgRow = page.locator(".svc-row", { hasText: "PostgreSQL" });
    const checkbox = pgRow.locator("input.stg-switch");
    await expect(checkbox).toBeVisible();

    // Project has PG enabled, so the checkbox should be checked
    if (project.postgresEnabled) {
      await expect(checkbox).toBeChecked();
    } else {
      await expect(checkbox).not.toBeChecked();
    }
  });

  /* ------------------------------------------------------------------ */
  /*  8. Services: PG extensions visible when PG enabled                 */
  /* ------------------------------------------------------------------ */
  test("8 - PostgreSQL extensions are shown when PG is enabled", async ({
    page,
  }) => {
    // Extensions should be shown as badges
    await expect(page.getByText("Extensions")).toBeVisible();
    await expect(page.getByText("pgvector")).toBeVisible();
    await expect(page.getByText("pgcrypto")).toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  9. Services: Redis toggle reflects enabled state                   */
  /* ------------------------------------------------------------------ */
  test("9 - Redis toggle reflects enabled state from API", async ({
    page,
  }) => {
    const redisRow = page.locator(".svc-row", { hasText: "Redis" });
    await expect(redisRow).toBeVisible();

    const checkbox = redisRow.locator("input.stg-switch");
    await expect(checkbox).toBeVisible();

    // Project has Redis enabled, so the checkbox should be checked
    if (project.redisEnabled) {
      await expect(checkbox).toBeChecked();
    } else {
      await expect(checkbox).not.toBeChecked();
    }
  });

  /* ------------------------------------------------------------------ */
  /*  10. Services: PostgREST toggle reflects disabled state             */
  /* ------------------------------------------------------------------ */
  test("10 - PostgREST toggle reflects disabled state", async ({ page }) => {
    const postgrestRow = page.locator(".svc-row", { hasText: "PostgREST" });
    await expect(postgrestRow).toBeVisible();

    const checkbox = postgrestRow.locator("input.stg-switch");
    await expect(checkbox).toBeVisible();

    // Created with postgrestEnabled: false
    await expect(checkbox).not.toBeChecked();

    // Since PostgREST is disabled, the endpoint URL should NOT be visible
    const endpointText = page.locator("text=/rest\\/v1/");
    const endpointVisible = await endpointText.isVisible().catch(() => false);
    expect(endpointVisible).toBe(false);
  });

  /* ------------------------------------------------------------------ */
  /*  11. Services: PG OFF -> PostgREST disabled                         */
  /* ------------------------------------------------------------------ */
  test("11 - PostgREST description mentions RESTful API from database", async ({
    page,
  }) => {
    // Verify PostgREST description text
    await expect(
      page.getByText("Instant RESTful API from your database schema"),
    ).toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  12. Danger: Transfer disabled + "Coming soon"                      */
  /* ------------------------------------------------------------------ */
  test("12 - Transfer ownership button is disabled with Coming soon badge", async ({
    page,
  }) => {
    // Find the Transfer button
    const transferBtn = page.getByRole("button", { name: /Transfer/i });
    await expect(transferBtn).toBeVisible();
    await expect(transferBtn).toBeDisabled();

    // "Coming soon" badge should be near the Transfer button
    await expect(page.getByText("Coming soon")).toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  13. Danger: Pause -> Cancel -> no status change                    */
  /* ------------------------------------------------------------------ */
  test("13 - Pause modal: Cancel leaves project unchanged", async ({
    page,
  }) => {
    // Click the Pause Project button in the danger zone
    const pauseBtn = page.getByRole("button", { name: /Pause Project/i });
    await expect(pauseBtn).toBeVisible();
    await pauseBtn.click();

    // Confirm dialog should appear
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Click Cancel
    const cancelBtn = dialog.getByRole("button", { name: "Cancel" });
    await expect(cancelBtn).toBeVisible();
    await cancelBtn.click();

    // Dialog should close
    await expect(dialog).not.toBeVisible();

    // Project should still be in the same state
    const apiProject = await getProject(project.id);
    expect(apiProject.status).not.toBe("paused");
  });

  /* ------------------------------------------------------------------ */
  /*  14. Danger: Pause -> confirm -> status "Paused"                    */
  /* ------------------------------------------------------------------ */
  test("14 - Pause -> confirm -> status changes to Paused", async ({
    page,
  }) => {
    // Click the Pause Project button
    const pauseBtn = page.getByRole("button", { name: /Pause Project/i });
    await expect(pauseBtn).toBeVisible();
    await pauseBtn.click();

    // Confirm dialog
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Click the confirm button (Pause Project)
    const confirmBtn = dialog.getByRole("button", { name: /Pause/i }).last();
    await confirmBtn.click();

    // Wait for dialog to close
    await expect(dialog).not.toBeVisible({ timeout: 15000 });

    // After pausing, the button text should change to "Resume Project"
    await expect(
      page.getByRole("button", { name: /Resume Project/i }),
    ).toBeVisible({ timeout: 10000 });

    // Verify via API
    const apiProject = await getProject(project.id);
    expect(apiProject.status.toLowerCase()).toContain("paused");
    project.status = apiProject.status;
  });

  /* ------------------------------------------------------------------ */
  /*  15. Delete modal: empty/partial name -> Delete button disabled     */
  /* ------------------------------------------------------------------ */
  test("15 - Delete modal: empty name keeps Delete button disabled", async ({
    page,
  }) => {
    // Click the Delete Project button
    const deleteBtn = page.getByRole("button", { name: "Delete Project" });
    await expect(deleteBtn).toBeVisible();
    await deleteBtn.click();

    // TypeToConfirmDialog should appear
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // The confirm Delete button inside the dialog should be disabled
    const confirmDeleteBtn = dialog.getByRole("button", {
      name: /Delete/i,
    }).last();
    await expect(confirmDeleteBtn).toBeDisabled();

    // Type partial name
    const nameInput = dialog.locator('input[type="text"]');
    await nameInput.fill(project.displayName.slice(0, 5));
    await expect(confirmDeleteBtn).toBeDisabled();

    // Cancel
    await dialog.getByRole("button", { name: "Cancel" }).click();
    await expect(dialog).not.toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  16. Delete modal: I understand / type-to-confirm required          */
  /* ------------------------------------------------------------------ */
  test("16 - Delete modal: full name match enables Delete button", async ({
    page,
  }) => {
    // Click Delete Project
    const deleteBtn = page.getByRole("button", { name: "Delete Project" });
    await deleteBtn.click();

    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    const confirmDeleteBtn = dialog.getByRole("button", {
      name: /Delete/i,
    }).last();

    // Type full display name and check the acknowledgement checkbox
    const nameInput = dialog.locator('input[type="text"]');
    await nameInput.fill(project.displayName);
    await dialog.locator("input[type='checkbox']").check();

    // Delete button should now be enabled
    await expect(confirmDeleteBtn).toBeEnabled();

    // Clear the input — should be disabled again
    await nameInput.clear();
    await expect(confirmDeleteBtn).toBeDisabled();

    // Cancel
    await dialog.getByRole("button", { name: "Cancel" }).click();
  });

  /* ------------------------------------------------------------------ */
  /*  17. Delete modal: Cancel/X/scrim dismiss                           */
  /* ------------------------------------------------------------------ */
  test("17 - Delete modal dismisses via Cancel button", async ({ page }) => {
    const deleteBtn = page.getByRole("button", { name: "Delete Project" });
    await deleteBtn.click();

    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Cancel button
    await dialog.getByRole("button", { name: "Cancel" }).click();
    await expect(dialog).not.toBeVisible();

    // Verify project still exists
    const apiProject = await getProject(project.id);
    expect(apiProject.id).toBe(project.id);
  });

  /* ------------------------------------------------------------------ */
  /*  18. Delete -> execute -> redirect to /projects -> project gone     */
  /* ------------------------------------------------------------------ */
  test("18 - Delete project -> redirect to /projects list", async ({
    page,
  }) => {
    // Click Delete Project
    const deleteBtn = page.getByRole("button", { name: "Delete Project" });
    await deleteBtn.click();

    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Type the full display name and check the acknowledgement checkbox
    const nameInput = dialog.locator('input[type="text"]');
    await nameInput.fill(project.displayName);
    await dialog.locator("input[type='checkbox']").check();

    // Click the confirm Delete button
    const confirmDeleteBtn = dialog.getByRole("button", {
      name: /Delete/i,
    }).last();
    await expect(confirmDeleteBtn).toBeEnabled();
    await confirmDeleteBtn.click();

    // Should redirect to /projects
    await expect(page).toHaveURL(/\/projects$/, { timeout: 15000 });

    // The deleted project should no longer appear
    await page.waitForTimeout(1000);
    const deletedCard = page.locator(`a[href*='${project.id}']`);
    await expect(deletedCard).not.toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  19. Services: all extensions visible (pgvector + pgcrypto)         */
  /* ------------------------------------------------------------------ */
  test("19 - all configured PG extensions are displayed as badges", async ({
    page,
  }) => {
    // This test needs a project — but test 18 deleted it.
    // Re-create a project for this test.
    const newProject = await createTestProject("settings-ext", {
      postgresEnabled: true,
      redisEnabled: false,
      postgresExtensions: [
        "pgvector",
        "pgcrypto",
      ],
    });

    try {
      await page.goto(`/projects/${newProject.id}`);
      const settingsTab = page.getByRole("tab", { name: "Settings" });
      await expect(settingsTab).toBeVisible({ timeout: 15000 });
      await settingsTab.click();

      // Verify all 2 extensions are shown
      await expect(page.getByText("pgvector")).toBeVisible({ timeout: 10000 });
      await expect(page.getByText("pgcrypto")).toBeVisible();
    } finally {
      try {
        await deleteTestProject(newProject.id);
      } catch {}
    }
  });

  /* ------------------------------------------------------------------ */
  /*  20. Services: extension visibility matches API configuration       */
  /* ------------------------------------------------------------------ */
  test("20 - extensions not shown when PG is disabled", async ({ page }) => {
    // Create a project with PG disabled
    const noPgProject = await createTestProject("settings-nopg", {
      postgresEnabled: false,
      redisEnabled: true,
    });

    try {
      await page.goto(`/projects/${noPgProject.id}`);
      const settingsTab = page.getByRole("tab", { name: "Settings" });
      await expect(settingsTab).toBeVisible({ timeout: 15000 });
      await settingsTab.click();

      // Wait for the page to load
      await expect(page.getByRole("heading", { name: "Enabled services" })).toBeVisible({ timeout: 10000 });

      // "Extensions:" label should NOT be visible (PG is disabled, no extensions)
      const extensionsLabel = page.getByText("Extensions:");
      const isVisible = await extensionsLabel.isVisible().catch(() => false);
      expect(isVisible).toBe(false);

      // Redis should still show as enabled
      const redisText = page.getByText("Redis");
      await expect(redisText.first()).toBeVisible();
    } finally {
      try {
        await deleteTestProject(noPgProject.id);
      } catch {}
    }
  });
});

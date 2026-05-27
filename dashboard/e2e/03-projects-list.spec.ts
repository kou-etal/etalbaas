import { test, expect, Page } from "@playwright/test";
import {
  createTestProject,
  deleteTestProject,
  listProjects,
  uniqueName,
  cleanupE2eProjects,
} from "./helpers/api";

/**
 * 03 - Projects List Page
 *
 * Covers the /projects page: listing, searching, filtering, card navigation,
 * create-project modal (open/close/submit), keyboard shortcuts, and empty state.
 */

/* ================================================================== */
/*  Section A: Tests with API-seeded projects (read-only interactions) */
/* ================================================================== */
test.describe.serial("Projects list - seeded data", () => {
  let projects: Array<{ id: string; displayName: string }>;

  test.beforeAll(async () => {
    projects = [];
    for (const suffix of ["alpha", "beta", "gamma"]) {
      const p = await createTestProject(suffix);
      projects.push(p);
    }
  });

  test.afterAll(async () => {
    for (const p of projects) {
      try {
        await deleteTestProject(p.id);
      } catch {
        // Ignore errors during cleanup
      }
    }
  });

  /* 1. Page loads with project cards */
  test("1 - page loads with project cards showing", async ({ page }) => {
    await page.goto("/projects");
    await page.waitForLoadState("networkidle");

    // Wait for the page to transition from loading to loaded/empty state
    await page.waitForFunction(
      () => document.body.dataset.state !== "loading",
      { timeout: 20000 },
    );

    // Wait for at least the seeded cards to appear
    const cards = page.locator("a.card[href^='/projects/']");
    await expect(cards.first()).toBeVisible({ timeout: 15000 });

    const count = await cards.count();
    expect(count).toBeGreaterThanOrEqual(3);
  });

  /* 2. Search filters projects */
  test("2 - search filters projects by name", async ({ page }) => {
    await page.goto("/projects");
    const cards = page.locator("a.card[href^='/projects/']");
    await expect(cards.first()).toBeVisible({ timeout: 15000 });

    const totalBefore = await cards.count();

    // Type a keyword unique to one project (the alpha suffix is in the uniqueName)
    const searchInput = page.getByLabel("Search projects");
    await searchInput.fill(projects[0].displayName);

    // After filtering, exactly 1 card should be visible
    await expect(cards).toHaveCount(1);
    await expect(cards.first()).toContainText(projects[0].displayName);

    // Clear search - all cards come back
    await searchInput.clear();
    await expect(cards).toHaveCount(totalBefore);
  });

  /* 3. Filter chips work */
  test("3 - filter chips switch between statuses", async ({ page }) => {
    await page.goto("/projects");
    const cards = page.locator("a.card[href^='/projects/']");
    await expect(cards.first()).toBeVisible({ timeout: 15000 });

    const allCount = await cards.count();

    // Click "Ready" chip
    await page.locator("button.chip", { hasText: "Ready" }).click();

    // The visible cards should only be "ready" ones (could be 0 or more)
    const readyCards = page.locator("a.card[href^='/projects/']");
    const readyCount = await readyCards.count();
    expect(readyCount).toBeLessThanOrEqual(allCount);

    // Each visible card should have a "Ready" badge
    for (let i = 0; i < readyCount; i++) {
      await expect(readyCards.nth(i).locator(".badge")).toContainText("Ready");
    }

    // Click "All" chip to restore
    await page.locator("button.chip", { hasText: "All" }).click();
    await expect(cards).toHaveCount(allCount);
  });

  /* 4. Card click navigates to project detail */
  test("4 - card click navigates to project detail page", async ({ page }) => {
    await page.goto("/projects");
    const cards = page.locator("a.card[href^='/projects/']");
    await expect(cards.first()).toBeVisible({ timeout: 15000 });

    // Read the href of the first card
    const href = await cards.first().getAttribute("href");
    expect(href).toBeTruthy();

    await cards.first().click();
    await expect(page).toHaveURL(new RegExp(`/projects/[a-zA-Z0-9-]+$`));
  });

  /* 5. Modal dismiss methods: Cancel, X button, scrim click */
  test("5 - modal dismiss via Cancel, X button, and scrim click", async ({ page }) => {
    await page.goto("/projects");
    await expect(page.locator("#new-project")).toBeVisible({ timeout: 15000 });

    const dialog = page.getByRole("dialog");

    // --- Cancel button ---
    await page.locator("#new-project").click();
    await expect(dialog).toBeVisible();
    await dialog.getByRole("button", { name: "Cancel" }).click();
    await expect(dialog).not.toBeVisible();

    // --- X (close) button ---
    await page.locator("#new-project").click();
    await expect(dialog).toBeVisible();
    await dialog.getByRole("button", { name: "Close" }).click();
    await expect(dialog).not.toBeVisible();

    // --- Scrim click ---
    await page.locator("#new-project").click();
    await expect(dialog).toBeVisible();

    // Click the scrim (the dialog backdrop element itself, not the inner .modal)
    // The scrim is the outer element with role="dialog"
    const scrim = page.locator(".modal-scrim.open");
    const scrimBox = await scrim.boundingBox();
    expect(scrimBox).toBeTruthy();
    // Click top-left corner (outside the centered .modal)
    await page.mouse.click(scrimBox!.x + 5, scrimBox!.y + 5);
    await expect(dialog).not.toBeVisible();
  });

  /* 6. Modal: empty name disables Create button */
  test("6 - empty name disables Create button", async ({ page }) => {
    await page.goto("/projects");
    await page.locator("#new-project").click();

    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    const submitBtn = dialog.locator("#cp-submit");
    const nameInput = dialog.locator("#cp-name");

    // Initially the name is empty, so Create should be disabled
    await expect(submitBtn).toBeDisabled();

    // Type a valid name
    await nameInput.fill("test-project-name");
    await expect(submitBtn).toBeEnabled();

    // Clear the name
    await nameInput.clear();
    await expect(submitBtn).toBeDisabled();

    // Close
    await dialog.getByRole("button", { name: "Cancel" }).click();
  });

  /* 7. PostgREST disabled when PG is off */
  test("7 - PostgREST checkbox disabled when PostgreSQL is off", async ({ page }) => {
    await page.goto("/projects");
    await page.locator("#new-project").click();

    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Find PG and PostgREST checkboxes by their parent svc-card labels
    const pgCheckbox = dialog.locator(".svc-card .icon.pg").locator("..").locator("input[type='checkbox']");
    const apiCheckbox = dialog.locator(".svc-card .icon.api").locator("..").locator("input[type='checkbox']");

    // Default: PG is ON, PostgREST should be enabled
    await expect(pgCheckbox).toBeChecked();
    await expect(apiCheckbox).toBeEnabled();

    // Turn PG off
    await pgCheckbox.uncheck();
    await expect(pgCheckbox).not.toBeChecked();

    // PostgREST should now be disabled (and unchecked)
    await expect(apiCheckbox).toBeDisabled();

    // Turn PG back on
    await pgCheckbox.check();
    await expect(apiCheckbox).toBeEnabled();

    await dialog.getByRole("button", { name: "Cancel" }).click();
  });

  /* 12. Keyboard / focuses search */
  test("12 - keyboard / focuses search input", async ({ page }) => {
    await page.goto("/projects");
    const searchInput = page.getByLabel("Search projects");
    await expect(searchInput).toBeVisible({ timeout: 15000 });

    // Press / key
    await page.keyboard.press("/");
    await expect(searchInput).toBeFocused();

    // Type a filter query while focused
    await page.keyboard.type(projects[0].displayName);

    const cards = page.locator("a.card[href^='/projects/']");
    await expect(cards).toHaveCount(1);
    await expect(cards.first()).toContainText(projects[0].displayName);
  });

  /* 13. Keyboard Escape closes modal */
  test("13 - keyboard Escape closes modal", async ({ page }) => {
    await page.goto("/projects");
    await expect(page.locator("#new-project")).toBeVisible({ timeout: 15000 });

    const dialog = page.getByRole("dialog");

    // Press N to open modal
    await page.keyboard.press("n");
    await expect(dialog).toBeVisible();

    // Press Escape to close
    await page.keyboard.press("Escape");
    await expect(dialog).not.toBeVisible();
  });

  /* 17. Project count in header correct */
  test("17 - All filter chip count matches card count", async ({ page }) => {
    await page.goto("/projects");
    const cards = page.locator("a.card[href^='/projects/']");
    await expect(cards.first()).toBeVisible({ timeout: 15000 });

    const cardCount = await cards.count();

    // The "All" chip displays the count in a .count span
    const allChip = page.locator("button.chip", { hasText: "All" });
    const countSpan = allChip.locator(".count");
    await expect(countSpan).toHaveText(String(cardCount));
  });
});

/* ================================================================== */
/*  Section B: Tests that create projects via modal                    */
/* ================================================================== */
test.describe.serial("Projects list - create via modal", () => {
  const createdIds: string[] = [];

  test.afterAll(async () => {
    for (const id of createdIds) {
      try {
        await deleteTestProject(id);
      } catch {
        // Ignore cleanup errors
      }
    }
  });

  /** Helper: fill modal, submit, wait for modal close, return project card */
  async function createViaModal(
    page: Page,
    name: string,
    opts: {
      description?: string;
      pg?: boolean;
      redis?: boolean;
      postgrest?: boolean;
      pgvector?: boolean;
      pgcrypto?: boolean;
    } = {},
  ): Promise<void> {
    await page.locator("#new-project").click();
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Fill name
    await dialog.locator("#cp-name").fill(name);

    // Fill description
    if (opts.description) {
      await dialog.locator("#cp-desc").fill(opts.description);
    }

    // PG toggle (default is ON in modal)
    const pgCheckbox = dialog.locator(".svc-card .icon.pg").locator("..").locator("input[type='checkbox']");
    if (opts.pg === false) {
      await pgCheckbox.uncheck();
    } else if (opts.pg === true) {
      await pgCheckbox.check();
    }

    // Redis toggle (default is OFF)
    const rdCheckbox = dialog.locator(".svc-card .icon.rd").locator("..").locator("input[type='checkbox']");
    if (opts.redis === true) {
      await rdCheckbox.check();
    } else if (opts.redis === false) {
      await rdCheckbox.uncheck();
    }

    // PostgREST toggle (default is ON when PG is on)
    const apiCheckbox = dialog.locator(".svc-card .icon.api").locator("..").locator("input[type='checkbox']");
    if (opts.postgrest === false) {
      await apiCheckbox.uncheck();
    } else if (opts.postgrest === true) {
      await apiCheckbox.check();
    }

    // Extension checkboxes
    if (opts.pgvector) {
      await dialog.locator(".ext-row", { hasText: "pgvector" }).locator("input[type='checkbox']").check();
    }
    if (opts.pgcrypto) {
      await dialog.locator(".ext-row", { hasText: "pgcrypto" }).locator("input[type='checkbox']").check();
    }

    // Submit
    await dialog.locator("#cp-submit").click();

    // Wait for modal to close (API call + success view ~1.4s + close)
    await expect(dialog).not.toBeVisible({ timeout: 30000 });
  }

  /* 8. Create project (PG only) */
  test("8 - create project with PG only", async ({ page }) => {
    const projectName = uniqueName("pg-only");

    await page.goto("/projects");
    await expect(page.locator("#new-project")).toBeVisible({ timeout: 15000 });

    await createViaModal(page, projectName, {
      description: "E2E test: PG only",
      pg: true,
      redis: false,
      postgrest: false,
    });

    // Toast should appear
    await expect(page.locator(".toast.show")).toBeVisible();

    // Card should appear in the list
    const newCard = page.locator("a.card", { hasText: projectName });
    await expect(newCard).toBeVisible({ timeout: 10000 });

    // Extract ID from href for cleanup
    const href = await newCard.getAttribute("href");
    if (href) {
      const id = href.replace("/projects/", "");
      createdIds.push(id);
    }
  });

  /* 9. Create project (all services) */
  test("9 - create project with all services and extensions", async ({ page }) => {
    const projectName = uniqueName("all-svcs");

    await page.goto("/projects");
    await expect(page.locator("#new-project")).toBeVisible({ timeout: 15000 });

    await createViaModal(page, projectName, {
      description: "E2E test: all services",
      pg: true,
      redis: true,
      postgrest: true,
      pgvector: true,
      pgcrypto: true,
    });

    // Card should appear
    const newCard = page.locator("a.card", { hasText: projectName });
    await expect(newCard).toBeVisible({ timeout: 10000 });

    // Verify service indicators are present and not dimmed
    const pgSvc = newCard.locator(".svc.pg");
    const rdSvc = newCard.locator(".svc.rd");
    const apiSvc = newCard.locator(".svc.api");
    await expect(pgSvc).toBeVisible();
    await expect(rdSvc).toBeVisible();
    await expect(apiSvc).toBeVisible();

    // PG, RD, API should NOT have the "dimmed" class
    await expect(pgSvc).not.toHaveClass(/dimmed/);
    await expect(rdSvc).not.toHaveClass(/dimmed/);
    await expect(apiSvc).not.toHaveClass(/dimmed/);

    // Extract ID for cleanup
    const href = await newCard.getAttribute("href");
    if (href) {
      const id = href.replace("/projects/", "");
      createdIds.push(id);
    }
  });

  /* 10. Create project (no services) */
  test("10 - create project with no services", async ({ page }) => {
    const projectName = uniqueName("no-svcs");

    await page.goto("/projects");
    await expect(page.locator("#new-project")).toBeVisible({ timeout: 15000 });

    await createViaModal(page, projectName, {
      description: "E2E test: no services",
      pg: false,
      redis: false,
    });

    // Card should appear
    const newCard = page.locator("a.card", { hasText: projectName });
    await expect(newCard).toBeVisible({ timeout: 10000 });

    // All service indicators should be dimmed
    await expect(newCard.locator(".svc.pg")).toHaveClass(/dimmed/);
    await expect(newCard.locator(".svc.rd")).toHaveClass(/dimmed/);
    await expect(newCard.locator(".svc.api")).toHaveClass(/dimmed/);

    // Extract ID for cleanup
    const href = await newCard.getAttribute("href");
    if (href) {
      const id = href.replace("/projects/", "");
      createdIds.push(id);
    }
  });

  /* 11. Keyboard N opens modal, Ctrl+Enter submits */
  test("11 - keyboard N opens modal, Ctrl+Enter creates project", async ({ page }) => {
    const projectName = uniqueName("kbd-create");

    await page.goto("/projects");
    await expect(page.locator("#new-project")).toBeVisible({ timeout: 15000 });

    const dialog = page.getByRole("dialog");

    // Ensure focus is not on an input (keyboard shortcut ignores inputs)
    await page.locator("h1").first().click();

    // Press N to open modal
    await page.keyboard.press("n");
    await expect(dialog).toBeVisible();

    // Type project name
    await dialog.locator("#cp-name").fill(projectName);

    // Ensure Create button is enabled
    await expect(dialog.locator("#cp-submit")).toBeEnabled();

    // Ctrl+Enter to submit
    await page.keyboard.press("Control+Enter");

    // Wait for modal to close
    await expect(dialog).not.toBeVisible({ timeout: 20000 });

    // Card should appear
    const newCard = page.locator("a.card", { hasText: projectName });
    await expect(newCard).toBeVisible({ timeout: 10000 });

    // Extract ID for cleanup
    const href = await newCard.getAttribute("href");
    if (href) {
      const id = href.replace("/projects/", "");
      createdIds.push(id);
    }
  });

  /* 15. Search finds created project */
  test("15 - search finds a project created via modal", async ({ page }) => {
    // Use the first created project (test 8)
    test.skip(createdIds.length === 0, "No projects were created in prior tests");

    await page.goto("/projects");
    const cards = page.locator("a.card[href^='/projects/']");
    await expect(cards.first()).toBeVisible({ timeout: 15000 });

    // Search for the PG-only project name (starts with "e2e-pg-only")
    const searchInput = page.getByLabel("Search projects");
    await searchInput.fill("e2e-pg-only");

    // Should find at least 1 result
    await expect(cards.first()).toBeVisible();
    const count = await cards.count();
    expect(count).toBeGreaterThanOrEqual(1);
  });

  /* 16. Created project detail navigation */
  test("16 - click created project card navigates to detail", async ({ page }) => {
    test.skip(createdIds.length === 0, "No projects were created in prior tests");

    await page.goto("/projects");
    const cards = page.locator("a.card[href^='/projects/']");
    await expect(cards.first()).toBeVisible({ timeout: 15000 });

    // Find a card for the first created project
    const targetCard = page.locator(`a.card[href='/projects/${createdIds[0]}']`);
    await expect(targetCard).toBeVisible();

    // Read name for later verification
    const projectNameText = await targetCard.locator(".name").textContent();

    await targetCard.click();
    await expect(page).toHaveURL(new RegExp(`/projects/${createdIds[0]}$`));

    // Project name should be visible on the detail page
    if (projectNameText) {
      await expect(page.getByText(projectNameText).first()).toBeVisible({ timeout: 10000 });
    }
  });

  /* 18. Filter chip counts update after creation */
  test("18 - filter chip counts reflect created projects", async ({ page }) => {
    await page.goto("/projects");
    const cards = page.locator("a.card[href^='/projects/']");
    await expect(cards.first()).toBeVisible({ timeout: 15000 });

    const cardCount = await cards.count();

    // "All" chip count should equal card count
    const allChip = page.locator("button.chip", { hasText: "All" });
    const allCount = allChip.locator(".count");
    await expect(allCount).toHaveText(String(cardCount));

    // Sum of individual filters should equal total
    const readyChip = page.locator("button.chip", { hasText: "Ready" });
    const activeChip = page.locator("button.chip", { hasText: "Active" });
    const issuesChip = page.locator("button.chip", { hasText: "Issues" });

    const readyCount = parseInt((await readyChip.locator(".count").textContent()) || "0", 10);
    const activeCount = parseInt((await activeChip.locator(".count").textContent()) || "0", 10);
    const issuesCount = parseInt((await issuesChip.locator(".count").textContent()) || "0", 10);

    // The individual filter counts should sum to the total
    expect(readyCount + activeCount + issuesCount).toBe(cardCount);
  });
});

/* ================================================================== */
/*  Section C: Empty state test                                        */
/* ================================================================== */
test.describe.serial("Projects list - empty state", () => {
  /* 14. Empty state shows when no projects exist */
  test("14 - empty state shows and New Project button works", async ({ page }) => {
    // Clean up all e2e- prefixed projects to reach empty state
    // This is inherently destructive, so we only do it if we can safely assume
    // that the only projects are e2e-test projects.
    const allProjects = await listProjects();
    const hasNonE2eProjects = allProjects.some(
      (p) => !p.displayName.startsWith("e2e-"),
    );

    if (hasNonE2eProjects) {
      // There are real projects we should not delete - skip the empty state test
      test.skip(true, "Non-e2e projects exist; cannot safely test empty state");
      return;
    }

    // Delete all e2e projects
    await cleanupE2eProjects();

    await page.goto("/projects");

    // The empty-block should be visible
    const emptyBlock = page.locator("#empty-block");
    await expect(emptyBlock).toBeVisible({ timeout: 15000 });
    await expect(emptyBlock.getByText("No projects yet")).toBeVisible();

    // Click the "New Project" button inside the empty state
    await emptyBlock.getByRole("button", { name: /New Project/i }).click();

    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Create a project from the empty state
    const projectName = uniqueName("empty-state");
    await dialog.locator("#cp-name").fill(projectName);
    await dialog.locator("#cp-submit").click();

    // Wait for modal to close
    await expect(dialog).not.toBeVisible({ timeout: 20000 });

    // Card should now appear and empty state should be gone
    const newCard = page.locator("a.card", { hasText: projectName });
    await expect(newCard).toBeVisible({ timeout: 10000 });
    await expect(emptyBlock).not.toBeVisible();

    // Clean up the project we just created
    const href = await newCard.getAttribute("href");
    if (href) {
      const id = href.replace("/projects/", "");
      try {
        await deleteTestProject(id);
      } catch {
        // Ignore
      }
    }
  });
});

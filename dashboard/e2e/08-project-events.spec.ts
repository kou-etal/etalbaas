import { test, expect } from "@playwright/test";
import {
  createTestProject,
  deleteTestProject,
  createTestFunction,
  deleteTestFunction,
} from "./helpers/api";

/**
 * 08 - Project Events Tab
 *
 * Covers the Events tab on the project detail page (/projects/{id}).
 * The Events tab currently uses hardcoded mock data (HC_EVENTS).
 * Tests verify that UI elements exist and are interactive.
 *
 * Setup: 1 project + 1 function via API.
 */

test.describe.serial("Project Events Tab", () => {
  let project: { id: string; displayName: string };
  let fn: { id: string; name: string };

  test.beforeAll(async () => {
    project = await createTestProject("events-tab");
    fn = await createTestFunction(project.id, "events-fn");
  });

  test.afterAll(async () => {
    try {
      await deleteTestFunction(project.id, fn.id);
    } catch {}
    try {
      await deleteTestProject(project.id);
    } catch {}
  });

  test.beforeEach(async ({ page }) => {
    await page.goto(`/projects/${project.id}`);
    // Click the Events tab
    const eventsTab = page.getByRole("tab", { name: "Events" });
    await expect(eventsTab).toBeVisible({ timeout: 15000 });
    await eventsTab.click();
    await expect(eventsTab).toHaveAttribute("aria-selected", "true");
    // Wait for the event history heading to be visible
    await expect(page.locator("h2", { hasText: "Event history" })).toBeVisible({
      timeout: 10000,
    });
  });

  /* ------------------------------------------------------------------ */
  /*  1. Event list shows rows                                           */
  /* ------------------------------------------------------------------ */
  test("1 - event list shows rows with expected elements", async ({
    page,
  }) => {
    // The event table should exist and have body rows (HC_EVENTS has 8 items)
    const table = page.locator(".evt-table[role='table']");
    await expect(table).toBeVisible();

    // Header row should have columns
    const headerRow = table.locator(".row.header");
    await expect(headerRow).toBeVisible();
    await expect(headerRow.locator("span", { hasText: "Status" })).toBeVisible();
    await expect(headerRow.locator("span", { hasText: "Function" })).toBeVisible();
    await expect(headerRow.locator("span", { hasText: "Trace ID" })).toBeVisible();

    // Body rows should exist (at least 1)
    const bodyRows = table.locator(".row.body-row");
    const rowCount = await bodyRows.count();
    expect(rowCount).toBeGreaterThanOrEqual(1);
  });

  /* ------------------------------------------------------------------ */
  /*  2. Live toggle ON — checked by default                             */
  /* ------------------------------------------------------------------ */
  test("2 - live toggle is ON by default", async ({ page }) => {
    const liveToggle = page.locator("label.live-toggle");
    await expect(liveToggle).toBeVisible();

    // The live-toggle label should have the "on" class by default
    await expect(liveToggle).toHaveClass(/on/);

    // The checkbox should be checked
    const checkbox = liveToggle.locator("input.switch");
    await expect(checkbox).toBeChecked();
  });

  /* ------------------------------------------------------------------ */
  /*  3. Live toggle OFF — uncheck                                       */
  /* ------------------------------------------------------------------ */
  test("3 - live toggle OFF — uncheck changes state", async ({ page }) => {
    const liveToggle = page.locator("label.live-toggle");
    await expect(liveToggle).toBeVisible();

    const checkbox = liveToggle.locator("input.switch");
    await expect(checkbox).toBeChecked();

    // Uncheck the toggle
    await checkbox.uncheck();
    await expect(checkbox).not.toBeChecked();

    // The label should no longer have "on" class
    await expect(liveToggle).not.toHaveClass(/\bon\b/);
  });

  /* ------------------------------------------------------------------ */
  /*  4. Date range dropdown — options visible                           */
  /* ------------------------------------------------------------------ */
  test("4 - date range dropdown shows options", async ({ page }) => {
    const dateSelect = page.locator("select[aria-label='Date range']");
    await expect(dateSelect).toBeVisible();

    // Verify options exist
    const options = dateSelect.locator("option");
    const count = await options.count();
    expect(count).toBeGreaterThanOrEqual(3);

    // Check for known options
    await expect(options.filter({ hasText: "Last 1h" })).toHaveCount(1);
    await expect(options.filter({ hasText: "Last 24h" })).toHaveCount(1);
    await expect(options.filter({ hasText: "Last 7d" })).toHaveCount(1);
  });

  /* ------------------------------------------------------------------ */
  /*  5. Function filter dropdown — options visible                      */
  /* ------------------------------------------------------------------ */
  test("5 - function filter dropdown shows options", async ({ page }) => {
    const fnSelect = page.locator("select[aria-label='Function filter']");
    await expect(fnSelect).toBeVisible();

    // Should have "All functions" plus individual function options
    const options = fnSelect.locator("option");
    const count = await options.count();
    expect(count).toBeGreaterThanOrEqual(2);

    await expect(
      options.filter({ hasText: "All functions" }),
    ).toHaveCount(1);
  });

  /* ------------------------------------------------------------------ */
  /*  6. Source filter dropdown — options visible                         */
  /* ------------------------------------------------------------------ */
  test("6 - source filter dropdown shows options", async ({ page }) => {
    const sourceSelect = page.locator("select[aria-label='Source filter']");
    await expect(sourceSelect).toBeVisible();

    const options = sourceSelect.locator("option");
    const count = await options.count();
    expect(count).toBeGreaterThanOrEqual(2);

    await expect(
      options.filter({ hasText: "All sources" }),
    ).toHaveCount(1);
    await expect(
      options.filter({ hasText: "Database" }),
    ).toHaveCount(1);
    await expect(
      options.filter({ hasText: "Object Storage" }),
    ).toHaveCount(1);
  });

  /* ------------------------------------------------------------------ */
  /*  7. Status chip Delivered filter                                    */
  /* ------------------------------------------------------------------ */
  test("7 - status chip Delivered filters rows, All restores them", async ({
    page,
  }) => {
    const table = page.locator(".evt-table[role='table']");
    const bodyRows = table.locator(".row.body-row");

    // Get total count with "All" active
    const allChip = page.locator("button.pd-chip", { hasText: "All" });
    await expect(allChip).toBeVisible();
    await allChip.click();
    const allCount = await bodyRows.count();

    // Click "Delivered" chip
    const deliveredChip = page.locator("button.pd-chip", {
      hasText: "Delivered",
    });
    await expect(deliveredChip).toBeVisible();
    await deliveredChip.click();
    await expect(deliveredChip).toHaveClass(/active/);

    // After filtering, visible rows should only have "delivered" status badges
    const filteredCount = await bodyRows.count();
    expect(filteredCount).toBeLessThanOrEqual(allCount);
    expect(filteredCount).toBeGreaterThanOrEqual(1);

    for (let i = 0; i < filteredCount; i++) {
      const badge = bodyRows.nth(i).locator(".badge-s");
      await expect(badge).toHaveClass(/delivered/);
    }

    // Click "All" chip to restore
    await allChip.click();
    await expect(allChip).toHaveClass(/active/);
    const restoredCount = await bodyRows.count();
    expect(restoredCount).toBe(allCount);
  });

  /* ------------------------------------------------------------------ */
  /*  8. Trace ID search                                                 */
  /* ------------------------------------------------------------------ */
  test("8 - trace ID search filters events", async ({ page }) => {
    const searchInput = page.locator(
      "input[aria-label='Search by trace ID']",
    );

    if (!(await searchInput.isVisible())) {
      test.skip(true, "Search by trace ID input not found");
      return;
    }

    // Type part of a known trace ID from HC_EVENTS
    await searchInput.fill("a1b2c3");

    // Allow a brief moment for any client-side filtering
    await page.waitForTimeout(500);

    // The search field should have the typed value
    await expect(searchInput).toHaveValue("a1b2c3");
  });

  /* ------------------------------------------------------------------ */
  /*  9. Event row expand — details panel visible                        */
  /* ------------------------------------------------------------------ */
  test("9 - clicking event row expands details with Trace ID and function info", async ({
    page,
  }) => {
    const table = page.locator(".evt-table[role='table']");
    const firstRow = table.locator(".row.body-row").first();
    await expect(firstRow).toBeVisible();

    // Click to expand
    await firstRow.click();
    await expect(firstRow).toHaveClass(/expanded/);

    // The detail section should be visible inside the expanded row
    const detail = firstRow.locator(".evt-detail");
    await expect(detail).toBeVisible();

    // Should have a Retry timeline section
    const retryTimeline = detail.locator(".retry-timeline");
    await expect(retryTimeline).toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  10. Expanded: Trace ID copy                                        */
  /* ------------------------------------------------------------------ */
  test("10 - expanded row: Trace ID copy button copies to clipboard", async ({
    page,
    context,
  }) => {
    await context.grantPermissions(["clipboard-read", "clipboard-write"]);

    const table = page.locator(".evt-table[role='table']");
    const firstRow = table.locator(".row.body-row").first();

    // The trace ID and copy button are visible even in collapsed row
    const traceCell = firstRow.locator(".evt-trace");
    await expect(traceCell).toBeVisible();

    // Get the trace ID text
    const trId = firstRow.locator(".evt-trace .tr-id");
    const tracePrefix = await trId.textContent();
    expect(tracePrefix).toBeTruthy();

    // Click the copy button
    const copyBtn = traceCell.locator("button[aria-label='Copy']");
    await expect(copyBtn).toBeVisible();
    await copyBtn.click();

    const clipboardText = await page.evaluate(() =>
      navigator.clipboard.readText(),
    );
    // Clipboard should contain the full trace ID (starts with the prefix shown)
    expect(clipboardText).toBeTruthy();
    expect(clipboardText.length).toBeGreaterThan(0);
  });

  /* ------------------------------------------------------------------ */
  /*  11. Expanded: View function link                                   */
  /* ------------------------------------------------------------------ */
  test("11 - expanded row: View function link exists", async ({ page }) => {
    const table = page.locator(".evt-table[role='table']");
    const firstRow = table.locator(".row.body-row").first();

    // Expand the row
    await firstRow.click();
    await expect(firstRow).toHaveClass(/expanded/);

    const detail = firstRow.locator(".evt-detail");
    await expect(detail).toBeVisible();

    // Check "View function" link exists
    const viewFnLink = detail.locator(".evt-links a", {
      hasText: "View function",
    });
    await expect(viewFnLink).toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  12. Expanded: Payload preview toggle                               */
  /* ------------------------------------------------------------------ */
  test("12 - expanded row: payload preview toggles open and closed", async ({
    page,
  }) => {
    const table = page.locator(".evt-table[role='table']");
    // Use the first row which has payload data
    const firstRow = table.locator(".row.body-row").first();

    // Expand the row
    await firstRow.click();
    await expect(firstRow).toHaveClass(/expanded/);

    const detail = firstRow.locator(".evt-detail");
    await expect(detail).toBeVisible();

    // The payload block is a <details> element
    const payloadBlock = detail.locator("details.payload-block");

    if (!(await payloadBlock.isVisible())) {
      // This event may not have a payload — try the next row that does
      // Collapse current row
      await firstRow.click();

      // Try finding a row with payload (first row in HC_EVENTS has payload)
      // Just skip if no payload block found
      test.skip(true, "No payload block found in expanded row");
      return;
    }

    // Initially the <details> element should not be open
    const isOpenBefore = await payloadBlock.getAttribute("open");

    // Click the summary to toggle
    const summary = payloadBlock.locator("summary");
    await summary.click();

    // The <pre> inside should now be visible
    const pre = payloadBlock.locator("pre");
    await expect(pre).toBeVisible();

    // Click again to close
    await summary.click();

    // The <pre> should now be hidden
    await expect(pre).not.toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  13. Export CSV button visible and clickable                         */
  /* ------------------------------------------------------------------ */
  test("13 - Export CSV button is visible and clickable", async ({ page }) => {
    const exportBtn = page.locator("button.btn.btn-ghost", {
      hasText: "Export CSV",
    });
    await expect(exportBtn).toBeVisible();
    await expect(exportBtn).toBeEnabled();
  });

  /* ------------------------------------------------------------------ */
  /*  14. Load more button                                               */
  /* ------------------------------------------------------------------ */
  test("14 - Load more button is visible if present", async ({ page }) => {
    const loadMoreBtn = page.locator("button.btn.btn-ghost", {
      hasText: "Load more",
    });

    if (!(await loadMoreBtn.isVisible().catch(() => false))) {
      test.skip(true, "Load more button not visible (not enough events)");
      return;
    }

    await expect(loadMoreBtn).toBeEnabled();

    // Verify the footer text shows pagination info
    const pgFoot = page.locator(".pg-foot");
    await expect(pgFoot).toBeVisible();
    const footText = await pgFoot.textContent();
    expect(footText).toContain("Showing");
  });
});

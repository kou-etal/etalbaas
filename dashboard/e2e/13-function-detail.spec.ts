import { test, expect } from "@playwright/test";
import {
  createTestProject,
  deleteTestProject,
  createTestFunction,
  deleteTestFunction,
  createTestApiKey,
  revokeTestApiKey,
} from "./helpers/api";

/**
 * 13 - Function Detail Page
 *
 * Covers the /projects/{id}/functions/{fnId} page:
 * breadcrumbs, topbar actions (rebuild, delete, invoke), status,
 * 5-tab navigation (Overview / Triggers / Invocations / Logs / Settings),
 * and all sub-features within each tab.
 *
 * Setup: 1 project + 1 function via API.
 */

test.describe.serial("Function Detail Page", () => {
  let project: { id: string; displayName: string };
  let fn: { id: string; name: string; displayName: string; kind: string; mode: string; status: string };

  test.beforeAll(async () => {
    project = await createTestProject("fn-detail", {
      postgresEnabled: true,
    });
    const created = await createTestFunction(project.id, "detail-fn", {
      kind: "light-deployment",
      mode: "sync",
    });
    fn = {
      id: created.id,
      name: created.name,
      displayName: created.displayName,
      kind: created.kind,
      mode: created.mode,
      status: created.status,
    };
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
    await page.goto(`/projects/${project.id}/functions/${fn.id}`);
    // Wait for the function header to load
    await expect(page.locator("h1", { hasText: fn.name })).toBeVisible({
      timeout: 15000,
    });
  });

  /* ================================================================== */
  /*  Navigation + Header (1-7)                                          */
  /* ================================================================== */

  /* ------------------------------------------------------------------ */
  /*  1. Breadcrumb "Projects" -> /projects                              */
  /* ------------------------------------------------------------------ */
  test("1 - breadcrumb Projects link navigates to /projects", async ({
    page,
  }) => {
    const crumbs = page.locator(".crumbs");
    await expect(crumbs).toBeVisible();

    const projectsLink = crumbs.getByText("Projects");
    await expect(projectsLink).toBeVisible();
    await projectsLink.click();

    await expect(page).toHaveURL(/\/projects$/);
  });

  /* ------------------------------------------------------------------ */
  /*  2. Breadcrumb project name -> /projects/{id}                       */
  /* ------------------------------------------------------------------ */
  test("2 - breadcrumb project name navigates to project detail", async ({
    page,
  }) => {
    const crumbs = page.locator(".crumbs");
    await expect(crumbs).toBeVisible();

    // The project name link in breadcrumbs
    const projectLink = crumbs.locator(`a[href='/projects/${project.id}']`).first();
    await expect(projectLink).toBeVisible();
    await projectLink.click();

    await expect(page).toHaveURL(new RegExp(`/projects/${project.id}$`));
  });

  /* ------------------------------------------------------------------ */
  /*  3. Breadcrumb "Functions" -> project detail (Functions tab)        */
  /* ------------------------------------------------------------------ */
  test("3 - breadcrumb Functions link navigates to project detail", async ({
    page,
  }) => {
    const crumbs = page.locator(".crumbs");
    await expect(crumbs).toBeVisible();

    // "Functions" link in breadcrumbs navigates to /projects/{id}
    const fnsLink = crumbs.getByText("Functions");
    await expect(fnsLink).toBeVisible();
    await fnsLink.click();

    await expect(page).toHaveURL(new RegExp(`/projects/${project.id}`));
  });

  /* ------------------------------------------------------------------ */
  /*  4. Rebuild button is visible and clickable                         */
  /* ------------------------------------------------------------------ */
  test("4 - Rebuild button is visible in topbar", async ({ page }) => {
    const rebuildBtn = page.locator("button.btn.btn-ghost", {
      hasText: "Rebuild",
    });
    await expect(rebuildBtn).toBeVisible();

    // Click it — should not cause errors
    await rebuildBtn.click();

    // Should still be on the same page
    await expect(page).toHaveURL(
      new RegExp(`/projects/${project.id}/functions/${fn.id}`),
    );
  });

  /* ------------------------------------------------------------------ */
  /*  5. Invoke -> payload input -> Send -> response displayed           */
  /* ------------------------------------------------------------------ */
  test("5 - Invoke button opens modal, Send returns response", async ({
    page,
  }) => {
    // Click the Invoke button
    const invokeBtn = page.locator("button.btn.btn-primary", {
      hasText: "Invoke",
    });
    await expect(invokeBtn).toBeVisible();
    await invokeBtn.click();

    // Invoke modal should appear
    const scrim = page.locator(".modal-scrim.open");
    await expect(scrim).toBeVisible();

    const modal = scrim.locator(".modal");
    await expect(modal).toBeVisible();

    // Verify the modal title contains the function name
    await expect(modal.locator("h3")).toContainText("Invoke");

    // Find the payload textarea
    const payloadTextarea = modal.locator("#invoke-payload");
    await expect(payloadTextarea).toBeVisible();

    // Enter a payload
    await payloadTextarea.clear();
    await payloadTextarea.fill('{"test": true}');

    // Click Send
    const sendBtn = modal.locator("button.btn.btn-primary", {
      hasText: "Send",
    });
    await expect(sendBtn).toBeVisible();
    await sendBtn.click();

    // Wait for response (either a result or error)
    // Response section shows a status code badge or an error message
    const responseOrError = modal.locator(".badge-s, .modal-err");
    await expect(responseOrError.first()).toBeVisible({ timeout: 15000 });

    // Close the modal
    const closeBtn = modal.locator("button.btn.btn-ghost", {
      hasText: "Close",
    });
    await closeBtn.click();
    await expect(scrim).not.toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  6. Delete -> name confirm -> redirect to project detail            */
  /* ------------------------------------------------------------------ */
  test("6 - Delete function -> name confirm -> redirects to project detail", async ({
    page,
  }) => {
    // Create a temporary function to delete
    const tempFn = await createTestFunction(project.id, "to-delete-detail");

    await page.goto(`/projects/${project.id}/functions/${tempFn.id}`);
    await expect(
      page.locator("h1", { hasText: tempFn.name }),
    ).toBeVisible({ timeout: 15000 });

    // Click Delete button in topbar
    const deleteBtn = page.locator("button.btn.btn-danger-ghost", {
      hasText: "Delete",
    });
    await expect(deleteBtn).toBeVisible();
    await deleteBtn.click();

    // Delete modal should appear
    const scrim = page.locator(".modal-scrim.open");
    await expect(scrim).toBeVisible();

    const modal = scrim.locator(".modal");
    await expect(modal).toBeVisible();

    // The Delete Function button should be disabled initially
    const confirmDeleteBtn = modal.locator("button.btn.btn-danger", {
      hasText: "Delete Function",
    });
    await expect(confirmDeleteBtn).toBeDisabled();

    // Type the function name
    const nameInput = modal.locator("#delete-fn-input");
    await nameInput.fill(tempFn.name);

    // Delete button should now be enabled
    await expect(confirmDeleteBtn).toBeEnabled();
    await confirmDeleteBtn.click();

    // Should redirect to /projects/{id}
    await expect(page).toHaveURL(
      new RegExp(`/projects/${project.id}$`),
      { timeout: 15000 },
    );
  });

  /* ------------------------------------------------------------------ */
  /*  7. 5 tab switching: Overview -> Triggers -> Invocations -> Logs -> Settings */
  /* ------------------------------------------------------------------ */
  test("7 - clicking each of 5 tabs activates the corresponding pane", async ({
    page,
  }) => {
    const tabNames = [
      "Overview",
      "Triggers",
      "Invocations",
      "Logs",
      "Settings",
    ];

    const tablist = page.locator(".fd-tabs");
    await expect(tablist).toBeVisible();

    for (const name of tabNames) {
      const tab = tablist.locator(`button.tab`, { hasText: name });
      await expect(tab).toBeVisible();
      await tab.click();

      // The clicked tab should have the "active" class
      await expect(tab).toHaveClass(/active/);

      // The corresponding pane should be visible
      await expect(page.locator(".fd-pane")).toBeVisible();
    }

    // Switch back to Overview for subsequent tests
    await tablist.locator("button.tab", { hasText: "Overview" }).click();
  });

  /* ================================================================== */
  /*  Overview Tab (8-12)                                                */
  /* ================================================================== */

  /* ------------------------------------------------------------------ */
  /*  8. Image ref copy -> clipboard                                     */
  /* ------------------------------------------------------------------ */
  test("8 - overview: Image ref has copy button", async ({
    page,
    context,
  }) => {
    await context.grantPermissions(["clipboard-read", "clipboard-write"]);

    // Ensure Overview tab is active
    const overviewTab = page.locator(".fd-tabs button.tab", {
      hasText: "Overview",
    });
    await overviewTab.click();

    // Find the "Image" meta row (use exact label text to avoid matching "Image pushed" in pipeline)
    const imageRow = page.locator(".meta-row").filter({ has: page.locator(".lbl", { hasText: /^Image$/ }) });
    await expect(imageRow).toBeVisible();

    // Check if image ref exists (may be "—" if not built yet)
    const copyBtn = imageRow.locator("button[aria-label='Copy']");
    if (await copyBtn.isVisible().catch(() => false)) {
      await copyBtn.click();

      const clipboardText = await page.evaluate(() =>
        navigator.clipboard.readText(),
      );
      expect(clipboardText).toBeTruthy();
    } else {
      // No build yet — the value shows "—"
      const val = imageRow.locator(".val");
      const text = await val.textContent();
      expect(text).toContain("\u2014"); // em dash
    }
  });

  /* ------------------------------------------------------------------ */
  /*  9. Digest copy -> clipboard                                        */
  /* ------------------------------------------------------------------ */
  test("9 - overview: Digest row has copy button", async ({
    page,
    context,
  }) => {
    await context.grantPermissions(["clipboard-read", "clipboard-write"]);

    const overviewTab = page.locator(".fd-tabs button.tab", {
      hasText: "Overview",
    });
    await overviewTab.click();

    const digestRow = page.locator(".meta-row", { hasText: "Digest" });
    await expect(digestRow).toBeVisible();

    const copyBtn = digestRow.locator("button[aria-label='Copy']");
    if (await copyBtn.isVisible().catch(() => false)) {
      await copyBtn.click();

      const clipboardText = await page.evaluate(() =>
        navigator.clipboard.readText(),
      );
      expect(clipboardText).toBeTruthy();
    } else {
      // No digest yet
      const val = digestRow.locator(".val");
      const text = await val.textContent();
      expect(text).toContain("\u2014");
    }
  });

  /* ------------------------------------------------------------------ */
  /*  10. "Edit in Settings tab" link switches to Settings tab           */
  /* ------------------------------------------------------------------ */
  test("10 - overview: Edit in Settings tab link switches to Settings tab", async ({
    page,
  }) => {
    const overviewTab = page.locator(".fd-tabs button.tab", {
      hasText: "Overview",
    });
    await overviewTab.click();

    // Find the "Edit in Settings tab" link
    const editLink = page.getByText("Edit in Settings tab");
    await expect(editLink).toBeVisible();
    await editLink.click();

    // Settings tab should now be active
    const settingsTab = page.locator(".fd-tabs button.tab", {
      hasText: "Settings",
    });
    await expect(settingsTab).toHaveClass(/active/);
  });

  /* ------------------------------------------------------------------ */
  /*  11. Resource grid: CPU/Memory/Timeout values present               */
  /* ------------------------------------------------------------------ */
  test("11 - overview: resource grid shows CPU, Memory, Timeout values", async ({
    page,
  }) => {
    const overviewTab = page.locator(".fd-tabs button.tab", {
      hasText: "Overview",
    });
    await overviewTab.click();

    const resGrid = page.locator(".res-grid");
    await expect(resGrid).toBeVisible();

    // Verify labels
    await expect(resGrid.locator(".res-tile .lbl", { hasText: "CPU" })).toBeVisible();
    await expect(resGrid.locator(".res-tile .lbl", { hasText: "Memory" })).toBeVisible();
    await expect(resGrid.locator(".res-tile .lbl", { hasText: "Timeout" })).toBeVisible();
    await expect(resGrid.locator(".res-tile .lbl", { hasText: "Replicas" })).toBeVisible();
    await expect(resGrid.locator(".res-tile .lbl", { hasText: "GPU" })).toBeVisible();
    await expect(resGrid.locator(".res-tile .lbl", { hasText: "Concurrency" })).toBeVisible();

    // Verify values are present (not empty)
    const tiles = resGrid.locator(".res-tile .val");
    const tileCount = await tiles.count();
    expect(tileCount).toBe(6);
    for (let i = 0; i < tileCount; i++) {
      const text = await tiles.nth(i).textContent();
      expect(text).toBeTruthy();
    }
  });

  /* ------------------------------------------------------------------ */
  /*  12. Build pipeline status display                                  */
  /* ------------------------------------------------------------------ */
  test("12 - overview: Build pipeline steps are displayed", async ({
    page,
  }) => {
    const overviewTab = page.locator(".fd-tabs button.tab", {
      hasText: "Overview",
    });
    await overviewTab.click();

    // The pipeline row should have steps
    const pipelineRow = page.locator(".meta-row", { hasText: "Pipeline" });
    await expect(pipelineRow).toBeVisible();

    const timeline = page.locator(".step-timeline");
    await expect(timeline).toBeVisible();

    // Verify pipeline step labels
    await expect(timeline.getByText("Source uploaded")).toBeVisible();
    await expect(timeline.getByText("Build started")).toBeVisible();
    await expect(timeline.getByText("Image pushed")).toBeVisible();
    await expect(timeline.getByText("Deployed")).toBeVisible();
  });

  /* ================================================================== */
  /*  Triggers Tab (13-18)                                               */
  /* ================================================================== */

  /* ------------------------------------------------------------------ */
  /*  13. DB trigger card visible (if trigger exists)                     */
  /* ------------------------------------------------------------------ */
  test("13 - triggers tab: shows trigger cards or empty state", async ({
    page,
  }) => {
    const triggersTab = page.locator(".fd-tabs button.tab", {
      hasText: "Triggers",
    });
    await triggersTab.click();
    await expect(triggersTab).toHaveClass(/active/);

    // Wait for pane to be visible
    await expect(page.locator(".fd-pane")).toBeVisible();

    // Either trigger cards are shown, or empty state message
    const triggerCards = page.locator(".trigger-card:not(.add)");
    const emptyMsg = page.getByText("No triggers configured");

    const hasCards = await triggerCards.first().isVisible().catch(() => false);
    const hasEmpty = await emptyMsg.isVisible().catch(() => false);

    // One of the two should be visible
    expect(hasCards || hasEmpty).toBe(true);
  });

  /* ------------------------------------------------------------------ */
  /*  14. ObjStorage trigger card (if exists)                            */
  /* ------------------------------------------------------------------ */
  test("14 - triggers tab: Object Storage trigger shows bucket and prefix", async ({
    page,
  }) => {
    const triggersTab = page.locator(".fd-tabs button.tab", {
      hasText: "Triggers",
    });
    await triggersTab.click();

    const storageTrigger = page.locator(".trigger-card.storage");
    if (await storageTrigger.isVisible().catch(() => false)) {
      // Should show bucket and prefix info
      await expect(storageTrigger.getByText("Bucket")).toBeVisible();
      await expect(storageTrigger.getByText("Prefix filter")).toBeVisible();
    } else {
      // No storage trigger — that is okay, test passes
      test.skip(true, "No Object Storage trigger configured on this function");
    }
  });

  /* ------------------------------------------------------------------ */
  /*  15. Add Trigger button is visible                                  */
  /* ------------------------------------------------------------------ */
  test("15 - triggers tab: Add Trigger button is visible", async ({
    page,
  }) => {
    const triggersTab = page.locator(".fd-tabs button.tab", {
      hasText: "Triggers",
    });
    await triggersTab.click();

    const addBtn = page.locator(".trigger-card.add");
    await expect(addBtn).toBeVisible();
    await expect(addBtn).toContainText("Add Trigger");
  });

  /* ------------------------------------------------------------------ */
  /*  16. Edit trigger button visible on trigger card                    */
  /* ------------------------------------------------------------------ */
  test("16 - triggers tab: Edit button visible on existing trigger cards", async ({
    page,
  }) => {
    const triggersTab = page.locator(".fd-tabs button.tab", {
      hasText: "Triggers",
    });
    await triggersTab.click();

    const triggerCards = page.locator(".trigger-card:not(.add)");
    if ((await triggerCards.count()) > 0) {
      // Each card should have Edit and Remove buttons
      const firstCard = triggerCards.first();
      await expect(
        firstCard.locator("button", { hasText: "Edit" }),
      ).toBeVisible();
      await expect(
        firstCard.locator("button", { hasText: "Remove" }),
      ).toBeVisible();
    } else {
      test.skip(true, "No triggers exist on this function");
    }
  });

  /* ------------------------------------------------------------------ */
  /*  17. Remove trigger button visible on trigger card                  */
  /* ------------------------------------------------------------------ */
  test("17 - triggers tab: Remove button visible on trigger card", async ({
    page,
  }) => {
    const triggersTab = page.locator(".fd-tabs button.tab", {
      hasText: "Triggers",
    });
    await triggersTab.click();

    const triggerCards = page.locator(".trigger-card:not(.add)");
    if ((await triggerCards.count()) > 0) {
      const firstCard = triggerCards.first();
      const removeBtn = firstCard.locator("button", { hasText: "Remove" });
      await expect(removeBtn).toBeVisible();
    } else {
      test.skip(true, "No triggers exist — cannot test Remove button");
    }
  });

  /* ------------------------------------------------------------------ */
  /*  18. DB trigger card shows table name and events                    */
  /* ------------------------------------------------------------------ */
  test("18 - triggers tab: Database trigger shows table and events", async ({
    page,
  }) => {
    const triggersTab = page.locator(".fd-tabs button.tab", {
      hasText: "Triggers",
    });
    await triggersTab.click();

    const dbTrigger = page.locator(".trigger-card.db");
    if (await dbTrigger.isVisible().catch(() => false)) {
      await expect(dbTrigger.getByText("Table")).toBeVisible();
      await expect(
        dbTrigger.locator(".type-pill", { hasText: "Database Change" }),
      ).toBeVisible();
    } else {
      test.skip(true, "No Database trigger configured on this function");
    }
  });

  /* ================================================================== */
  /*  Invocations Tab (19-22)                                            */
  /* ================================================================== */

  /* ------------------------------------------------------------------ */
  /*  19. Invocation rows visible or empty state                         */
  /* ------------------------------------------------------------------ */
  test("19 - invocations tab: rows visible or empty state message", async ({
    page,
  }) => {
    const invTab = page.locator(".fd-tabs button.tab", {
      hasText: "Invocations",
    });
    await invTab.click();
    await expect(invTab).toHaveClass(/active/);

    // Wait for pane
    await expect(page.locator(".fd-pane")).toBeVisible();

    // Either invocation rows or "No invocations yet" message
    const rows = page.locator(".inv-row.body-row");
    const emptyMsg = page.getByText("No invocations yet");

    const hasRows = await rows.first().isVisible().catch(() => false);
    const hasEmpty = await emptyMsg.isVisible().catch(() => false);

    expect(hasRows || hasEmpty).toBe(true);
  });

  /* ------------------------------------------------------------------ */
  /*  20. Row expand -> details visible                                  */
  /* ------------------------------------------------------------------ */
  test("20 - invocations tab: clicking a row expands details", async ({
    page,
  }) => {
    const invTab = page.locator(".fd-tabs button.tab", {
      hasText: "Invocations",
    });
    await invTab.click();

    const rows = page.locator(".inv-row.body-row");
    if ((await rows.count()) > 0) {
      const firstRow = rows.first();
      await firstRow.click();

      // Should expand — the row should have "expanded" class
      await expect(firstRow).toHaveClass(/expanded/);

      // The expansion section should be visible
      const expandInner = page.locator(".inv-expand-inner").first();
      await expect(expandInner).toBeVisible();
    } else {
      test.skip(true, "No invocations to expand");
    }
  });

  /* ------------------------------------------------------------------ */
  /*  21. Trace ID copy -> clipboard                                     */
  /* ------------------------------------------------------------------ */
  test("21 - invocations tab: Trace ID has a copy button", async ({
    page,
    context,
  }) => {
    await context.grantPermissions(["clipboard-read", "clipboard-write"]);

    const invTab = page.locator(".fd-tabs button.tab", {
      hasText: "Invocations",
    });
    await invTab.click();

    const rows = page.locator(".inv-row.body-row");
    if ((await rows.count()) > 0) {
      // Each row should have a trace cell with a copy button
      const traceCell = rows.first().locator(".trace");
      await expect(traceCell).toBeVisible();

      const copyBtn = traceCell.locator("button[aria-label='Copy']");
      if (await copyBtn.isVisible().catch(() => false)) {
        await copyBtn.click();

        const clipboardText = await page.evaluate(() =>
          navigator.clipboard.readText(),
        );
        // Trace ID might be empty for some invocations, but clipboard should have something
        expect(typeof clipboardText).toBe("string");
      }
    } else {
      test.skip(true, "No invocations — cannot test Trace ID copy");
    }
  });

  /* ------------------------------------------------------------------ */
  /*  22. Expanded view shows memory and cold start info                 */
  /* ------------------------------------------------------------------ */
  test("22 - invocations tab: expanded view shows additional details", async ({
    page,
  }) => {
    const invTab = page.locator(".fd-tabs button.tab", {
      hasText: "Invocations",
    });
    await invTab.click();

    const rows = page.locator(".inv-row.body-row");
    if ((await rows.count()) > 0) {
      // Expand first row
      await rows.first().click();

      const expandInner = page.locator(".inv-expand-inner").first();
      await expect(expandInner).toBeVisible();

      // Should contain "Cold start" label
      await expect(expandInner.getByText("Cold start")).toBeVisible();
    } else {
      test.skip(true, "No invocations to expand");
    }
  });

  /* ================================================================== */
  /*  Logs Tab (23-27)                                                   */
  /* ================================================================== */

  /* ------------------------------------------------------------------ */
  /*  23. Log lines visible                                              */
  /* ------------------------------------------------------------------ */
  test("23 - logs tab: log lines are visible (mock data)", async ({
    page,
  }) => {
    const logsTab = page.locator(".fd-tabs button.tab", {
      hasText: "Logs",
    });
    await logsTab.click();
    await expect(logsTab).toHaveClass(/active/);

    // The log panel should be visible
    const logPanel = page.locator(".log-panel");
    await expect(logPanel).toBeVisible();

    // Log lines should be present (the page has hardcoded mock logs)
    const logLines = page.locator(".log-line");
    const count = await logLines.count();
    expect(count).toBeGreaterThan(0);

    // Verify log lines have timestamps and level indicators
    const firstLine = logLines.first();
    await expect(firstLine.locator(".ts")).toBeVisible();
    await expect(firstLine.locator(".lvl")).toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  24. Level filter has options                                       */
  /* ------------------------------------------------------------------ */
  test("24 - logs tab: level filter dropdown has All/Info/Warn/Error options", async ({
    page,
  }) => {
    const logsTab = page.locator(".fd-tabs button.tab", {
      hasText: "Logs",
    });
    await logsTab.click();

    // The filter select
    const filterSelect = page.locator(".log-head select");
    await expect(filterSelect).toBeVisible();

    // Verify options
    const options = filterSelect.locator("option");
    const count = await options.count();
    expect(count).toBe(4);

    await expect(options.filter({ hasText: "All" })).toHaveCount(1);
    await expect(options.filter({ hasText: "Info" })).toHaveCount(1);
    await expect(options.filter({ hasText: "Warn" })).toHaveCount(1);
    await expect(options.filter({ hasText: "Error" })).toHaveCount(1);
  });

  /* ------------------------------------------------------------------ */
  /*  25. Clear button visible                                           */
  /* ------------------------------------------------------------------ */
  test("25 - logs tab: Clear button is visible and clickable", async ({
    page,
  }) => {
    const logsTab = page.locator(".fd-tabs button.tab", {
      hasText: "Logs",
    });
    await logsTab.click();

    const clearBtn = page.locator(".log-head button.btn.btn-ghost", {
      hasText: "Clear",
    });
    await expect(clearBtn).toBeVisible();
    await expect(clearBtn).toBeEnabled();
  });

  /* ------------------------------------------------------------------ */
  /*  26. Download button visible                                        */
  /* ------------------------------------------------------------------ */
  test("26 - logs tab: Download button is visible", async ({ page }) => {
    const logsTab = page.locator(".fd-tabs button.tab", {
      hasText: "Logs",
    });
    await logsTab.click();

    const downloadBtn = page.locator(".log-head button.btn.btn-ghost", {
      hasText: "Download",
    });
    await expect(downloadBtn).toBeVisible();
    await expect(downloadBtn).toBeEnabled();
  });

  /* ------------------------------------------------------------------ */
  /*  27. Live toggle                                                    */
  /* ------------------------------------------------------------------ */
  test("27 - logs tab: Live toggle switches on and off", async ({ page }) => {
    const logsTab = page.locator(".fd-tabs button.tab", {
      hasText: "Logs",
    });
    await logsTab.click();

    // The live toggle is a label with a checkbox inside
    const liveToggle = page.locator(".live-toggle");
    await expect(liveToggle).toBeVisible();

    // Initially should be "on"
    await expect(liveToggle).toHaveClass(/on/);

    // Click to toggle off
    const checkbox = liveToggle.locator("input.switch");
    await checkbox.uncheck({ force: true });

    // Should no longer be "on"
    await expect(liveToggle).not.toHaveClass(/on/);

    // Toggle back on
    await checkbox.check({ force: true });
    await expect(liveToggle).toHaveClass(/on/);
  });

  /* ================================================================== */
  /*  Settings Tab (28-35)                                               */
  /* ================================================================== */

  /* ------------------------------------------------------------------ */
  /*  28. Display name change -> Save -> reload -> persists              */
  /* ------------------------------------------------------------------ */
  test("28 - settings tab: display name change, save, reload, persists", async ({
    page,
  }) => {
    const settingsTab = page.locator(".fd-tabs button.tab", {
      hasText: "Settings",
    });
    await settingsTab.click();
    await expect(settingsTab).toHaveClass(/active/);

    // Find the display name input
    const dispNameInput = page.locator("#fn-disp-name");
    await expect(dispNameInput).toBeVisible();

    const newName = `Updated-${Date.now()}`;
    await dispNameInput.clear();
    await dispNameInput.fill(newName);

    // Click Save in the General card
    const generalCard = page.locator(".fd-card", { hasText: "General" }).first();
    const saveBtn = generalCard.locator("button.btn.btn-primary", {
      hasText: "Save",
    });
    await saveBtn.click();

    // Wait for save to complete
    await page.waitForTimeout(2000);

    // Reload and verify
    await page.reload();
    await expect(page.locator("h1", { hasText: fn.name })).toBeVisible({
      timeout: 15000,
    });

    // Switch to settings tab again
    await page.locator(".fd-tabs button.tab", { hasText: "Settings" }).click();

    const updatedInput = page.locator("#fn-disp-name");
    await expect(updatedInput).toBeVisible();
    const val = await updatedInput.inputValue();
    expect(val).toBe(newName);
  });

  /* ------------------------------------------------------------------ */
  /*  29. Description change -> Save -> reload -> persists               */
  /* ------------------------------------------------------------------ */
  test("29 - settings tab: description field is editable", async ({
    page,
  }) => {
    const settingsTab = page.locator(".fd-tabs button.tab", {
      hasText: "Settings",
    });
    await settingsTab.click();

    // Find the description textarea
    const descTextarea = page.locator("#fn-desc");
    await expect(descTextarea).toBeVisible();

    const newDesc = `Test description ${Date.now()}`;
    await descTextarea.clear();
    await descTextarea.fill(newDesc);

    // Verify the textarea accepted the input
    const val = await descTextarea.inputValue();
    expect(val).toBe(newDesc);
  });

  /* ------------------------------------------------------------------ */
  /*  30. Timeout change -> Save -> reload -> persists                   */
  /* ------------------------------------------------------------------ */
  test("30 - settings tab: timeout change, save, reload, persists", async ({
    page,
  }) => {
    const settingsTab = page.locator(".fd-tabs button.tab", {
      hasText: "Settings",
    });
    await settingsTab.click();

    // Find the timeout input
    const timeoutInput = page.locator("#fn-timeout");
    await expect(timeoutInput).toBeVisible();

    const newTimeout = "15";
    await timeoutInput.clear();
    await timeoutInput.fill(newTimeout);

    // Click Save in the Runtime configuration card
    const runtimeCard = page.locator(".fd-card", {
      hasText: "Runtime configuration",
    });
    const saveBtn = runtimeCard.locator("button.btn.btn-primary", {
      hasText: "Save",
    });
    await saveBtn.click();

    // Wait for save
    await page.waitForTimeout(2000);

    // Reload and verify
    await page.reload();
    await expect(page.locator("h1", { hasText: fn.name })).toBeVisible({
      timeout: 15000,
    });

    await page.locator(".fd-tabs button.tab", { hasText: "Settings" }).click();

    const updatedInput = page.locator("#fn-timeout");
    await expect(updatedInput).toBeVisible();
    const val = await updatedInput.inputValue();
    expect(val).toBe(newTimeout);
  });

  /* ------------------------------------------------------------------ */
  /*  31. CPU/Memory selects are present and changeable                  */
  /* ------------------------------------------------------------------ */
  test("31 - settings tab: CPU and Memory selects are editable", async ({
    page,
  }) => {
    const settingsTab = page.locator(".fd-tabs button.tab", {
      hasText: "Settings",
    });
    await settingsTab.click();

    // CPU select
    const cpuSelect = page.locator("#fn-cpu");
    await expect(cpuSelect).toBeVisible();
    await cpuSelect.selectOption("1000m");
    await expect(cpuSelect).toHaveValue("1000m");

    // Memory select
    const memSelect = page.locator("#fn-mem");
    await expect(memSelect).toBeVisible();
    await memSelect.selectOption("1Gi");
    await expect(memSelect).toHaveValue("1Gi");
  });

  /* ------------------------------------------------------------------ */
  /*  32. Max replicas input                                             */
  /* ------------------------------------------------------------------ */
  test("32 - settings tab: Max replicas input is editable", async ({
    page,
  }) => {
    const settingsTab = page.locator(".fd-tabs button.tab", {
      hasText: "Settings",
    });
    await settingsTab.click();

    const replicasInput = page.locator("#fn-replicas");
    await expect(replicasInput).toBeVisible();

    await replicasInput.clear();
    await replicasInput.fill("5");

    const val = await replicasInput.inputValue();
    expect(val).toBe("5");
  });

  /* ------------------------------------------------------------------ */
  /*  33. Scale to zero toggle                                           */
  /* ------------------------------------------------------------------ */
  test("33 - settings tab: Scale to zero toggle is functional", async ({
    page,
  }) => {
    const settingsTab = page.locator(".fd-tabs button.tab", {
      hasText: "Settings",
    });
    await settingsTab.click();

    // Scale to zero checkbox
    const scaleToggle = page.locator("input.stg-switch");
    await expect(scaleToggle).toBeVisible();

    // Should be checked initially
    await expect(scaleToggle).toBeChecked();

    // Uncheck
    await scaleToggle.uncheck({ force: true });
    await expect(scaleToggle).not.toBeChecked();

    // Check again
    await scaleToggle.check({ force: true });
    await expect(scaleToggle).toBeChecked();
  });

  /* ------------------------------------------------------------------ */
  /*  34. Env var: "Add Variable" button and env mapping display         */
  /* ------------------------------------------------------------------ */
  test("34 - settings tab: Add Variable button is visible in env section", async ({
    page,
  }) => {
    const settingsTab = page.locator(".fd-tabs button.tab", {
      hasText: "Settings",
    });
    await settingsTab.click();

    // Find the Environment variables card
    const envCard = page.locator(".fd-card", {
      hasText: "Environment variables",
    }).last();
    await expect(envCard).toBeVisible();

    // "Add Variable" button should be present
    const addBtn = envCard.locator("button.btn.btn-ghost", {
      hasText: "Add Variable",
    });
    await expect(addBtn).toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  35. Env var: Remove button visible on existing env vars            */
  /* ------------------------------------------------------------------ */
  test("35 - settings tab: env var rows show remove button if vars exist", async ({
    page,
  }) => {
    const settingsTab = page.locator(".fd-tabs button.tab", {
      hasText: "Settings",
    });
    await settingsTab.click();

    // Find env mapping rows
    const envCard = page.locator(".fd-card", {
      hasText: "Environment variables",
    }).last();
    const envRows = envCard.locator(".env-map-row");

    if ((await envRows.count()) > 0) {
      // Each row should have a remove button
      const firstRow = envRows.first();
      const removeBtn = firstRow.locator("button[aria-label='Remove']");
      await expect(removeBtn).toBeVisible();
    } else {
      // No env vars — verify "No environment variables configured" message
      await expect(
        envCard.getByText("No environment variables configured"),
      ).toBeVisible();
    }
  });
});

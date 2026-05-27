import { test, expect } from "@playwright/test";
import {
  createTestProject,
  deleteTestProject,
  createTestFunction,
  deleteTestFunction,
  createTestApiKey,
  revokeTestApiKey,
  getProject,
} from "./helpers/api";

/**
 * 05 - Project Overview (Detail Page, Overview Tab)
 *
 * Covers the /projects/{id} page default Overview tab:
 * status card, stats cards, connect panel, clipboard operations,
 * tab navigation via stat cards, and key reveal/mask toggling.
 */

test.describe.serial("Project Overview", () => {
  let project: { id: string; displayName: string; status: string };
  let fn: { id: string; name: string };
  let anonKey: { apiKey: { id: string; keyPrefix: string }; rawKey: string };
  let serviceKey: { apiKey: { id: string; keyPrefix: string }; rawKey: string };

  test.beforeAll(async () => {
    project = await createTestProject("overview", {
      postgresEnabled: true,
      postgrestEnabled: true,
    });
    fn = await createTestFunction(project.id, "overview-fn");
    anonKey = await createTestApiKey(project.id, "e2e-anon", "anon", 90);
    serviceKey = await createTestApiKey(
      project.id,
      "e2e-svc",
      "service_role",
      90,
    );
  });

  test.afterAll(async () => {
    try {
      await revokeTestApiKey(project.id, serviceKey.apiKey.id);
    } catch {}
    try {
      await revokeTestApiKey(project.id, anonKey.apiKey.id);
    } catch {}
    try {
      await deleteTestFunction(project.id, fn.id);
    } catch {}
    try {
      await deleteTestProject(project.id);
    } catch {}
  });

  test.beforeEach(async ({ page }) => {
    await page.goto("/projects/" + project.id);
    // Wait for the status card to be visible (page loaded)
    await expect(page.locator("section.status-card")).toBeVisible({
      timeout: 15000,
    });
  });

  /* ------------------------------------------------------------------ */
  /*  1. Project ID copy                                                 */
  /* ------------------------------------------------------------------ */
  test("1 - copy project ID to clipboard", async ({ page, context }) => {
    await context.grantPermissions(["clipboard-read", "clipboard-write"]);

    // Find the copy button near the id: label
    const idRow = page.locator("section.status-card").locator("text=id:").locator("..");
    const copyBtn = idRow.locator("button[aria-label='Copy']");
    await expect(copyBtn).toBeVisible();
    await copyBtn.click();

    const clipboardText = await page.evaluate(() =>
      navigator.clipboard.readText(),
    );
    expect(clipboardText).toBe(project.id);
  });

  /* ------------------------------------------------------------------ */
  /*  2. Functions card → Functions tab                                  */
  /* ------------------------------------------------------------------ */
  test("2 - click Functions stat card navigates to Functions tab", async ({
    page,
  }) => {
    const fnCard = page
      .locator("section.pd-stats .pd-stat", { hasText: "Functions" });
    await expect(fnCard).toBeVisible();
    await fnCard.click();

    const fnTab = page.getByRole("tab", { name: "Functions" });
    await expect(fnTab).toHaveAttribute("aria-selected", "true");
  });

  /* ------------------------------------------------------------------ */
  /*  3. Storage card → Storage tab                                      */
  /* ------------------------------------------------------------------ */
  test("3 - click Storage stat card navigates to Storage tab", async ({
    page,
  }) => {
    const storageCard = page
      .locator("section.pd-stats .pd-stat", { hasText: "Storage" });
    await expect(storageCard).toBeVisible();
    await storageCard.click();

    const storageTab = page.getByRole("tab", { name: "Storage" });
    await expect(storageTab).toHaveAttribute("aria-selected", "true");
  });

  /* ------------------------------------------------------------------ */
  /*  4. API Keys card → API Keys tab                                    */
  /* ------------------------------------------------------------------ */
  test("4 - click API Keys stat card navigates to API Keys tab", async ({
    page,
  }) => {
    const apiKeysCard = page
      .locator("section.pd-stats .pd-stat", { hasText: "API keys" });
    await expect(apiKeysCard).toBeVisible();
    await apiKeysCard.click();

    const apiKeysTab = page.getByRole("tab", { name: "API Keys" });
    await expect(apiKeysTab).toHaveAttribute("aria-selected", "true");
  });

  /* ------------------------------------------------------------------ */
  /*  5. View all events → Events tab                                    */
  /* ------------------------------------------------------------------ */
  test("5 - click View all events link navigates to Events tab", async ({
    page,
  }) => {
    const viewAllLink = page.getByText("View all events");
    await expect(viewAllLink).toBeVisible();
    await viewAllLink.click();

    const eventsTab = page.getByRole("tab", { name: "Events" });
    await expect(eventsTab).toHaveAttribute("aria-selected", "true");
  });

  /* ------------------------------------------------------------------ */
  /*  6. API endpoint copy                                               */
  /* ------------------------------------------------------------------ */
  test("6 - copy API endpoint to clipboard", async ({ page, context }) => {
    await context.grantPermissions(["clipboard-read", "clipboard-write"]);

    // Find the API endpoint row and its copy button
    const endpointRow = page
      .locator("section.grid-2")
      .locator("text=API endpoint")
      .locator("..");
    const copyBtn = endpointRow.locator("button[aria-label='Copy']");
    await expect(copyBtn).toBeVisible();
    await copyBtn.click();

    const clipboardText = await page.evaluate(() =>
      navigator.clipboard.readText(),
    );
    // The API endpoint should be a URL
    expect(clipboardText).toMatch(/^https?:\/\//);
  });

  /* ------------------------------------------------------------------ */
  /*  7. Anon key: masked → reveal → copy                               */
  /* ------------------------------------------------------------------ */
  test("7 - anon key: masked by default, reveal shows prefix, copy works", async ({
    page,
    context,
  }) => {
    await context.grantPermissions(["clipboard-read", "clipboard-write"]);

    // Find the anon key row in the connect panel
    const connectPanel = page.locator("section.grid-2");
    const anonRow = connectPanel
      .locator(".row")
      .filter({ hasText: "Anon key" })
      .first();

    // Initially the value should be masked (contains bullet characters)
    const valueDisplay = anonRow.locator(".mono, code").first();
    await expect(valueDisplay).toBeVisible();
    const maskedText = await valueDisplay.textContent();
    expect(maskedText).toContain("●");

    // Click the reveal button
    const revealBtn = anonRow.locator("button[aria-label='Reveal key']");
    await expect(revealBtn).toBeVisible();
    await revealBtn.click();

    // After reveal, the key prefix should be visible
    const revealedText = await valueDisplay.textContent();
    expect(revealedText).toContain(anonKey.apiKey.keyPrefix);

    // Click the copy button
    const copyBtn = anonRow.locator("button[aria-label='Copy']");
    await expect(copyBtn).toBeVisible();
    await copyBtn.click();

    const clipboardText = await page.evaluate(() =>
      navigator.clipboard.readText(),
    );
    expect(clipboardText).toContain(anonKey.apiKey.keyPrefix);
  });

  /* ------------------------------------------------------------------ */
  /*  8. Service role key: masked → reveal → copy                        */
  /* ------------------------------------------------------------------ */
  test("8 - service role key: masked by default, reveal shows prefix, copy works", async ({
    page,
    context,
  }) => {
    await context.grantPermissions(["clipboard-read", "clipboard-write"]);

    // Find the service role key row in the connect panel
    const connectPanel = page.locator("section.grid-2");
    const svcRow = connectPanel
      .locator(".row")
      .filter({ hasText: "Service role key" })
      .first();

    // Initially masked
    const valueDisplay = svcRow.locator(".mono, code").first();
    await expect(valueDisplay).toBeVisible();
    const maskedText = await valueDisplay.textContent();
    expect(maskedText).toContain("●");

    // Click reveal
    const revealBtn = svcRow.locator("button[aria-label='Reveal key']");
    await expect(revealBtn).toBeVisible();
    await revealBtn.click();

    // After reveal, the key prefix should be visible
    const revealedText = await valueDisplay.textContent();
    expect(revealedText).toContain(serviceKey.apiKey.keyPrefix);

    // Click copy
    const copyBtn = svcRow.locator("button[aria-label='Copy']");
    await expect(copyBtn).toBeVisible();
    await copyBtn.click();

    const clipboardText = await page.evaluate(() =>
      navigator.clipboard.readText(),
    );
    expect(clipboardText).toContain(serviceKey.apiKey.keyPrefix);
  });

  /* ------------------------------------------------------------------ */
  /*  9. Status badge matches API status                                 */
  /* ------------------------------------------------------------------ */
  test("9 - status badge text matches API project status", async ({
    page,
  }) => {
    // Fetch the latest status from the API
    const apiProject = await getProject(project.id);

    // Find the status badge in the status card
    const badge = page.locator("section.status-card span[class*='badge-lg']");
    await expect(badge).toBeVisible();

    const badgeText = await badge.textContent();
    expect(badgeText).toBeTruthy();

    // Compare case-insensitively: API returns e.g. "ready", badge shows "Ready"
    expect(badgeText!.toLowerCase()).toBe(apiProject.status.toLowerCase());
  });

  /* ------------------------------------------------------------------ */
  /*  10. Stats cards show correct counts                                */
  /* ------------------------------------------------------------------ */
  test("10 - stats cards show correct counts for functions, storage, and API keys", async ({
    page,
  }) => {
    const statsSection = page.locator("section.pd-stats");

    // Functions card should show 1 (we created 1 function)
    const fnCard = statsSection.locator(".pd-stat", { hasText: "Functions" });
    const fnCount = fnCard.locator(".num");
    await expect(fnCount).toHaveText("1");

    // API keys card should show 2 (we created anon + service_role)
    const apiCard = statsSection.locator(".pd-stat", { hasText: "API keys" });
    const apiCount = apiCard.locator(".num");
    await expect(apiCount).toHaveText("2");

    // Storage buckets card - should show 0 (we did not create any buckets)
    const storageCard = statsSection.locator(".pd-stat", {
      hasText: "Storage",
    });
    const storageCount = storageCard.locator(".num");
    await expect(storageCount).toHaveText("0");
  });

  /* ------------------------------------------------------------------ */
  /*  11. Connection Info: API endpoint is present and copyable          */
  /* ------------------------------------------------------------------ */
  test("11 - connection info shows API endpoint with copy button", async ({
    page,
    context,
  }) => {
    await context.grantPermissions(["clipboard-read", "clipboard-write"]);

    const connectPanel = page.locator("section.grid-2");
    await expect(connectPanel).toBeVisible();

    // API endpoint label should be present
    const endpointLabel = connectPanel.locator("span.row-lbl", {
      hasText: "API endpoint",
    });
    await expect(endpointLabel).toBeVisible();

    // Copy button should be near the endpoint
    const endpointRow = endpointLabel.locator("..");
    const copyBtn = endpointRow.locator("button[aria-label='Copy']");
    await expect(copyBtn).toBeVisible();

    await copyBtn.click();
    const clipboardText = await page.evaluate(() =>
      navigator.clipboard.readText(),
    );
    // Should be a valid URL
    expect(clipboardText).toBeTruthy();
    expect(clipboardText).toMatch(/^https?:\/\//);
  });

  /* ------------------------------------------------------------------ */
  /*  12. Key reveal toggle: reveal → visible → hide → masked again      */
  /* ------------------------------------------------------------------ */
  test("12 - key reveal toggle: reveal shows key, click again masks it", async ({
    page,
  }) => {
    // Use the anon key row for this toggle test
    const connectPanel = page.locator("section.grid-2");
    const anonRow = connectPanel
      .locator(".row")
      .filter({ hasText: "Anon key" })
      .first();

    const valueDisplay = anonRow.locator(".mono, code").first();
    await expect(valueDisplay).toBeVisible();

    // Initially masked
    const initialText = await valueDisplay.textContent();
    expect(initialText).toContain("●");

    // Click reveal
    const revealBtn = anonRow.locator("button[aria-label='Reveal key']");
    await revealBtn.click();

    // Key prefix should be visible
    const revealedText = await valueDisplay.textContent();
    expect(revealedText).toContain(anonKey.apiKey.keyPrefix);
    expect(revealedText).not.toContain("●");

    // Click reveal again to mask
    await revealBtn.click();

    // Should be masked again
    const maskedAgain = await valueDisplay.textContent();
    expect(maskedAgain).toContain("●");
    expect(maskedAgain).not.toContain(anonKey.apiKey.keyPrefix);
  });
});

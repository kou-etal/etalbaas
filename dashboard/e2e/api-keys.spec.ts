import { test, expect } from "@playwright/test";

test.describe("API Keys", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/projects");
    await page.waitForTimeout(2000);

    const projectLink = page.locator("[href^='/projects/']").first();
    if (await projectLink.isVisible()) {
      await projectLink.click();
      await page.waitForURL(/\/projects\/[a-zA-Z0-9]+$/);
      // Navigate to API Keys tab
      await page.getByRole("link", { name: "API Keys" }).click();
      await page.waitForURL(/\/api-keys$/);
    } else {
      test.skip();
    }
  });

  test("should display API Keys page", async ({ page }) => {
    await expect(
      page.getByRole("heading", { name: "API Keys", exact: true })
    ).toBeVisible();
  });

  test("should create an API key", async ({ page }) => {
    // Use first() because both header and empty state have "Create Key" button
    await page.getByRole("button", { name: /create key/i }).first().click();

    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    const keyName = `e2e-key-${Date.now()}`;
    await dialog.getByLabel("Key Name").fill(keyName);
    await dialog.getByRole("button", { name: /create key/i }).click();

    // Should show the raw key
    await expect(page.getByText("API Key Created")).toBeVisible({
      timeout: 10000,
    });

    // Raw key should be displayed
    const keyDisplay = page.locator("code").first();
    await expect(keyDisplay).toBeVisible();
    const keyText = await keyDisplay.textContent();
    expect(keyText).toBeTruthy();
    expect(keyText!.length).toBeGreaterThan(10);

    // Close dialog
    await page.getByRole("button", { name: "Done" }).click();

    // Verify key appears in the list with keyPrefix
    await expect(page.getByText(keyName)).toBeVisible({ timeout: 5000 });
  });

  test("should show keyPrefix (not prefix) in table", async ({ page }) => {
    // If there are keys, they should display with masked format
    const table = page.locator("table");
    if (await table.isVisible().catch(() => false)) {
      const rows = page.locator("tbody tr");
      const count = await rows.count();
      if (count > 0) {
        // Key column should show something like "abc12345••••••••"
        const keyCell = rows.first().locator("code").first();
        const text = await keyCell.textContent();
        expect(text).toContain("••••");
      }
    }
  });

  test("should show role column", async ({ page }) => {
    // Role column header only exists when table is visible (keys exist)
    const table = page.locator("table");
    if (await table.isVisible().catch(() => false)) {
      await expect(page.getByText("Role", { exact: true })).toBeVisible();
    }
  });
});

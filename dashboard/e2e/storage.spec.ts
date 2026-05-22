import { test, expect } from "@playwright/test";

test.describe("Storage", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/projects");
    await page.waitForTimeout(2000);

    const projectLink = page.locator("[href^='/projects/']").first();
    if (await projectLink.isVisible()) {
      await projectLink.click();
      await page.waitForURL(/\/projects\/[a-zA-Z0-9]+$/);
      // Navigate to Storage tab
      await page.getByRole("link", { name: "Storage" }).click();
      await page.waitForURL(/\/storage$/);
    } else {
      test.skip();
    }
  });

  test("should display storage page", async ({ page }) => {
    await expect(
      page.getByRole("heading", { name: "Buckets", exact: true })
    ).toBeVisible();
  });

  test("should show accessLevel select in create dialog (not checkbox)", async ({
    page,
  }) => {
    // Use first() because both header and empty state have "Create Bucket" button
    await page.getByRole("button", { name: /create bucket/i }).first().click();

    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Should have "Access Level" select, not a checkbox
    await expect(dialog.getByText("Access Level")).toBeVisible();

    // Should have a combobox/select (not a checkbox)
    await expect(dialog.getByRole("combobox")).toBeVisible();

    // Close without submitting
    await dialog.getByRole("button", { name: "Cancel" }).click();
  });

  test("should display accessLevel badge (not isPublic)", async ({ page }) => {
    // If buckets exist, check their access level display
    const cards = page.locator("[class*='rounded-lg border']");
    const count = await cards.count().catch(() => 0);
    if (count > 0) {
      // Cards should show access level text (public, private, protected)
      const pageText = await page.locator("main").textContent();
      if (pageText) {
        const hasAccessLevel =
          pageText.toLowerCase().includes("public") ||
          pageText.toLowerCase().includes("private") ||
          pageText.toLowerCase().includes("protected");
        // If there are bucket cards, they should show access level info
        expect(hasAccessLevel).toBeTruthy();
      }
    }
  });

  test("should not show objectCount (removed from proto)", async ({
    page,
  }) => {
    // Page should not contain "objects" count text
    const pageText = await page.locator("body").textContent();
    expect(pageText).not.toContain("0 objects");
  });
});

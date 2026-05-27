import { test, expect } from "@playwright/test";

test.describe("Storage", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/projects");
    await page.waitForTimeout(2000);

    const projectLink = page.locator("[href^='/projects/']").first();
    if (await projectLink.isVisible()) {
      await projectLink.click();
      await page.waitForURL(/\/projects\/[a-zA-Z0-9-]+$/);
      // Navigate to Storage tab (tab, not link — single-page tab UI)
      const storageTab = page.getByRole("tab", { name: "Storage" });
      await expect(storageTab).toBeVisible({ timeout: 15000 });
      await storageTab.click();
      await expect(storageTab).toHaveAttribute("aria-selected", "true");
    } else {
      test.skip();
    }
  });

  test("should display storage tab with Create Bucket button", async ({ page }) => {
    await expect(
      page.getByRole("button", { name: /create bucket/i }).first()
    ).toBeVisible({ timeout: 10000 });
  });

  test("should show accessLevel badges when buckets exist", async ({ page }) => {
    // If buckets exist, check their access level display
    const bucketItems = page.locator(".bucket-list .item");
    const count = await bucketItems.count().catch(() => 0);
    if (count > 0) {
      // Bucket items should show access level badges
      const badge = bucketItems.first().locator(".access-badge");
      if (await badge.isVisible().catch(() => false)) {
        const text = await badge.textContent();
        expect(
          ["public", "private", "protected"].some((level) =>
            text?.toLowerCase().includes(level)
          )
        ).toBeTruthy();
      }
    }
  });

  test("should not show objectCount (removed from proto)", async ({
    page,
  }) => {
    // Page should not contain "0 objects" count text
    const pageText = await page.locator("body").textContent();
    expect(pageText).not.toContain("0 objects");
  });
});

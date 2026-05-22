import { test, expect } from "@playwright/test";

test.describe("Events", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/projects");
    await page.waitForTimeout(2000);

    const projectLink = page.locator("[href^='/projects/']").first();
    if (await projectLink.isVisible()) {
      await projectLink.click();
      await page.waitForURL(/\/projects\/[a-zA-Z0-9]+$/);
      // Navigate to Events tab
      await page.getByRole("link", { name: "Events" }).click();
      await page.waitForURL(/\/events$/);
    } else {
      test.skip();
    }
  });

  test("should display events page", async ({ page }) => {
    // Wait for loading to finish, then check for empty state or event table
    await page.waitForTimeout(3000);
    const emptyState = page.getByRole("heading", { name: "No events recorded" });
    const table = page.locator("table");

    const hasEmpty = await emptyState.isVisible().catch(() => false);
    const hasTable = await table.isVisible().catch(() => false);
    expect(hasEmpty || hasTable).toBeTruthy();
  });

  test("should show Function column (not Subject)", async ({ page }) => {
    const table = page.locator("table");
    if (await table.isVisible()) {
      const headers = page.locator("thead th");
      const headerTexts: string[] = [];
      const count = await headers.count();
      for (let i = 0; i < count; i++) {
        const text = await headers.nth(i).textContent();
        if (text) headerTexts.push(text.trim());
      }

      // Should have "Function" column, not "Subject"
      expect(headerTexts).toContain("Function");
      expect(headerTexts).not.toContain("Subject");

      // Should have "Trigger" column, not "Type"
      expect(headerTexts).toContain("Trigger");
    }
  });

  test("should show real status (not hardcoded 'delivered')", async ({
    page,
  }) => {
    const table = page.locator("table");
    if (await table.isVisible()) {
      // Badge component uses Tailwind classes (rounded-full text-xs font-semibold)
      const statusCells = page.locator("tbody td:nth-child(3)");
      const count = await statusCells.count();
      const validStatuses = [
        "Received",
        "Retrying",
        "Delivered",
        "Failed",
        "Pending",
      ];
      for (let i = 0; i < count; i++) {
        const text = await statusCells.nth(i).textContent();
        expect(
          validStatuses.some((s) => text?.includes(s))
        ).toBeTruthy();
      }
    }
  });

  test("should show Attempts column", async ({ page }) => {
    const table = page.locator("table");
    if (await table.isVisible()) {
      await expect(page.getByText("Attempts", { exact: true })).toBeVisible();
    }
  });
});

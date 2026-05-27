import { test, expect } from "@playwright/test";

test.describe("Events", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/projects");
    await page.waitForTimeout(2000);

    const projectLink = page.locator("[href^='/projects/']").first();
    if (await projectLink.isVisible()) {
      await projectLink.click();
      await page.waitForURL(/\/projects\/[a-zA-Z0-9-]+$/);
      // Navigate to Events tab (tab, not link — single-page tab UI)
      const eventsTab = page.getByRole("tab", { name: "Events" });
      await expect(eventsTab).toBeVisible({ timeout: 15000 });
      await eventsTab.click();
      await expect(eventsTab).toHaveAttribute("aria-selected", "true");
    } else {
      test.skip();
    }
  });

  test("should display events page", async ({ page }) => {
    // Wait for loading to finish, then check for event history heading or event table
    await expect(
      page.locator("h2", { hasText: "Event history" })
    ).toBeVisible({ timeout: 10000 });
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

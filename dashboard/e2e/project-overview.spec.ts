import { test, expect } from "@playwright/test";

test.describe("Project Overview", () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to projects and open the first one
    await page.goto("/projects");
    await page.waitForTimeout(2000);

    const projectLink = page.locator("[href^='/projects/']").first();
    if (await projectLink.isVisible()) {
      await projectLink.click();
      await page.waitForURL(/\/projects\/[a-zA-Z0-9-]+$/);
    } else {
      test.skip();
    }
  });

  test("should display project id and description", async ({ page }) => {
    // Project header should show project ID
    await expect(page.locator("section.status-card")).toBeVisible({
      timeout: 15000,
    });
    const idLabel = page.locator("section.status-card .mono", {
      hasText: "id:",
    });
    await expect(idLabel).toBeVisible();
  });

  test("should show real status badge (not hardcoded 'active')", async ({
    page,
  }) => {
    // StatusBadge should show a real status from the API
    const validStatuses = [
      "Pending",
      "Provisioning",
      "Ready",
      "Paused",
      "Failed",
      "Deleted",
    ];

    // Find the status badge in the status card
    const badge = page.locator("section.status-card span[class*='badge-lg']");
    await expect(badge).toBeVisible({ timeout: 10000 });

    const text = await badge.textContent();
    expect(validStatuses.some((s) => text?.includes(s))).toBeTruthy();
    // Should NOT be hardcoded "Active"
    expect(text).not.toBe("Active");
  });

  test("should display stats cards", async ({ page }) => {
    // Stats section should have cards for Functions, Storage, API keys
    const statsSection = page.locator("section.pd-stats");
    await expect(statsSection).toBeVisible({ timeout: 10000 });

    await expect(
      statsSection.locator(".pd-stat", { hasText: "Functions" })
    ).toBeVisible();
    await expect(
      statsSection.locator(".pd-stat", { hasText: "Storage" })
    ).toBeVisible();
    await expect(
      statsSection.locator(".pd-stat", { hasText: "API keys" })
    ).toBeVisible();
  });

  test("should not show Region field", async ({ page }) => {
    // Region was removed from proto, should not appear
    const pageText = await page.locator("section.status-card").textContent();
    expect(pageText?.toLowerCase()).not.toContain("region");
  });

  test("should display Connection Info", async ({ page }) => {
    await expect(page.getByText("Connect to your project")).toBeVisible({
      timeout: 10000,
    });
    await expect(page.getByText("API endpoint")).toBeVisible();
  });
});

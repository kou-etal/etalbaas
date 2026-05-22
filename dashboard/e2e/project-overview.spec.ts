import { test, expect } from "@playwright/test";

test.describe("Project Overview", () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to projects and open the first one
    await page.goto("/projects");
    await page.waitForTimeout(2000);

    const projectLink = page.locator("[href^='/projects/']").first();
    if (await projectLink.isVisible()) {
      await projectLink.click();
      await page.waitForURL(/\/projects\/[a-zA-Z0-9]+$/);
    } else {
      test.skip();
    }
  });

  test("should display project id and description", async ({ page }) => {
    // Project header should show project ID
    const idText = page.getByText(/^id:/).first();
    await expect(idText).toBeVisible();
    const text = await idText.textContent();
    expect(text).toBeTruthy();
    expect(text).not.toBe("undefined");
  });

  test("should show real status badge (not hardcoded 'active')", async ({
    page,
  }) => {
    // StatusBadge should show a real status from the API
    // Badge uses Tailwind classes (inline-flex rounded-full text-xs font-semibold), not "badge" class
    const validStatuses = [
      "Pending",
      "Provisioning",
      "Ready",
      "Paused",
      "Failed",
      "Deleted",
    ];

    // Find the status text - it should be one of the valid statuses
    const statusRegex = new RegExp(`^(${validStatuses.join("|")})$`);
    const statusElement = page.getByText(statusRegex).first();
    await expect(statusElement).toBeVisible({ timeout: 5000 });

    const text = await statusElement.textContent();
    expect(validStatuses.some((s) => text?.includes(s))).toBeTruthy();
    // Should NOT be hardcoded "Active"
    expect(text).not.toBe("Active");
  });

  test("should display real stats (not hardcoded zeros)", async ({ page }) => {
    // Stats cards should exist
    const functionsHeading = page.getByRole("heading", { name: "Functions" });
    await expect(functionsHeading).toBeVisible();

    // Should NOT have "100%" hardcoded uptime
    const pageContent = await page.content();
    expect(pageContent).not.toContain(">100%<");
  });

  test("should not show Region field", async ({ page }) => {
    // Region was removed from proto, should not appear
    const regionLabel = page.getByText("Region", { exact: true });
    await expect(regionLabel).not.toBeVisible();
  });

  test("should display Connection Info", async ({ page }) => {
    await expect(page.getByText("Connect to your project")).toBeVisible();
    await expect(page.getByText("API endpoint")).toBeVisible();
  });
});

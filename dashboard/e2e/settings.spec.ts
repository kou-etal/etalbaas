import { test, expect } from "@playwright/test";

test.describe("Settings", () => {
  test("should display global settings page", async ({ page }) => {
    await page.goto("/settings");
    await expect(
      page.getByRole("heading", { name: "Settings" })
    ).toBeVisible();
  });

  test("should show user profile", async ({ page }) => {
    await page.goto("/settings");
    await expect(page.getByText("Profile")).toBeVisible();
    // Should show user email (not placeholder)
    const emailElement = page.locator('[class*="font-medium"]').first();
    await expect(emailElement).toBeVisible();
  });

  test("should show authentication providers", async ({ page }) => {
    await page.goto("/settings");
    // Click the Authentication tab
    await page.getByRole("button", { name: "Authentication" }).click();
    await expect(page.getByText("Connected accounts")).toBeVisible();
    await expect(page.getByText("GitHub")).toBeVisible();
  });

  test("should show instance info", async ({ page }) => {
    await page.goto("/settings");
    // Click the Instance tab
    await page.getByRole("button", { name: "Instance" }).click();
    await expect(page.getByText("Instance information")).toBeVisible();
    await expect(page.getByText("Version", { exact: true })).toBeVisible();
    await expect(page.getByText("Plan")).toBeVisible();
  });

  test("should show sign out button", async ({ page }) => {
    await page.goto("/settings");
    await expect(
      page.getByRole("button", { name: /sign out/i })
    ).toBeVisible();
  });
});

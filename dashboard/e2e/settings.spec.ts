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
    await expect(
      page.getByRole("heading", { name: "Profile" })
    ).toBeVisible();
    // Should show user email (not placeholder)
    const emailElement = page.locator('#profile-email');
    await expect(emailElement).toBeVisible();
  });

  test("should show authentication providers", async ({ page }) => {
    await page.goto("/settings");
    // Click the Authentication nav link (anchor, not button)
    await page
      .locator('nav.settings-nav a[href="#sec-auth"]')
      .click();
    await expect(page.getByText("Connected accounts")).toBeVisible();
    await expect(page.getByText("GitHub", { exact: true })).toBeVisible();
  });

  test("should show instance info", async ({ page }) => {
    await page.goto("/settings");
    // Click the Instance nav link (anchor, not button)
    await page
      .locator('nav.settings-nav a[href="#sec-instance"]')
      .click();
    await expect(page.getByText("Instance information")).toBeVisible();
    await expect(page.getByText("Version", { exact: true })).toBeVisible();
  });

  test("should show sign out in sidebar dropdown", async ({ page }) => {
    await page.goto("/settings");
    // Sign out is in the sidebar avatar dropdown
    const sidebar = page.locator("aside[aria-label='Primary navigation']");
    const avatarBtn = sidebar.locator("[aria-label='Open account menu']");
    await expect(avatarBtn).toBeVisible({ timeout: 10000 });
    await avatarBtn.click();
    const dropdown = page.locator("[role='menu']");
    await expect(dropdown).toBeVisible({ timeout: 5000 });
    await expect(
      dropdown.getByRole("menuitem", { name: /sign out/i })
    ).toBeVisible();
  });
});

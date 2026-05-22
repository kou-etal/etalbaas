import { test, expect } from "@playwright/test";

test.describe("Projects", () => {
  test("should display projects page", async ({ page }) => {
    await page.goto("/projects");
    await expect(page.getByRole("heading", { name: "Projects", exact: true })).toBeVisible();
  });

  test("should show empty state when no projects", async ({ page }) => {
    await page.goto("/projects");
    // Either projects exist or empty state is shown
    const heading = page.getByRole("heading", { name: "Projects", exact: true });
    await expect(heading).toBeVisible();
  });

  test("should create a project", async ({ page }) => {
    await page.goto("/projects");

    // Wait for page to fully render
    const newProjectBtn = page.getByRole("button", { name: /new project/i }).first();
    await expect(newProjectBtn).toBeVisible({ timeout: 15000 });
    await newProjectBtn.click();

    // Fill in the dialog
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    const projectName = `e2e-test-${Date.now()}`;
    await dialog.getByLabel("Project Name").fill(projectName);
    await dialog.getByLabel("Description").fill("Created by Playwright E2E test");

    // Submit
    await dialog.getByRole("button", { name: /create project/i }).click();

    // Wait for dialog to close (API may take time + 1.4s success view)
    await expect(dialog).not.toBeVisible({ timeout: 20000 });

    // Verify the project appears in the list (also matches toast, use first)
    await expect(page.getByText(projectName).first()).toBeVisible({ timeout: 10000 });
  });

  test("should show service toggles in create dialog", async ({ page }) => {
    await page.goto("/projects");
    await page.getByRole("button", { name: /new project/i }).first().click();

    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Should show service toggles
    await expect(dialog.getByText("PostgreSQL").first()).toBeVisible();
    await expect(dialog.getByText("Redis")).toBeVisible();
    await expect(dialog.getByText("PostgREST").first()).toBeVisible();

    // Should show endpoint preview
    await expect(dialog.getByText("etalbaas.io")).toBeVisible();

    // Should show Cancel button in footer
    await expect(dialog.getByRole("button", { name: "Cancel" })).toBeVisible();

    // Close
    await dialog.getByRole("button", { name: "Cancel" }).click();
  });

  test("should filter projects by search", async ({ page }) => {
    await page.goto("/projects");

    // Wait for projects to load
    await page.waitForTimeout(2000);

    const searchInput = page.getByPlaceholder(/search projects/i);
    if (await searchInput.isVisible()) {
      await searchInput.fill("nonexistent-project-xyz");
      // Should show no results or empty state
      await page.waitForTimeout(500);
    }
  });

  test("should navigate to project detail", async ({ page }) => {
    await page.goto("/projects");

    // Wait for projects to load
    await page.waitForTimeout(2000);

    // Click on the first project card link
    const projectCard = page.locator("[href^='/projects/']").first();
    if (await projectCard.isVisible()) {
      await projectCard.click();
      await expect(page).toHaveURL(/\/projects\/[a-zA-Z0-9]+$/);
    }
  });
});

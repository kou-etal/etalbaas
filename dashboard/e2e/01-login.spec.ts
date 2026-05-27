import { test, expect } from "@playwright/test";

test.describe("Login Page", () => {
  test("GitHub button visible with SVG icon", async ({ browser }) => {
    const ctx = await browser.newContext();
    const page = await ctx.newPage();
    await page.goto("/login");

    const githubBtn = page.getByRole("button", {
      name: /continue with github/i,
    });
    await expect(githubBtn).toBeVisible();
    await expect(githubBtn.locator("svg")).toBeVisible();

    await ctx.close();
  });

  test("Google button visible with SVG icon", async ({ browser }) => {
    const ctx = await browser.newContext();
    const page = await ctx.newPage();
    await page.goto("/login");

    const googleBtn = page.getByRole("button", {
      name: /continue with google/i,
    });
    await expect(googleBtn).toBeVisible();
    await expect(googleBtn.locator("svg")).toBeVisible();

    await ctx.close();
  });

  test("both buttons are enabled", async ({ browser }) => {
    const ctx = await browser.newContext();
    const page = await ctx.newPage();
    await page.goto("/login", { waitUntil: "networkidle" });

    const githubBtn = page.getByRole("button", {
      name: /continue with github/i,
    });
    const googleBtn = page.getByRole("button", {
      name: /continue with google/i,
    });

    await expect(githubBtn).toBeVisible({ timeout: 15_000 });
    await expect(googleBtn).toBeVisible({ timeout: 15_000 });
    await expect(githubBtn).toBeEnabled();
    await expect(googleBtn).toBeEnabled();

    await ctx.close();
  });

  test("authenticated user redirects to /projects", async ({ page }) => {
    await page.goto("/login");
    await expect(page).toHaveURL(/\/projects/);
  });
});

import { test, expect } from "@playwright/test";

test.describe("Sidebar", () => {
  test("sidebar visible with logo, nav items, avatar", async ({ page }) => {
    await page.goto("/projects");

    const sidebar = page.locator("aside[aria-label='Primary navigation']");
    await expect(sidebar).toBeVisible();

    // Logo
    await expect(sidebar.locator("svg").first()).toBeVisible();

    // Nav items
    await expect(
      sidebar.getByRole("button", { name: "Projects" }).or(
        sidebar.getByRole("link", { name: "Projects" })
      )
    ).toBeVisible();
    await expect(
      sidebar.getByRole("button", { name: "Settings" }).or(
        sidebar.getByRole("link", { name: "Settings" })
      )
    ).toBeVisible();

    // Avatar
    await expect(
      sidebar.getByRole("button", { name: /account menu/i })
    ).toBeVisible();
  });

  test("Projects nav has active state on /projects", async ({ page }) => {
    await page.goto("/projects");

    const sidebar = page.locator("aside[aria-label='Primary navigation']");
    const projectsNav = sidebar.getByRole("button", { name: "Projects" }).or(
      sidebar.getByRole("link", { name: "Projects" })
    );
    await expect(projectsNav).toHaveAttribute("aria-current", "page");
  });

  test("Settings nav has active state on /settings", async ({ page }) => {
    await page.goto("/settings");

    const sidebar = page.locator("aside[aria-label='Primary navigation']");
    const settingsNav = sidebar.getByRole("button", { name: "Settings" }).or(
      sidebar.getByRole("link", { name: "Settings" })
    );
    await expect(settingsNav).toHaveAttribute("aria-current", "page");
  });

  test("Projects nav click navigates to /projects", async ({ page }) => {
    await page.goto("/settings");

    const sidebar = page.locator("aside[aria-label='Primary navigation']");
    const projectsNav = sidebar.getByRole("button", { name: "Projects" }).or(
      sidebar.getByRole("link", { name: "Projects" })
    );
    await projectsNav.click();

    await expect(page).toHaveURL(/\/projects/);
  });

  test("Settings nav click navigates to /settings", async ({ page }) => {
    await page.goto("/projects");

    const sidebar = page.locator("aside[aria-label='Primary navigation']");
    const settingsNav = sidebar.getByRole("button", { name: "Settings" }).or(
      sidebar.getByRole("link", { name: "Settings" })
    );
    await settingsNav.click();

    await expect(page).toHaveURL(/\/settings/);
  });

  test("avatar click shows dropdown with email and sign out", async ({
    page,
  }) => {
    await page.goto("/projects");

    // Wait for the avatar button to be fully interactive
    const avatarBtn = page.locator("[aria-label='Open account menu']");
    await expect(avatarBtn).toBeVisible({ timeout: 10000 });
    await avatarBtn.click();

    // Radix DropdownMenu renders content via portal with role="menu"
    const dropdown = page.locator("[role='menu']");
    await expect(dropdown).toBeVisible({ timeout: 15000 });
    await expect(dropdown.getByText(/@/)).toBeVisible();

    // Sign out option
    await expect(
      dropdown.getByRole("menuitem", { name: /sign out/i })
    ).toBeVisible();
  });

  test("sign out redirects to /login", async ({ browser }) => {
    // Use a separate browser context to avoid invalidating auth for other tests
    const ctx = await browser.newContext({
      storageState: ".auth/user.json",
    });
    const page = await ctx.newPage();
    await page.goto("/projects");

    const avatarBtn = page.locator("[aria-label='Open account menu']");
    await expect(avatarBtn).toBeVisible({ timeout: 10000 });
    await avatarBtn.click();

    const dropdown = page.locator("[role='menu']");
    await expect(dropdown).toBeVisible({ timeout: 15000 });
    await dropdown.getByRole("menuitem", { name: /sign out/i }).click();

    await expect(page).toHaveURL(/\/login/, { timeout: 15000 });

    await ctx.close();
  });
});

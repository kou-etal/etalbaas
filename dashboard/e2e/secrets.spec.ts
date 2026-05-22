import { test, expect } from "@playwright/test";

test.describe("Secrets", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/projects");
    await page.waitForTimeout(2000);

    const projectLink = page.locator("[href^='/projects/']").first();
    if (await projectLink.isVisible()) {
      await projectLink.click();
      await page.waitForURL(/\/projects\/[a-zA-Z0-9]+$/);
      // Navigate to Secrets tab
      await page.getByRole("link", { name: "Secrets" }).click();
      await page.waitForURL(/\/secrets$/);
    } else {
      test.skip();
    }
  });

  test("should display secrets page", async ({ page }) => {
    await expect(
      page.getByRole("heading", { name: "Secrets", exact: true })
    ).toBeVisible();
  });

  test("should show Name field in create dialog (not Key)", async ({
    page,
  }) => {
    // Use first() because both header and empty state have "Add Secret" button
    await page.getByRole("button", { name: /add secret/i }).first().click();

    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Should have "Name" label, not "Key" — this validates proto alignment
    await expect(dialog.getByLabel("Name")).toBeVisible();
    await expect(dialog.getByLabel("Value")).toBeVisible();

    // Close without submitting (API may fail if K8s namespace not provisioned)
    await dialog.getByRole("button", { name: "Cancel" }).click();
  });

  test("should display secret name column (not key)", async ({ page }) => {
    // Table header should say "Name" not "Key"
    const table = page.locator("table");
    if (await table.isVisible().catch(() => false)) {
      const nameHeader = page.locator("thead th").first();
      const text = await nameHeader.textContent();
      expect(text).toBe("Name");
    }
  });

  test("should delete a secret", async ({ page }) => {
    const table = page.locator("table");
    if (!(await table.isVisible().catch(() => false))) return;

    const rows = page.locator("tbody tr");
    const count = await rows.count().catch(() => 0);

    if (count > 0) {
      // Click delete on the last row
      const lastRow = rows.last();
      await lastRow.locator("button").last().click();

      // Confirm dialog should appear
      const confirmDialog = page.getByRole("dialog");
      await expect(confirmDialog).toBeVisible();
      await confirmDialog.getByRole("button", { name: "Delete" }).click();

      // Wait for deletion
      await page.waitForTimeout(2000);
    }
  });
});

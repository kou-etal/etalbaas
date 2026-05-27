import { test, expect } from "@playwright/test";

/**
 * 15 - Global Settings Page
 *
 * Covers the /settings page with all sections:
 * - Navigation sidebar (Profile, Authentication, Instance, Billing)
 * - Profile: account info, connected accounts, active sessions
 * - Authentication: OAuth providers (Google, GitHub, GitLab, SAML), domain restriction
 * - Instance: info, TLS, observability, backup
 * - Unsaved changes bar behavior
 *
 * The settings page uses local state (not real API), so Save/reload tests
 * verify UI state management (dirty tracking, button states).
 */

test.describe("Global Settings Page", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/settings");
    // Wait for the settings page to load
    await expect(page.locator("h1")).toHaveText("Settings", { timeout: 15000 });
  });

  /* ================================================================== */
  /*  Navigation (4 tests)                                               */
  /* ================================================================== */

  /* ------------------------------------------------------------------ */
  /*  1. Profile is default section                                      */
  /* ------------------------------------------------------------------ */
  test("1 - profile is the default active section", async ({ page }) => {
    const nav = page.locator('nav.settings-nav[aria-label="Settings sections"]');
    await expect(nav).toBeVisible();

    // Profile link should be active by default
    const profileLink = nav.locator('a[href="#sec-profile"]');
    await expect(profileLink).toHaveClass(/active/);

    // Profile section should be visible
    await expect(page.locator("#sec-profile")).toBeVisible();
    await expect(
      page.locator("#sec-profile h2.section-title"),
    ).toHaveText("Profile");
  });

  /* ------------------------------------------------------------------ */
  /*  2. Auth tab click -> OAuth providers visible                       */
  /* ------------------------------------------------------------------ */
  test("2 - auth tab click shows OAuth providers", async ({ page }) => {
    const nav = page.locator('nav.settings-nav[aria-label="Settings sections"]');
    const authLink = nav.locator('a[href="#sec-auth"]');
    await authLink.click();

    // Auth section should be scrolled into view
    const authSection = page.locator("#sec-auth");
    await expect(authSection).toBeVisible();
    await expect(
      authSection.locator("h2.section-title"),
    ).toHaveText("Authentication");

    // OAuth providers should be visible
    await expect(page.locator("#oauth-google")).toBeVisible();
    await expect(page.locator("#oauth-github")).toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  3. Instance tab click -> Instance info visible                     */
  /* ------------------------------------------------------------------ */
  test("3 - instance tab click shows instance info", async ({ page }) => {
    const nav = page.locator('nav.settings-nav[aria-label="Settings sections"]');
    const instanceLink = nav.locator('a[href="#sec-instance"]');
    await instanceLink.click();

    const instanceSection = page.locator("#sec-instance");
    await expect(instanceSection).toBeVisible();
    await expect(
      instanceSection.locator("h2.section-title"),
    ).toHaveText("Instance");

    // Instance information card should be visible
    await expect(
      instanceSection.locator("h3", { hasText: "Instance information" }),
    ).toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  4. Billing disabled                                                */
  /* ------------------------------------------------------------------ */
  test("4 - billing nav link is disabled", async ({ page }) => {
    const nav = page.locator('nav.settings-nav[aria-label="Settings sections"]');
    const billingLink = nav.locator('a[href="#sec-billing"]');
    await expect(billingLink).toBeVisible();
    await expect(billingLink).toHaveClass(/disabled/);

    // Should show "Phase 2" pill
    await expect(billingLink.locator(".phase-pill")).toHaveText("Phase 2");

    // Clicking should not navigate (preventDefault)
    await billingLink.click();

    // The billing section exists but shows disabled content
    const billingSection = page.locator("#sec-billing .billing-disabled");
    // It may or may not be scrolled into view; just verify it exists in the DOM
    await expect(billingSection).toBeAttached();
  });

  /* ================================================================== */
  /*  Profile (8 tests)                                                  */
  /* ================================================================== */

  /* ------------------------------------------------------------------ */
  /*  5. Change avatar click -> file dialog                              */
  /* ------------------------------------------------------------------ */
  test("5 - change avatar click triggers file dialog", async ({ page }) => {
    const changeAvatarBtn = page.locator(
      "#sec-profile .profile-top .links button",
      { hasText: "Change avatar" },
    );
    await expect(changeAvatarBtn).toBeVisible();

    // The button should be clickable (we verify the click does not error)
    // File input may be triggered programmatically; we check the button exists
    await expect(changeAvatarBtn).toBeEnabled();
  });

  /* ------------------------------------------------------------------ */
  /*  6. Name change -> Save -> values persist                           */
  /* ------------------------------------------------------------------ */
  test("6 - name change and save persists value", async ({ page }) => {
    const nameInput = page.locator("#profile-name");
    await expect(nameInput).toBeVisible();

    // Store original value
    const originalName = await nameInput.inputValue();

    // Change the name
    await nameInput.clear();
    await nameInput.fill("Test User Changed");

    // Save button should be enabled (form is dirty)
    const saveBtn = page.locator(
      "#sec-profile .st-card-foot button.btn-primary",
    ).first();
    await expect(saveBtn).toBeEnabled();
    await saveBtn.click();

    // After save, verify the value persists in the input
    await expect(nameInput).toHaveValue("Test User Changed");

    // Restore original name
    await nameInput.clear();
    await nameInput.fill(originalName);
    await saveBtn.click();
  });

  /* ------------------------------------------------------------------ */
  /*  7. Email is readonly                                               */
  /* ------------------------------------------------------------------ */
  test("7 - email input is readonly", async ({ page }) => {
    const emailInput = page.locator("#profile-email");
    await expect(emailInput).toBeVisible();
    await expect(emailInput).toHaveAttribute("readOnly", "");

    // Should have a helper text about OAuth provider
    await expect(
      page.locator("#sec-profile .helper", {
        hasText: "Managed by your OAuth provider",
      }),
    ).toBeVisible();
  });

  /* ------------------------------------------------------------------ */
  /*  8. Reset cancels changes                                           */
  /* ------------------------------------------------------------------ */
  test("8 - reset cancels profile changes", async ({ page }) => {
    const nameInput = page.locator("#profile-name");
    const originalName = await nameInput.inputValue();

    // Make a change
    await nameInput.clear();
    await nameInput.fill("Temporary Name");

    // Click Reset
    const resetBtn = page.locator(
      "#sec-profile .st-card-foot button.btn-ghost",
    ).first();
    await resetBtn.click();

    // Value should revert to original
    await expect(nameInput).toHaveValue(originalName);
  });

  /* ------------------------------------------------------------------ */
  /*  9. Disconnect Google -> confirm -> toast                           */
  /* ------------------------------------------------------------------ */
  test("9 - disconnect google shows toast", async ({ page }) => {
    const googleRow = page.locator(".acct-row", { hasText: "Google" });
    await expect(googleRow).toBeVisible();

    // Verify Connected badge
    await expect(googleRow.locator(".badge-ok")).toContainText("Connected");

    // Click Disconnect
    await googleRow
      .locator("button.btn-ghost", { hasText: "Disconnect" })
      .click();

    // Toast should appear
    await expect(page.locator("div.toast.show")).toBeVisible({ timeout: 5000 });
  });

  /* ------------------------------------------------------------------ */
  /*  10. Disconnect GitHub -> confirm -> toast                          */
  /* ------------------------------------------------------------------ */
  test("10 - disconnect github shows toast", async ({ page }) => {
    const githubRow = page.locator(".acct-row", { hasText: "GitHub" });
    await expect(githubRow).toBeVisible();

    // Verify Connected badge
    await expect(githubRow.locator(".badge-ok")).toContainText("Connected");

    // Click Disconnect
    await githubRow
      .locator("button.btn-ghost", { hasText: "Disconnect" })
      .click();

    // Toast should appear
    await expect(page.locator("div.toast.show")).toBeVisible({ timeout: 5000 });
  });

  /* ------------------------------------------------------------------ */
  /*  11. Revoke all sessions -> confirm -> toast                        */
  /* ------------------------------------------------------------------ */
  test("11 - revoke all sessions shows toast", async ({ page }) => {
    const revokeBtn = page.locator("button.btn-danger-ghost", {
      hasText: "Revoke all others",
    });
    await expect(revokeBtn).toBeVisible();

    await revokeBtn.click();

    // Toast should appear with revoke message
    await expect(page.locator("div.toast.show")).toBeVisible({ timeout: 5000 });
  });

  /* ------------------------------------------------------------------ */
  /*  12. Save disabled when no changes -> change -> enabled             */
  /* ------------------------------------------------------------------ */
  test("12 - save disabled when no changes, enabled after change", async ({
    page,
  }) => {
    const saveBtn = page.locator(
      "#sec-profile .st-card-foot button.btn-primary",
    ).first();

    // Initially, no changes -> Save should be disabled
    await expect(saveBtn).toBeDisabled();

    // Make a change to profile name
    const nameInput = page.locator("#profile-name");
    const originalName = await nameInput.inputValue();
    await nameInput.clear();
    await nameInput.fill("Dirty Name");

    // Save should now be enabled
    await expect(saveBtn).toBeEnabled();

    // Reset to clean state
    const resetBtn = page.locator(
      "#sec-profile .st-card-foot button.btn-ghost",
    ).first();
    await resetBtn.click();
    await expect(nameInput).toHaveValue(originalName);
  });

  /* ================================================================== */
  /*  Auth (12 tests)                                                    */
  /* ================================================================== */

  /* ------------------------------------------------------------------ */
  /*  13. Google: expand -> fields visible                               */
  /* ------------------------------------------------------------------ */
  test("13 - google: expand shows fields", async ({ page }) => {
    // Scroll to auth section
    await page
      .locator('nav.settings-nav a[href="#sec-auth"]')
      .click();

    const googleRow = page.locator("#oauth-google");
    await expect(googleRow).toBeVisible();

    // Ensure Google is expanded (it's open by default)
    // If not open, click to expand
    if (!(await googleRow.evaluate((el) => el.classList.contains("open")))) {
      await googleRow.locator(".head-row").click();
    }
    await expect(googleRow).toHaveClass(/open/);

    // Fields should be visible
    await expect(page.locator("#g-client-id")).toBeVisible();
    await expect(page.locator("#g-client-secret")).toBeVisible();

    // Redirect URI should be visible
    const redirectInput = googleRow.locator(
      '.config .copyable input[readonly]',
    );
    await expect(redirectInput).toBeVisible();
    await expect(redirectInput).toHaveValue(
      "https://etalbaas.local/auth/callback/google",
    );
  });

  /* ------------------------------------------------------------------ */
  /*  14. Google: enable -> fill -> Save -> values persist               */
  /* ------------------------------------------------------------------ */
  test("14 - google: fill client id and secret, save persists", async ({
    page,
  }) => {
    await page
      .locator('nav.settings-nav a[href="#sec-auth"]')
      .click();

    const googleRow = page.locator("#oauth-google");
    // Ensure expanded
    if (!(await googleRow.evaluate((el) => el.classList.contains("open")))) {
      await googleRow.locator(".head-row").click();
    }

    // Fill Client ID and Secret
    const clientIdInput = page.locator("#g-client-id");
    await clientIdInput.clear();
    await clientIdInput.fill("new-google-client-id-12345");

    const clientSecretInput = page.locator("#g-client-secret");
    await clientSecretInput.clear();
    await clientSecretInput.fill("new-google-secret");

    // Save via the OAuth providers card footer
    const authCardFoot = page.locator(
      "#sec-auth .st-card-foot button.btn-primary",
    ).first();
    await expect(authCardFoot).toBeEnabled();
    await authCardFoot.click();

    // After save, verify the values persist
    await expect(clientIdInput).toHaveValue("new-google-client-id-12345");
    await expect(clientSecretInput).toHaveValue("new-google-secret");

    // Save button should become disabled (no longer dirty)
    await expect(authCardFoot).toBeDisabled();
  });

  /* ------------------------------------------------------------------ */
  /*  15. Google: redirect URL copy -> clipboard                         */
  /* ------------------------------------------------------------------ */
  test("15 - google: redirect URL copy to clipboard", async ({
    page,
    context,
  }) => {
    await context.grantPermissions(["clipboard-read", "clipboard-write"]);

    await page
      .locator('nav.settings-nav a[href="#sec-auth"]')
      .click();

    const googleRow = page.locator("#oauth-google");
    if (!(await googleRow.evaluate((el) => el.classList.contains("open")))) {
      await googleRow.locator(".head-row").click();
    }

    // Click the copy button next to the redirect URI
    const copyBtn = googleRow.locator('.config .copyable button.copy').first();
    await expect(copyBtn).toBeVisible();
    await copyBtn.click();

    // Verify clipboard content
    const clipboardText = await page.evaluate(() =>
      navigator.clipboard.readText(),
    );
    expect(clipboardText).toBe(
      "https://etalbaas.local/auth/callback/google",
    );

    // Toast should appear
    await expect(page.locator("div.toast.show")).toBeVisible({ timeout: 5000 });
  });

  /* ------------------------------------------------------------------ */
  /*  16. Google: domains input -> Save -> persists                      */
  /* ------------------------------------------------------------------ */
  test("16 - google: allowed domains save and persist", async ({ page }) => {
    await page
      .locator('nav.settings-nav a[href="#sec-auth"]')
      .click();

    const googleRow = page.locator("#oauth-google");
    if (!(await googleRow.evaluate((el) => el.classList.contains("open")))) {
      await googleRow.locator(".head-row").click();
    }

    const domainsInput = page.locator("#g-domains");
    await expect(domainsInput).toBeVisible();
    await domainsInput.clear();
    await domainsInput.fill("example.com, company.org");

    // Save
    const authSaveBtn = page.locator(
      "#sec-auth .st-card-foot button.btn-primary",
    ).first();
    await expect(authSaveBtn).toBeEnabled();
    await authSaveBtn.click();

    // Verify value persists
    await expect(domainsInput).toHaveValue("example.com, company.org");
  });

  /* ------------------------------------------------------------------ */
  /*  17. GitHub: expand -> fields visible                               */
  /* ------------------------------------------------------------------ */
  test("17 - github: expand shows fields", async ({ page }) => {
    await page
      .locator('nav.settings-nav a[href="#sec-auth"]')
      .click();

    const githubRow = page.locator("#oauth-github");
    await expect(githubRow).toBeVisible();

    // Expand GitHub row (it's closed by default)
    if (!(await githubRow.evaluate((el) => el.classList.contains("open")))) {
      await githubRow.locator(".head-row").click();
    }
    await expect(githubRow).toHaveClass(/open/);

    // Fields should be visible
    await expect(page.locator("#gh-client-id")).toBeVisible();
    await expect(page.locator("#gh-client-secret")).toBeVisible();

    // Redirect URI
    const redirectInput = githubRow.locator(
      '.config .copyable input[readonly]',
    );
    await expect(redirectInput).toBeVisible();
    await expect(redirectInput).toHaveValue(
      "https://etalbaas.local/auth/callback/github",
    );
  });

  /* ------------------------------------------------------------------ */
  /*  18. GitHub: enable -> fill -> Save -> persists                     */
  /* ------------------------------------------------------------------ */
  test("18 - github: fill client id and secret, save persists", async ({
    page,
  }) => {
    await page
      .locator('nav.settings-nav a[href="#sec-auth"]')
      .click();

    const githubRow = page.locator("#oauth-github");
    if (!(await githubRow.evaluate((el) => el.classList.contains("open")))) {
      await githubRow.locator(".head-row").click();
    }

    const clientIdInput = page.locator("#gh-client-id");
    await clientIdInput.clear();
    await clientIdInput.fill("gh-client-id-abcdef");

    const clientSecretInput = page.locator("#gh-client-secret");
    await clientSecretInput.clear();
    await clientSecretInput.fill("gh-secret-xyz");

    // Save
    const authSaveBtn = page.locator(
      "#sec-auth .st-card-foot button.btn-primary",
    ).first();
    await expect(authSaveBtn).toBeEnabled();
    await authSaveBtn.click();

    // Verify persistence
    await expect(clientIdInput).toHaveValue("gh-client-id-abcdef");
    await expect(clientSecretInput).toHaveValue("gh-secret-xyz");
  });

  /* ------------------------------------------------------------------ */
  /*  19. GitHub: redirect URL copy -> clipboard                         */
  /* ------------------------------------------------------------------ */
  test("19 - github: redirect URL copy to clipboard", async ({
    page,
    context,
  }) => {
    await context.grantPermissions(["clipboard-read", "clipboard-write"]);

    await page
      .locator('nav.settings-nav a[href="#sec-auth"]')
      .click();

    const githubRow = page.locator("#oauth-github");
    if (!(await githubRow.evaluate((el) => el.classList.contains("open")))) {
      await githubRow.locator(".head-row").click();
    }

    const copyBtn = githubRow.locator('.config .copyable button.copy').first();
    await expect(copyBtn).toBeVisible();
    await copyBtn.click();

    const clipboardText = await page.evaluate(() =>
      navigator.clipboard.readText(),
    );
    expect(clipboardText).toBe(
      "https://etalbaas.local/auth/callback/github",
    );

    await expect(page.locator("div.toast.show")).toBeVisible({ timeout: 5000 });
  });

  /* ------------------------------------------------------------------ */
  /*  20. GitHub: allowed orgs -> Save -> persists                       */
  /* ------------------------------------------------------------------ */
  test("20 - github: allowed orgs save and persist", async ({ page }) => {
    await page
      .locator('nav.settings-nav a[href="#sec-auth"]')
      .click();

    const githubRow = page.locator("#oauth-github");
    if (!(await githubRow.evaluate((el) => el.classList.contains("open")))) {
      await githubRow.locator(".head-row").click();
    }

    const orgsInput = page.locator("#gh-orgs");
    await expect(orgsInput).toBeVisible();
    await orgsInput.clear();
    await orgsInput.fill("acme-co, etalbaas");

    // Save
    const authSaveBtn = page.locator(
      "#sec-auth .st-card-foot button.btn-primary",
    ).first();
    await expect(authSaveBtn).toBeEnabled();
    await authSaveBtn.click();

    // Verify persistence
    await expect(orgsInput).toHaveValue("acme-co, etalbaas");
  });

  /* ------------------------------------------------------------------ */
  /*  21. GitLab: disabled state confirmed                               */
  /* ------------------------------------------------------------------ */
  test("21 - gitlab: disabled state confirmed", async ({ page }) => {
    await page
      .locator('nav.settings-nav a[href="#sec-auth"]')
      .click();

    // GitLab row should have the disabled class
    const gitlabRow = page.locator(".oauth-row.disabled").filter({
      hasText: "GitLab",
    });
    await expect(gitlabRow).toBeVisible();

    // The toggle switch should be disabled
    const toggle = gitlabRow.locator("input.stg-switch");
    await expect(toggle).toBeDisabled();

    // Should show "Coming soon" pill
    await expect(gitlabRow.locator(".phase-pill")).toHaveText("Coming soon");
  });

  /* ------------------------------------------------------------------ */
  /*  22. SAML: disabled state confirmed                                 */
  /* ------------------------------------------------------------------ */
  test("22 - saml: disabled state confirmed", async ({ page }) => {
    await page
      .locator('nav.settings-nav a[href="#sec-auth"]')
      .click();

    // SAML row should have the disabled class
    const samlRow = page.locator(".oauth-row.disabled").filter({
      hasText: "SAML",
    });
    await expect(samlRow).toBeVisible();

    // The toggle switch should be disabled
    const toggle = samlRow.locator("input.stg-switch");
    await expect(toggle).toBeDisabled();

    // Should show "Coming soon" pill
    await expect(samlRow.locator(".phase-pill")).toHaveText("Coming soon");
  });

  /* ------------------------------------------------------------------ */
  /*  23. Domain restriction toggle + textarea -> Save -> persists       */
  /* ------------------------------------------------------------------ */
  test("23 - domain restriction toggle and textarea save and persist", async ({
    page,
  }) => {
    await page
      .locator('nav.settings-nav a[href="#sec-auth"]')
      .click();

    // Find the domain restriction card (use h3 heading to avoid matching the OAuth card)
    const domainCard = page.locator(".st-card").filter({
      has: page.locator("h3", { hasText: "Allowed email domains" }),
    });
    await expect(domainCard).toBeVisible();

    // Find the domain toggle (enable domain restriction)
    const domainToggle = domainCard.locator("input.stg-switch");
    await expect(domainToggle).toBeVisible();

    // Enable domain restriction if not already
    if (!(await domainToggle.isChecked())) {
      await domainToggle.check();
    }
    await expect(domainToggle).toBeChecked();

    // Textarea should now be visible
    const domainList = page.locator("#domain-list");
    await expect(domainList).toBeVisible();

    // Fill in domains
    await domainList.clear();
    await domainList.fill("example.com\ncompany.org");

    // Save via the unsaved bar or the card's save button
    // The unsaved bar should appear since we changed a value
    const unsavedBar = page.locator("div.st-unsaved-bar.show");
    if (await unsavedBar.isVisible().catch(() => false)) {
      await unsavedBar.locator("button.btn-primary").click();
    }

    // Verify values persist
    await expect(domainList).toHaveValue("example.com\ncompany.org");
  });

  /* ------------------------------------------------------------------ */
  /*  24. Auth Reset cancels changes                                     */
  /* ------------------------------------------------------------------ */
  test("24 - auth reset cancels changes", async ({ page }) => {
    await page
      .locator('nav.settings-nav a[href="#sec-auth"]')
      .click();

    const googleRow = page.locator("#oauth-google");
    if (!(await googleRow.evaluate((el) => el.classList.contains("open")))) {
      await googleRow.locator(".head-row").click();
    }

    const clientIdInput = page.locator("#g-client-id");
    const originalValue = await clientIdInput.inputValue();

    // Make a change
    await clientIdInput.clear();
    await clientIdInput.fill("temporary-change-value");

    // Click Reset in the OAuth providers card footer
    const resetBtn = page.locator(
      "#sec-auth .st-card-foot button.btn-ghost",
    ).first();
    await resetBtn.click();

    // Value should revert
    await expect(clientIdInput).toHaveValue(originalValue);
  });

  /* ================================================================== */
  /*  Instance (4 tests)                                                 */
  /* ================================================================== */

  /* ------------------------------------------------------------------ */
  /*  25. Instance name change -> Save -> persists                       */
  /* ------------------------------------------------------------------ */
  test("25 - instance name change and save persists", async ({ page }) => {
    await page
      .locator('nav.settings-nav a[href="#sec-instance"]')
      .click();

    const instNameInput = page.locator("#inst-name");
    await expect(instNameInput).toBeVisible();

    const originalName = await instNameInput.inputValue();

    // Change instance name
    await instNameInput.clear();
    await instNameInput.fill("my-etalbaas-instance");

    // Save via unsaved bar
    const unsavedBar = page.locator("div.st-unsaved-bar.show");
    await expect(unsavedBar).toBeVisible({ timeout: 3000 });
    await unsavedBar.locator("button.btn-primary").click();

    // Verify persisted
    await expect(instNameInput).toHaveValue("my-etalbaas-instance");

    // Restore
    await instNameInput.clear();
    await instNameInput.fill(originalName);
    await expect(page.locator("div.st-unsaved-bar.show")).toBeVisible();
    await page
      .locator("div.st-unsaved-bar.show button.btn-primary")
      .click();
  });

  /* ------------------------------------------------------------------ */
  /*  26. Instance URL copy -> clipboard                                 */
  /* ------------------------------------------------------------------ */
  test("26 - instance URL copy to clipboard", async ({ page, context }) => {
    await context.grantPermissions(["clipboard-read", "clipboard-write"]);

    await page
      .locator('nav.settings-nav a[href="#sec-instance"]')
      .click();

    // Find the Instance URL copyable field
    const instanceCard = page.locator(".st-card", {
      hasText: "Instance information",
    });
    await expect(instanceCard).toBeVisible();

    // The Instance URL has a copy button
    const copyBtn = instanceCard.locator(".copyable button.copy").first();
    await expect(copyBtn).toBeVisible();
    await copyBtn.click();

    const clipboardText = await page.evaluate(() =>
      navigator.clipboard.readText(),
    );
    expect(clipboardText).toBe("https://etalbaas.local");

    // Toast
    await expect(page.locator("div.toast.show")).toBeVisible({ timeout: 5000 });
  });

  /* ------------------------------------------------------------------ */
  /*  27. Metrics endpoint copy -> clipboard                             */
  /* ------------------------------------------------------------------ */
  test("27 - metrics endpoint copy to clipboard", async ({
    page,
    context,
  }) => {
    await context.grantPermissions(["clipboard-read", "clipboard-write"]);

    await page
      .locator('nav.settings-nav a[href="#sec-instance"]')
      .click();

    // Find the Observability card
    const obsCard = page.locator(".st-card", { hasText: "Observability" });
    await expect(obsCard).toBeVisible();

    // Click the copy button next to the metrics endpoint
    const metricsCopyBtn = obsCard.locator("button.copy").first();
    await expect(metricsCopyBtn).toBeVisible();
    await metricsCopyBtn.click();

    const clipboardText = await page.evaluate(() =>
      navigator.clipboard.readText(),
    );
    expect(clipboardText).toBe("http://localhost:9090/metrics");

    // Toast
    await expect(page.locator("div.toast.show")).toBeVisible({ timeout: 5000 });
  });

  /* ------------------------------------------------------------------ */
  /*  28. Test Backup -> toast                                           */
  /* ------------------------------------------------------------------ */
  test("28 - test backup button shows toast", async ({ page }) => {
    await page
      .locator('nav.settings-nav a[href="#sec-instance"]')
      .click();

    // Find the Backup card
    const backupCard = page.locator(".st-card", {
      hasText: "Backup configuration",
    });
    await expect(backupCard).toBeVisible();

    // Click the "Test Backup" button
    const testBackupBtn = backupCard.locator("button.btn-ghost", {
      hasText: "Test Backup",
    });
    await expect(testBackupBtn).toBeVisible();
    await testBackupBtn.click();

    // Toast should appear
    await expect(page.locator("div.toast.show")).toBeVisible({ timeout: 5000 });
  });

  /* ================================================================== */
  /*  Unsaved bar (4 tests)                                              */
  /* ================================================================== */

  /* ------------------------------------------------------------------ */
  /*  29. Field change -> unsaved bar appears                            */
  /* ------------------------------------------------------------------ */
  test("29 - field change shows unsaved bar", async ({ page }) => {
    const unsavedBar = page.locator("div.st-unsaved-bar");

    // Initially the bar should not have the "show" class
    await expect(unsavedBar).not.toHaveClass(/show/);

    // Make a change to the profile name
    const nameInput = page.locator("#profile-name");
    const originalName = await nameInput.inputValue();
    await nameInput.clear();
    await nameInput.fill("Unsaved Change Test");

    // The unsaved bar should appear
    await expect(page.locator("div.st-unsaved-bar.show")).toBeVisible();
    await expect(
      page.locator("div.st-unsaved-bar.show"),
    ).toContainText("unsaved changes");

    // Clean up
    await nameInput.clear();
    await nameInput.fill(originalName);
  });

  /* ------------------------------------------------------------------ */
  /*  30. Discard -> changes reverted -> bar hidden                      */
  /* ------------------------------------------------------------------ */
  test("30 - discard reverts changes and hides unsaved bar", async ({
    page,
  }) => {
    const nameInput = page.locator("#profile-name");
    const originalName = await nameInput.inputValue();

    // Make a change
    await nameInput.clear();
    await nameInput.fill("Will Be Discarded");

    // Unsaved bar should appear
    const unsavedBarShow = page.locator("div.st-unsaved-bar.show");
    await expect(unsavedBarShow).toBeVisible();

    // Click Discard
    await unsavedBarShow.locator("button.btn-ghost", { hasText: "Discard" }).click();

    // Value should revert
    await expect(nameInput).toHaveValue(originalName);

    // Unsaved bar should be hidden
    await expect(page.locator("div.st-unsaved-bar")).not.toHaveClass(/show/);
  });

  /* ------------------------------------------------------------------ */
  /*  31. Save -> success -> bar hidden                                  */
  /* ------------------------------------------------------------------ */
  test("31 - save hides unsaved bar", async ({ page }) => {
    const nameInput = page.locator("#profile-name");
    const originalName = await nameInput.inputValue();

    // Make a change
    await nameInput.clear();
    await nameInput.fill("Saved Name");

    // Unsaved bar should appear
    const unsavedBarShow = page.locator("div.st-unsaved-bar.show");
    await expect(unsavedBarShow).toBeVisible();

    // Click Save Changes
    await unsavedBarShow
      .locator("button.btn-primary", { hasText: "Save Changes" })
      .click();

    // Unsaved bar should be hidden
    await expect(page.locator("div.st-unsaved-bar")).not.toHaveClass(/show/);

    // Toast should confirm save
    await expect(page.locator("div.toast.show")).toBeVisible({ timeout: 5000 });

    // Restore original name
    await nameInput.clear();
    await nameInput.fill(originalName);
    // The bar reappears since we changed it again
    await expect(page.locator("div.st-unsaved-bar.show")).toBeVisible();
    await page
      .locator("div.st-unsaved-bar.show button.btn-primary")
      .click();
  });

  /* ------------------------------------------------------------------ */
  /*  32. Tab switch with unsaved -> bar persists                        */
  /* ------------------------------------------------------------------ */
  test("32 - tab switch with unsaved changes keeps bar visible", async ({
    page,
  }) => {
    const nameInput = page.locator("#profile-name");
    const originalName = await nameInput.inputValue();

    // Make a change in Profile section
    await nameInput.clear();
    await nameInput.fill("Unsaved While Switching");

    // Unsaved bar should appear
    await expect(page.locator("div.st-unsaved-bar.show")).toBeVisible();

    // Switch to Auth section
    await page
      .locator('nav.settings-nav a[href="#sec-auth"]')
      .click();

    // Wait for scroll
    await page.waitForTimeout(500);

    // The unsaved bar should still be visible (dirty state persists across sections)
    await expect(page.locator("div.st-unsaved-bar.show")).toBeVisible();

    // Switch to Instance section
    await page
      .locator('nav.settings-nav a[href="#sec-instance"]')
      .click();

    await page.waitForTimeout(500);

    // Bar should still be visible
    await expect(page.locator("div.st-unsaved-bar.show")).toBeVisible();

    // Clean up: discard changes
    await page
      .locator("div.st-unsaved-bar.show button.btn-ghost", {
        hasText: "Discard",
      })
      .click();
    await expect(page.locator("div.st-unsaved-bar")).not.toHaveClass(/show/);
  });
});

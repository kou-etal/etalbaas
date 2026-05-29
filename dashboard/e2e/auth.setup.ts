import { test as setup, expect } from "@playwright/test";
import { cleanupE2eProjects } from "./helpers/api";

// Allow self-signed certificates for API calls to Kind cluster
process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";

// GoTrue URL for API calls (server-side, direct access)
// Use port-forward or cluster-internal URL
const GOTRUE_API_URL =
  process.env.E2E_GOTRUE_API_URL || "http://localhost:9999";
const API_URL =
  process.env.E2E_API_URL || "https://api.local.etalbaas.dev";
const BASE_URL =
  process.env.E2E_BASE_URL || "https://dashboard.local.etalbaas.dev";

const EMAIL = process.env.E2E_USER_EMAIL || "e2e-test@etalbaas.dev";
const PASSWORD = process.env.E2E_USER_PASSWORD || "E2eTestPassword123!";

async function gotrueRequest(path: string, body: Record<string, string>) {
  const res = await fetch(`${GOTRUE_API_URL}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return res;
}

async function apiRequest(
  path: string,
  token: string,
  body: Record<string, unknown> = {}
) {
  const res = await fetch(`${API_URL}/${path}`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(body),
  });
  return res;
}

setup("authenticate", async ({ page, context }) => {
  // 1. Try to sign in with email/password via GoTrue API
  let session: Record<string, unknown>;

  const loginRes = await gotrueRequest("/token?grant_type=password", {
    email: EMAIL,
    password: PASSWORD,
  });

  if (loginRes.ok) {
    session = await loginRes.json();
  } else {
    // 2. User doesn't exist - create via signup
    const signupRes = await gotrueRequest("/signup", {
      email: EMAIL,
      password: PASSWORD,
    });

    if (!signupRes.ok) {
      const errorText = await signupRes.text();
      throw new Error(
        `GoTrue signup failed (${signupRes.status}): ${errorText}`
      );
    }

    // With GOTRUE_MAILER_AUTOCONFIRM=true, user is immediately active
    // Try login again
    const retryRes = await gotrueRequest("/token?grant_type=password", {
      email: EMAIL,
      password: PASSWORD,
    });

    if (!retryRes.ok) {
      const errorText = await retryRes.text();
      throw new Error(
        `GoTrue login failed after signup (${retryRes.status}): ${errorText}`
      );
    }

    session = await retryRes.json();
  }

  const accessToken = session.access_token as string;
  const refreshToken = session.refresh_token as string;
  expect(accessToken).toBeTruthy();

  // 2.5. Ensure tenant record exists in meta DB
  // GetMe returns 404 if tenant doesn't exist; in that case, the tenant
  // must be seeded externally (e.g. via kubectl exec or a setup script).
  // Here we verify it's accessible so tests don't fail silently.
  const meRes = await apiRequest(
    "etalbaas.tenant.v1.TenantService/GetMe",
    accessToken
  );
  if (!meRes.ok) {
    const body = await meRes.text();
    throw new Error(
      `Tenant record missing for E2E user. ` +
        `Seed it with: kubectl exec -n etalbaas deploy/postgres -- ` +
        `psql -U etalbaas -d etalbaas_meta -c ` +
        `"INSERT INTO tenants (id, email, display_name, plan, status) ` +
        `VALUES ('<user-id>', '${EMAIL}', 'E2E Test User', 'free', 'active') ` +
        `ON CONFLICT (id) DO NOTHING;" ` +
        `(API response: ${meRes.status} ${body})`
    );
  }

  // 2.6. Clean up stale e2e projects from previous runs to avoid 429 (project limit)
  await cleanupE2eProjects();

  // 3. Navigate to login page to establish origin
  await page.goto("/login");

  // 4. Inject GoTrue session into localStorage
  // GoTrueClient stores under storageKey "etalbaas-auth"
  // The actual key format used by @supabase/gotrue-js v2 is:
  //   `${storageKey}-code-verifier` for PKCE (if used)
  //   The session itself is stored as the storageKey value
  const sessionData = JSON.stringify({
    access_token: accessToken,
    refresh_token: refreshToken,
    token_type: session.token_type,
    expires_in: session.expires_in,
    expires_at:
      Math.floor(Date.now() / 1000) + (session.expires_in as number),
    user: session.user,
  });

  await page.evaluate(
    ({ key, value }) => {
      localStorage.setItem(key, value);
    },
    { key: "etalbaas-auth", value: sessionData }
  );

  // 5. Set auth cookie for Next.js middleware
  const domain = new URL(BASE_URL).hostname;
  await context.addCookies([
    {
      name: "etalbaas-auth-token",
      value: accessToken,
      domain,
      path: "/",
      httpOnly: false,
      secure: domain !== "localhost",
      sameSite: "Lax",
    },
  ]);

  // 6. Navigate to projects to verify auth works
  await page.goto("/projects");
  await expect(page.getByRole("heading", { name: "Projects", exact: true })).toBeVisible({
    timeout: 15000,
  });

  // 7. Save auth state for reuse by other tests
  await page.context().storageState({ path: ".auth/user.json" });
});

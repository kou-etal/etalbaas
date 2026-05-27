import { NextRequest } from "next/server";

/**
 * PROJECT_SERVICE_URL: Direct HTTP URL to the Project Service within the cluster.
 * Since the Gateway enforces SNI which doesn't work for internal service-to-service
 * calls, we connect directly to the Project Service and pass the tenant identity
 * extracted from the JWT in the x-tenant-id header (same as Gateway would do).
 */
const PROJECT_SERVICE_URL =
  process.env.PROJECT_SERVICE_URL ||
  "http://project.etalbaas.svc.cluster.local:8080";

/**
 * Extract JWT from the request (Authorization header or cookie).
 */
export function getJwtFromRequest(req: NextRequest): string | null {
  const authHeader = req.headers.get("authorization");
  if (authHeader?.startsWith("Bearer ")) {
    return authHeader.slice(7);
  }
  const cookie = req.cookies.get("etalbaas-auth-token");
  return cookie?.value ?? null;
}

/**
 * Decode JWT payload (no verification — the Gateway already verified it).
 */
function decodeJwtPayload(jwt: string): Record<string, unknown> | null {
  try {
    const parts = jwt.split(".");
    if (parts.length !== 3) return null;
    const payload = Buffer.from(parts[1], "base64url").toString("utf-8");
    return JSON.parse(payload);
  } catch {
    return null;
  }
}

/**
 * Verify project ownership by calling the Project Service directly.
 * Extracts the user ID (sub claim) from the JWT and passes it via the
 * X-User-ID header — the same header that Envoy Gateway's claimToHeaders
 * would inject after JWT verification. The downstream interceptor in the
 * Project Service reads this header to enforce tenant isolation.
 * Returns true only if the user owns the project (IDOR prevention, D-206).
 */
export async function verifyProjectOwnership(
  projectId: string,
  jwt: string,
): Promise<boolean> {
  try {
    const claims = decodeJwtPayload(jwt);
    if (!claims?.sub) return false;

    const url = `${PROJECT_SERVICE_URL}/etalbaas.project.v1.ProjectService/GetProject`;
    const response = await fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-User-ID": String(claims.sub),
      },
      body: JSON.stringify({ projectId }),
    });
    return response.ok;
  } catch {
    return false;
  }
}

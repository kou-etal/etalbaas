import { NextRequest, NextResponse } from "next/server";
import {
  getJwtFromRequest,
  verifyProjectOwnership,
} from "@/lib/api/server-auth";

/**
 * Catch-all proxy to PostgREST (DML / Data operations).
 *
 * Usage: /api/postgrest/{table}?project_id=xxx&limit=50&offset=0&order=id
 * Examples:
 *   GET    /api/postgrest/users?project_id=abc&limit=50
 *   POST   /api/postgrest/users?project_id=abc            (Insert)
 *   PATCH  /api/postgrest/users?project_id=abc&id=eq.1    (Update)
 *   DELETE /api/postgrest/users?project_id=abc&id=eq.1    (Delete)
 *   HEAD   /api/postgrest/users?project_id=abc            (Row count)
 *
 * Security:
 * - JWT extracted from cookie/header
 * - Project ownership verified via Project Service (D-206)
 * - PostgREST parameterizes all queries internally (SQL injection safe)
 */
async function proxyToPostgrest(
  req: NextRequest,
  params: { path: string[] },
) {
  const projectId = req.nextUrl.searchParams.get("project_id");
  if (!projectId) {
    return NextResponse.json(
      { error: "project_id query parameter is required" },
      { status: 400 },
    );
  }

  const jwt = getJwtFromRequest(req);
  if (!jwt) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  const isOwner = await verifyProjectOwnership(projectId, jwt);
  if (!isOwner) {
    return NextResponse.json({ error: "Forbidden" }, { status: 403 });
  }

  const path = params.path.join("/");
  const upstreamUrl = new URL(
    `http://postgrest.project-${projectId}.svc.cluster.local:3000/${path}`,
  );

  // Forward query parameters except project_id
  req.nextUrl.searchParams.forEach((value, key) => {
    if (key !== "project_id") {
      upstreamUrl.searchParams.set(key, value);
    }
  });

  try {
    const headers: Record<string, string> = {};
    const ct = req.headers.get("content-type");
    if (ct) headers["Content-Type"] = ct;
    const prefer = req.headers.get("prefer");
    if (prefer) headers["Prefer"] = prefer;

    const body =
      req.method !== "GET" && req.method !== "HEAD"
        ? await req.text()
        : undefined;

    const upstream = await fetch(upstreamUrl.toString(), {
      method: req.method,
      headers,
      body: body || undefined,
    });

    const responseHeaders: Record<string, string> = {
      "Content-Type":
        upstream.headers.get("Content-Type") || "application/json",
    };
    // Forward content-range for row count (HEAD with Prefer: count=exact)
    const contentRange = upstream.headers.get("content-range");
    if (contentRange) {
      responseHeaders["Content-Range"] = contentRange;
    }

    const responseBody = await upstream.text();
    return new NextResponse(responseBody, {
      status: upstream.status,
      headers: responseHeaders,
    });
  } catch (err) {
    return NextResponse.json(
      { error: `Failed to reach PostgREST: ${String(err)}` },
      { status: 502 },
    );
  }
}

export async function GET(
  req: NextRequest,
  { params }: { params: { path: string[] } },
) {
  return proxyToPostgrest(req, params);
}

export async function POST(
  req: NextRequest,
  { params }: { params: { path: string[] } },
) {
  return proxyToPostgrest(req, params);
}

export async function PATCH(
  req: NextRequest,
  { params }: { params: { path: string[] } },
) {
  return proxyToPostgrest(req, params);
}

export async function DELETE(
  req: NextRequest,
  { params }: { params: { path: string[] } },
) {
  return proxyToPostgrest(req, params);
}

export async function HEAD(
  req: NextRequest,
  { params }: { params: { path: string[] } },
) {
  return proxyToPostgrest(req, params);
}

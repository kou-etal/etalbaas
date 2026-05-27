import { NextRequest, NextResponse } from "next/server";
import {
  getJwtFromRequest,
  verifyProjectOwnership,
} from "@/lib/api/server-auth";

/**
 * Catch-all proxy to postgres-meta (DDL / Schema / SQL Editor).
 *
 * Usage: /api/database/{path}?project_id=xxx
 * Examples:
 *   GET  /api/database/tables?project_id=abc
 *   POST /api/database/query?project_id=abc   (SQL Editor)
 *   POST /api/database/tables?project_id=abc  (Create Table)
 *
 * Security:
 * - JWT extracted from cookie/header
 * - Project ownership verified via Project Service (D-206)
 * - postgres-meta is internal-only (no external HTTPRoute)
 */
async function proxyToPostgresMeta(
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
    `http://postgres-meta.project-${projectId}.svc.cluster.local:8080/${path}`,
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

    const body =
      req.method !== "GET" && req.method !== "HEAD"
        ? await req.text()
        : undefined;

    const upstream = await fetch(upstreamUrl.toString(), {
      method: req.method,
      headers,
      body: body || undefined,
    });

    const responseBody = await upstream.text();
    return new NextResponse(responseBody, {
      status: upstream.status,
      headers: {
        "Content-Type":
          upstream.headers.get("Content-Type") || "application/json",
      },
    });
  } catch (err) {
    return NextResponse.json(
      { error: `Failed to reach postgres-meta: ${String(err)}` },
      { status: 502 },
    );
  }
}

export async function GET(
  req: NextRequest,
  { params }: { params: { path: string[] } },
) {
  return proxyToPostgresMeta(req, params);
}

export async function POST(
  req: NextRequest,
  { params }: { params: { path: string[] } },
) {
  return proxyToPostgresMeta(req, params);
}

export async function PATCH(
  req: NextRequest,
  { params }: { params: { path: string[] } },
) {
  return proxyToPostgresMeta(req, params);
}

export async function DELETE(
  req: NextRequest,
  { params }: { params: { path: string[] } },
) {
  return proxyToPostgresMeta(req, params);
}

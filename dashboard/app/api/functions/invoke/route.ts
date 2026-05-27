import { NextRequest, NextResponse } from "next/server";

const API_BASE_URL = (
  process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"
).replace(/\/$/, "");

/**
 * Build the function invocation URL.
 * Management API: https://api.example.com
 * Function URL:   https://{projectId}.api.example.com/functions/{name}/invoke
 *
 * Local dev: http://localhost:8080 → http://{projectId}.localhost:8080/functions/{name}/invoke
 */
function buildInvokeUrl(projectId: string, functionName: string): string {
  try {
    const url = new URL(API_BASE_URL);
    url.hostname = `${projectId}.${url.hostname}`;
    url.pathname = `/functions/${functionName}/invoke`;
    return url.toString();
  } catch {
    return `${API_BASE_URL}/functions/${functionName}/invoke`;
  }
}

/**
 * POST /api/functions/invoke?project_id=xxx&function_name=yyy
 *
 * Server-side proxy for function invocation. The browser cannot call function
 * pods directly because they live on a wildcard subdomain (*.api.{domain})
 * with no CORS headers for the dashboard origin.
 */
export async function POST(req: NextRequest) {
  const projectId = req.nextUrl.searchParams.get("project_id");
  const functionName = req.nextUrl.searchParams.get("function_name");

  if (!projectId || !functionName) {
    return NextResponse.json(
      { error: "project_id and function_name query parameters are required" },
      { status: 400 },
    );
  }

  const invokeUrl = buildInvokeUrl(projectId, functionName);
  const body = await req.text();
  const contentType = req.headers.get("content-type") || "application/json";

  try {
    const upstream = await fetch(invokeUrl, {
      method: "POST",
      headers: { "Content-Type": contentType },
      body: body || undefined,
    });

    const responseBody = await upstream.text();
    return new NextResponse(responseBody, {
      status: upstream.status,
      headers: {
        "Content-Type": upstream.headers.get("Content-Type") || "text/plain",
      },
    });
  } catch (err) {
    return NextResponse.json(
      { error: `Failed to reach function: ${String(err)}` },
      { status: 502 },
    );
  }
}

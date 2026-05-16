import { NextResponse } from "next/server";

export async function GET(request: Request) {
  const { searchParams } = new URL(request.url);
  const origin = new URL(request.url).origin;

  // GoTrue redirects here with access_token in hash fragment
  // The client-side JS will handle the hash extraction
  // This route exists as the redirect target
  const error = searchParams.get("error");
  if (error) {
    return NextResponse.redirect(`${origin}/login?error=${error}`);
  }

  return NextResponse.redirect(`${origin}/projects`);
}

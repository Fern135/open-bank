import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

export function proxy(request: NextRequest) {
  if (request.nextUrl.pathname.startsWith("/api/")) {
    // Next's rewrite drops this header, and without it the API would see every
    // visitor as this one server and rate limit them all together.
    const headers = new Headers(request.headers);
    headers.set("x-forwarded-for", request.headers.get("x-forwarded-for") ?? "");
    return NextResponse.next({ request: { headers } });
  }

  // A convenience redirect only. The Go API checks every request, so a forged
  // cookie gets past this and then sees nothing.
  if (!request.cookies.has("refresh_token")) {
    return NextResponse.redirect(new URL("/sign-in", request.url));
  }
  return NextResponse.next();
}

export const config = {
  matcher: ["/dashboard/:path*", "/api/v1/:path*"],
};

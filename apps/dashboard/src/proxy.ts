import { NextResponse, type NextRequest } from "next/server";
import { auth0 } from "@/lib/auth0";

// Next 16 "proxy" (the renamed middleware convention). auth0.middleware
// mounts /auth/login, /auth/logout, /auth/callback, /auth/profile and
// /auth/access-token, and keeps the session cookie fresh. Everything is
// auth-required except the auth routes themselves and the public status
// pages.
export async function proxy(request: NextRequest) {
  const authResponse = await auth0.middleware(request);

  const { pathname } = request.nextUrl;
  if (pathname.startsWith("/auth/") || pathname.startsWith("/status/")) {
    return authResponse;
  }

  const session = await auth0.getSession(request);
  if (!session) {
    const login = new URL("/auth/login", request.url);
    login.searchParams.set("returnTo", pathname);
    return NextResponse.redirect(login);
  }
  return authResponse;
}

export const config = {
  matcher: [
    "/((?!_next/static|_next/image|favicon.ico|robots.txt|.*\\.(?:png|svg|ico)$).*)",
  ],
};

import { NextResponse, type NextRequest } from "next/server";
import { verifySessionToken, SESSION_COOKIE_NAME } from "@/lib/server/session";

const protectedPaths = ["/todos"];
const authPaths = ["/login", "/signup"];

function matchesPath(pathname: string, paths: string[]) {
  return paths.some((path) => pathname === path || pathname.startsWith(`${path}/`));
}

function logRequest(request: NextRequest, requestId: string, status: number, destination: string) {
  console.info(
    JSON.stringify({
      level: "info",
      service: "todo-app",
      event: "http_request",
      requestId,
      method: request.method,
      path: request.nextUrl.pathname,
      search: request.nextUrl.search,
      destination,
      status,
      userAgent: request.headers.get("user-agent"),
      forwardedFor: request.headers.get("x-forwarded-for"),
    }),
  );
}

export async function middleware(request: NextRequest) {
  const pathname = request.nextUrl.pathname;
  const requestId = request.headers.get("x-request-id") ?? crypto.randomUUID();

  if (pathname.startsWith("/_next") || pathname.startsWith("/api") || pathname.includes(".")) {
    const response = NextResponse.next();
    response.headers.set("x-request-id", requestId);
    logRequest(request, requestId, response.status, "next");
    return response;
  }

  const token = request.cookies.get(SESSION_COOKIE_NAME)?.value;
  const user = token ? await verifySessionToken(token) : null;

  if (matchesPath(pathname, protectedPaths) && !user) {
    const loginUrl = new URL("/login", request.url);
    loginUrl.searchParams.set("next", pathname);
    const response = NextResponse.redirect(loginUrl);
    response.headers.set("x-request-id", requestId);
    logRequest(request, requestId, response.status, "redirect-login");
    return response;
  }

  if ((pathname === "/" || matchesPath(pathname, authPaths)) && user) {
    const response = NextResponse.redirect(new URL("/todos", request.url));
    response.headers.set("x-request-id", requestId);
    logRequest(request, requestId, response.status, "redirect-todos");
    return response;
  }

  const response = NextResponse.next();
  response.headers.set("x-request-id", requestId);
  logRequest(request, requestId, response.status, "next");
  return response;
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico|sitemap.xml|robots.txt).*)"],
};
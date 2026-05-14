import { describe, expect, it, vi } from "vitest";

const verifySessionToken = vi.fn();

vi.mock("@/lib/server/session", () => ({
  SESSION_COOKIE_NAME: "todo_session",
  verifySessionToken,
}));

function createRequest(pathname: string, token?: string) {
  return {
    url: `https://example.com${pathname}`,
    nextUrl: { pathname, search: "" },
    headers: new Headers(),
    cookies: { get: vi.fn().mockReturnValue(token ? { value: token } : undefined) },
  } as never;
}

describe("middleware", () => {
  it("passes through static and api paths", async () => {
    const { middleware } = await import("./middleware");
    const response = await middleware(createRequest("/_next/static/foo.js"));
    expect(response.status).toBe(200);
  });

  it("redirects unauthenticated protected paths", async () => {
    verifySessionToken.mockResolvedValue(null);
    const { middleware } = await import("./middleware");
    const response = await middleware(createRequest("/todos", "bad"));
    expect(response.status).toBe(307);
    expect(response.headers.get("location")).toContain("/login?next=%2Ftodos");
  });

  it("redirects authenticated auth pages and root", async () => {
    verifySessionToken.mockResolvedValue({ id: "u1", name: "Hanako", email: "a@example.com" });
    const { middleware } = await import("./middleware");
    const authResponse = await middleware(createRequest("/login", "ok"));
    const rootResponse = await middleware(createRequest("/", "ok"));
    expect(authResponse.headers.get("location")).toContain("/todos");
    expect(rootResponse.headers.get("location")).toContain("/todos");
  });

  it("allows authenticated protected paths", async () => {
    verifySessionToken.mockResolvedValue({ id: "u1", name: "Hanako", email: "a@example.com" });
    const { middleware } = await import("./middleware");
    const response = await middleware(createRequest("/todos/1", "ok"));
    expect(response.status).toBe(200);
  });
});
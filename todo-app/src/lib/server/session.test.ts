import { beforeEach, describe, expect, it, vi } from "vitest";

const cookiesMock = vi.fn();
const jwtVerifyMock = vi.fn();

vi.mock("next/headers", () => ({
  cookies: cookiesMock,
}));

vi.mock("jose", () => ({
  jwtVerify: jwtVerifyMock,
}));

describe("session helpers", () => {
  beforeEach(() => {
    vi.resetModules();
  });

  it("verifies a valid token", async () => {
    vi.stubEnv("APP_SESSION_SECRET", "secret");
    jwtVerifyMock.mockResolvedValue({
      payload: { sub: "u1", name: "Hanako", email: "hanako@example.com" },
    });

    const { verifySessionToken } = await import("@/lib/server/session");

    await expect(verifySessionToken("token")).resolves.toEqual({
      id: "u1",
      name: "Hanako",
      email: "hanako@example.com",
    });
  });

  it("returns null for invalid claims or verification errors", async () => {
    vi.stubEnv("APP_SESSION_SECRET", "secret");
    const { verifySessionToken } = await import("@/lib/server/session");

    jwtVerifyMock.mockResolvedValueOnce({ payload: { sub: "", name: "a", email: "b" } });
    await expect(verifySessionToken("token")).resolves.toBeNull();

    jwtVerifyMock.mockRejectedValueOnce(new Error("bad token"));
    await expect(verifySessionToken("token")).resolves.toBeNull();
  });

  it("throws when secret is missing", async () => {
    jwtVerifyMock.mockResolvedValue({ payload: { sub: "u1", name: "a", email: "b" } });
    const { verifySessionToken } = await import("@/lib/server/session");

    await expect(verifySessionToken("token")).resolves.toBeNull();
  });

  it("gets the server session user from cookies", async () => {
    vi.stubEnv("APP_SESSION_SECRET", "secret");
    cookiesMock.mockResolvedValue({
      get: vi.fn().mockReturnValue({ value: "jwt" }),
    });
    jwtVerifyMock.mockResolvedValue({
      payload: { sub: "u1", name: "Hanako", email: "hanako@example.com" },
    });

    const { getServerSessionUser } = await import("@/lib/server/session");

    await expect(getServerSessionUser()).resolves.toEqual({
      id: "u1",
      name: "Hanako",
      email: "hanako@example.com",
    });
  });

  it("returns null without a cookie and exposes cookie options", async () => {
    cookiesMock.mockResolvedValue({ get: vi.fn().mockReturnValue(undefined) });
    vi.stubEnv("NODE_ENV", "production");

    const { getServerSessionUser, getSessionCookieOptions } = await import("@/lib/server/session");

    await expect(getServerSessionUser()).resolves.toBeNull();
    expect(getSessionCookieOptions()).toEqual({
      httpOnly: true,
      sameSite: "lax",
      secure: true,
      path: "/",
      maxAge: 60 * 60 * 12,
    });
  });
});
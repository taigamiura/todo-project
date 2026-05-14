import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";

const cookiesMock = vi.fn();
const getServerSessionUser = vi.fn();
const getSessionCookieOptions = vi.fn(() => ({ httpOnly: true, path: "/" }));

vi.mock("next/headers", () => ({ cookies: cookiesMock }));
vi.mock("@/lib/server/session", async () => {
  const actual = await vi.importActual<typeof import("@/lib/server/session")>("@/lib/server/session");

  return {
    ...actual,
    getServerSessionUser,
    getSessionCookieOptions,
  };
});

describe("auth api routes contract", () => {
  beforeAll(() => {
    vi.stubEnv("BFF_BASE_URL", "http://127.0.0.1:8080");
  });

  afterAll(() => {
    vi.unstubAllEnvs();
  });

  it("uses BFF mock for login and signup", async () => {
    const set = vi.fn();
    cookiesMock.mockResolvedValue({ set });

    const login = await import("@/app/api/auth/login/route");
    const loginResponse = await login.POST(
      new Request("http://test", {
        method: "POST",
        body: JSON.stringify({ email: "taro@example.com", password: "password123" }),
      }),
    );

    expect(loginResponse.status).toBe(200);
    await expect(loginResponse.json()).resolves.toEqual({
      user: {
        id: "user-1",
        name: "Taro Todo",
        email: "taro@example.com",
      },
    });
    expect(set).toHaveBeenCalledWith(
      "todo_session",
      "mock-access-token",
      expect.objectContaining({ httpOnly: true, path: "/" }),
    );

    const signup = await import("@/app/api/auth/signup/route");
    const signupResponse = await signup.POST(
      new Request("http://test", {
        method: "POST",
        body: JSON.stringify({ name: "Taro Todo", email: "taro@example.com", password: "password123" }),
      }),
    );

    expect(signupResponse.status).toBe(201);
    await expect(signupResponse.json()).resolves.toEqual({
      user: {
        id: "user-1",
        name: "Taro Todo",
        email: "taro@example.com",
      },
    });
  });

  it("keeps local-only routes under test", async () => {
    const set = vi.fn();
    cookiesMock.mockResolvedValue({ set, get: vi.fn() });

    const logout = await import("@/app/api/auth/logout/route");
    const logoutResponse = await logout.POST();
    expect(logoutResponse.status).toBe(200);

    getServerSessionUser.mockResolvedValueOnce(null).mockResolvedValueOnce({
      id: "user-1",
      name: "Taro Todo",
      email: "taro@example.com",
    });
    const session = await import("@/app/api/auth/session/route");
    expect((await session.GET()).status).toBe(401);
    expect((await session.GET()).status).toBe(200);
  });
});
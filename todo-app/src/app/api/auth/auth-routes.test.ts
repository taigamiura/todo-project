import { describe, expect, it, vi } from "vitest";
import { ApiError } from "@/lib/apiError";

const cookiesMock = vi.fn();
const requestBff = vi.fn();
const getServerSessionUser = vi.fn();
const getSessionCookieOptions = vi.fn(() => ({ httpOnly: true }));

vi.mock("next/headers", () => ({ cookies: cookiesMock }));
vi.mock("@/lib/server/bff", () => ({ requestBff }));
vi.mock("@/lib/server/session", () => ({
  SESSION_COOKIE_NAME: "todo_session",
  getSessionCookieOptions,
  getServerSessionUser,
}));

describe("auth api routes", () => {
  it("handles login route branches", async () => {
    const set = vi.fn();
    cookiesMock.mockResolvedValue({ set });
    requestBff.mockResolvedValue({ accessToken: "jwt", user: { id: "u1", name: "Hanako", email: "a@example.com" } });
    const { POST } = await import("@/app/api/auth/login/route");

    const okResponse = await POST(new Request("http://test", { method: "POST", body: JSON.stringify({ email: "a@example.com", password: "password123" }) }));
    expect(okResponse.status).toBe(200);
    expect(set).toHaveBeenCalled();

    const badResponse = await POST(new Request("http://test", { method: "POST", body: JSON.stringify({ email: "bad", password: "" }) }));
    expect(badResponse.status).toBe(400);

    requestBff.mockRejectedValueOnce(new ApiError({ status: 401, code: "AUTH_INVALID_CREDENTIALS", message: "bad auth" }));
    const failedResponse = await POST(new Request("http://test", { method: "POST", body: JSON.stringify({ email: "a@example.com", password: "password123" }) }));
    expect(failedResponse.status).toBe(401);
  });

  it("handles signup, logout and session routes", async () => {
    const set = vi.fn();
    cookiesMock.mockResolvedValue({ set, get: vi.fn() });
    requestBff.mockResolvedValue({ accessToken: "jwt", user: { id: "u1", name: "Hanako", email: "a@example.com" } });

    const signup = await import("@/app/api/auth/signup/route");
    const signupResponse = await signup.POST(new Request("http://test", { method: "POST", body: JSON.stringify({ name: "Hanako", email: "a@example.com", password: "password123" }) }));
    expect(signupResponse.status).toBe(201);

    requestBff.mockRejectedValueOnce(new ApiError({ status: 409, code: "USER_EMAIL_CONFLICT", message: "duplicate" }));
    const failedSignup = await signup.POST(new Request("http://test", { method: "POST", body: JSON.stringify({ name: "Hanako", email: "a@example.com", password: "password123" }) }));
    expect(failedSignup.status).toBe(409);

    const logout = await import("@/app/api/auth/logout/route");
    const logoutResponse = await logout.POST();
    expect(logoutResponse.status).toBe(200);
    expect(set).toHaveBeenCalledWith("todo_session", "", expect.objectContaining({ maxAge: 0 }));

    getServerSessionUser.mockResolvedValueOnce(null).mockResolvedValueOnce({ id: "u1", name: "Hanako", email: "a@example.com" });
    const session = await import("@/app/api/auth/session/route");
    expect((await session.GET()).status).toBe(401);
    expect((await session.GET()).status).toBe(200);
  });
});
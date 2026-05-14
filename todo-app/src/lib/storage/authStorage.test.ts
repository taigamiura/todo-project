import { describe, expect, it, vi } from "vitest";
import type { User } from "@/features/auth/types/auth";

describe("authStorage", () => {
  it("fetches current user and auth endpoints", async () => {
    const user: User = { id: "u1", name: "Hanako", email: "a@example.com" };
    vi.stubGlobal("fetch", vi.fn()
      .mockResolvedValueOnce({ ok: true, json: vi.fn().mockResolvedValue({ user }) })
      .mockResolvedValueOnce({ ok: true, json: vi.fn().mockResolvedValue({ user }) })
      .mockResolvedValueOnce({ ok: true, json: vi.fn().mockResolvedValue({ user }) })
      .mockResolvedValueOnce({ ok: true, json: vi.fn().mockResolvedValue({ ok: true }) }));

    const authStorage = await import("@/lib/storage/authStorage");
    const listener = vi.fn();
    const unsubscribe = authStorage.subscribeAuthStorage(listener);
    window.dispatchEvent(new Event("todo-app:auth-storage"));
    expect(listener).toHaveBeenCalled();
    unsubscribe();

    await expect(authStorage.getCurrentUser()).resolves.toEqual(user);
    await expect(authStorage.authenticateUser({ email: user.email, password: "password" })).resolves.toEqual(user);
    await expect(authStorage.createUser({ name: user.name, email: user.email, password: "password123" })).resolves.toEqual(user);
    await expect(authStorage.clearCurrentSession()).resolves.toBeUndefined();
  });

  it("returns null or throws on error", async () => {
    vi.stubGlobal("fetch", vi.fn()
      .mockResolvedValueOnce({ ok: false, json: vi.fn().mockResolvedValue({}) })
      .mockResolvedValueOnce({ ok: false, status: 401, json: vi.fn().mockResolvedValue({ error: { code: "AUTH_INVALID_CREDENTIALS", message: "bad auth", status: 401 } }) }));

    const authStorage = await import("@/lib/storage/authStorage");
    await expect(authStorage.getCurrentUser()).resolves.toBeNull();
    await expect(authStorage.authenticateUser({ email: "a", password: "b" })).rejects.toThrow("bad auth");
  });
});
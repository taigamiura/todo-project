import { describe, expect, it, vi } from "vitest";

describe("requestBff", () => {
  it("uses the configured base URL and returns json", async () => {
    vi.stubEnv("BFF_BASE_URL", "http://bff.test");
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: vi.fn().mockResolvedValue({ status: "ok" }),
    }));

    const { requestBff, getBffBaseUrl } = await import("@/lib/server/bff");
    expect(getBffBaseUrl()).toBe("http://bff.test");
    await expect(requestBff("/healthz")).resolves.toEqual({ status: "ok" });
    expect(fetch).toHaveBeenCalledWith("http://bff.test/healthz", expect.objectContaining({
      method: "GET",
    }));
  });

  it("returns null for 204 and falls back to statusText on bad json", async () => {
    vi.stubGlobal("fetch", vi.fn()
      .mockResolvedValueOnce({ ok: true, status: 204, json: vi.fn() })
      .mockResolvedValueOnce({ ok: false, statusText: "Bad Gateway", json: vi.fn().mockRejectedValue(new Error("x")) }));

    const { requestBff } = await import("@/lib/server/bff");
    await expect(requestBff("/v1/todos/{id}", { method: "DELETE", params: { id: "todo-1" } })).resolves.toBeNull();
    await expect(requestBff("/healthz")).rejects.toThrow("Bad Gateway");
  });

  it("surfaces upstream error payloads", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({
      ok: false,
      status: 400,
      json: vi.fn().mockResolvedValue({ error: { code: "BAD_REQUEST", message: "bad request", status: 400 } }),
    }));

    const { requestBff } = await import("@/lib/server/bff");
    await expect(requestBff("/v1/auth/login", { method: "POST", body: { email: "a@example.com", password: "password123" } })).rejects.toThrow("bad request");
  });
});
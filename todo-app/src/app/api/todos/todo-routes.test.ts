import { describe, expect, it, vi } from "vitest";
import { ApiError } from "@/lib/apiError";

const cookiesMock = vi.fn();
const requestBff = vi.fn();

vi.mock("next/headers", () => ({ cookies: cookiesMock }));
vi.mock("@/lib/server/bff", () => ({ requestBff }));
vi.mock("@/lib/server/session", () => ({ SESSION_COOKIE_NAME: "todo_session" }));

const todo = { id: "t1", title: "Todo", description: "desc", completed: false, createdAt: "x", updatedAt: "x" };

describe("todo api routes", () => {
  it("handles collection routes", async () => {
    cookiesMock.mockResolvedValue({ get: vi.fn().mockReturnValue({ value: "jwt" }) });
    requestBff.mockResolvedValueOnce([todo]).mockResolvedValueOnce(todo);
    const routes = await import("@/app/api/todos/route");

    expect((await routes.GET()).status).toBe(200);
    expect((await routes.POST(new Request("http://test", { method: "POST", body: JSON.stringify({ title: "Todo", description: "desc", completed: false }) }))).status).toBe(201);

    expect((await routes.POST(new Request("http://test", { method: "POST", body: JSON.stringify({ title: "", description: "desc", completed: false }) }))).status).toBe(400);
    cookiesMock.mockResolvedValueOnce({ get: vi.fn().mockReturnValue(undefined) });
    expect((await routes.GET()).status).toBe(401);
  });

  it("handles item routes", async () => {
    cookiesMock.mockResolvedValue({ get: vi.fn().mockReturnValue({ value: "jwt" }) });
    requestBff.mockResolvedValueOnce(todo).mockResolvedValueOnce(todo).mockResolvedValueOnce(null);
    const routes = await import("@/app/api/todos/[id]/route");
    const context = { params: Promise.resolve({ id: "t1" }) };

    expect((await routes.GET(new Request("http://test"), context)).status).toBe(200);
    expect((await routes.PATCH(new Request("http://test", { method: "PATCH", body: JSON.stringify({ title: "Todo", description: "desc", completed: false }) }), context)).status).toBe(200);
    expect((await routes.DELETE(new Request("http://test", { method: "DELETE" }), context)).status).toBe(204);

    expect((await routes.PATCH(new Request("http://test", { method: "PATCH", body: JSON.stringify({ title: "", description: "desc", completed: false }) }), context)).status).toBe(400);

    requestBff.mockRejectedValue(new ApiError({ status: 404, code: "TODO_NOT_FOUND", message: "missing" }));
    expect((await routes.GET(new Request("http://test"), context)).status).toBe(404);
    expect((await routes.PATCH(new Request("http://test", { method: "PATCH", body: JSON.stringify({ title: "Todo", description: "desc", completed: false }) }), context)).status).toBe(404);
    expect((await routes.DELETE(new Request("http://test", { method: "DELETE" }), context)).status).toBe(404);
  });
});
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";

const cookiesMock = vi.fn();

vi.mock("next/headers", () => ({ cookies: cookiesMock }));
vi.mock("@/lib/server/session", async () => {
  const actual = await vi.importActual<typeof import("@/lib/server/session")>("@/lib/server/session");

  return {
    ...actual,
    SESSION_COOKIE_NAME: "todo_session",
  };
});

describe("todo api routes contract", () => {
  beforeAll(() => {
    vi.stubEnv("BFF_BASE_URL", "http://127.0.0.1:8080");
  });

  afterAll(() => {
    vi.unstubAllEnvs();
  });

  it("uses BFF mock for collection routes", async () => {
    cookiesMock.mockResolvedValue({ get: vi.fn().mockReturnValue({ value: "contract-jwt" }) });
    const routes = await import("@/app/api/todos/route");

    const listResponse = await routes.GET();
    expect(listResponse.status).toBe(200);
    await expect(listResponse.json()).resolves.toEqual({
      todos: [
        {
          id: "todo-1",
          title: "Buy milk",
          description: "2 liters",
          completed: false,
          createdAt: "2026-05-11T00:00:00Z",
          updatedAt: "2026-05-11T00:00:00Z",
        },
      ],
    });

    const createResponse = await routes.POST(
      new Request("http://test", {
        method: "POST",
        body: JSON.stringify({ title: "Buy milk", description: "2 liters", completed: false }),
      }),
    );
    expect(createResponse.status).toBe(201);
    await expect(createResponse.json()).resolves.toEqual({
      todo: {
        id: "todo-1",
        title: "Buy milk",
        description: "2 liters",
        completed: false,
        createdAt: "2026-05-11T00:00:00Z",
        updatedAt: "2026-05-11T00:00:00Z",
      },
    });
  });

  it("uses BFF mock for item routes", async () => {
    cookiesMock.mockResolvedValue({ get: vi.fn().mockReturnValue({ value: "contract-jwt" }) });
    const routes = await import("@/app/api/todos/[id]/route");
    const context = { params: Promise.resolve({ id: "todo-1" }) };

    const getResponse = await routes.GET(new Request("http://test"), context);
    expect(getResponse.status).toBe(200);
    await expect(getResponse.json()).resolves.toEqual({
      todo: {
        id: "todo-1",
        title: "Buy milk",
        description: "2 liters",
        completed: false,
        createdAt: "2026-05-11T00:00:00Z",
        updatedAt: "2026-05-11T00:00:00Z",
      },
    });

    const patchResponse = await routes.PATCH(
      new Request("http://test", {
        method: "PATCH",
        body: JSON.stringify({ title: "Buy milk", description: "2 liters", completed: false }),
      }),
      context,
    );
    expect(patchResponse.status).toBe(200);
    await expect(patchResponse.json()).resolves.toEqual({
      todo: {
        id: "todo-1",
        title: "Buy milk",
        description: "2 liters",
        completed: false,
        createdAt: "2026-05-11T00:00:00Z",
        updatedAt: "2026-05-11T00:00:00Z",
      },
    });

    const deleteResponse = await routes.DELETE(new Request("http://test", { method: "DELETE" }), context);
    expect(deleteResponse.status).toBe(204);
  });
});
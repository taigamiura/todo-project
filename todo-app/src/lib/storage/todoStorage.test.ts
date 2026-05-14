import { describe, expect, it, vi } from "vitest";

const todo = {
  id: "t1",
  title: "Todo",
  description: "desc",
  completed: false,
  createdAt: "2024-01-01T00:00:00.000Z",
  updatedAt: "2024-01-01T00:00:00.000Z",
};

describe("todoStorage", () => {
  it("fetches and mutates todos through api endpoints", async () => {
    vi.stubGlobal("fetch", vi.fn()
      .mockResolvedValueOnce({ ok: true, json: vi.fn().mockResolvedValue({ todos: [todo] }) })
      .mockResolvedValueOnce({ ok: true, json: vi.fn().mockResolvedValue({ todo }) })
      .mockResolvedValueOnce({ ok: true, json: vi.fn().mockResolvedValue({ todo }) })
      .mockResolvedValueOnce({ ok: true, json: vi.fn().mockResolvedValue({ todo }) })
      .mockResolvedValueOnce({ ok: true, json: vi.fn().mockRejectedValue(new Error("no body")) }));

    const todoStorage = await import("@/lib/storage/todoStorage");
    const listener = vi.fn();
    const unsubscribe = todoStorage.subscribeTodoStorage(listener);
    window.dispatchEvent(new Event("todo-app:todo-storage"));
    expect(listener).toHaveBeenCalled();
    unsubscribe();

    await expect(todoStorage.getTodosByUserId()).resolves.toEqual([todo]);
    await expect(todoStorage.getTodoById("t1")).resolves.toEqual(todo);
    await expect(todoStorage.createTodo({ title: "Todo", description: "desc", completed: false })).resolves.toEqual(todo);
    await expect(todoStorage.updateTodoById("t1", { title: "Todo", description: "desc", completed: true })).resolves.toEqual(todo);
    await expect(todoStorage.deleteTodoById("t1")).resolves.toBeUndefined();
  });

  it("throws upstream errors", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: false, status: 500, json: vi.fn().mockResolvedValue({ error: { code: "TODO_LIST_FAILED", message: "failed", status: 500 } }) }));
    const todoStorage = await import("@/lib/storage/todoStorage");
    await expect(todoStorage.getTodosByUserId()).rejects.toThrow("failed");
  });
});
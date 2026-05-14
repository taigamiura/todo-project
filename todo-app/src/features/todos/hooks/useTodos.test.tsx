import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

const createTodo = vi.fn();
const getTodosByUserId = vi.fn();
const subscribeTodoStorage = vi.fn();
const updateTodoById = vi.fn();

vi.mock("@/lib/storage/todoStorage", () => ({
    createTodo,
    getTodosByUserId,
    subscribeTodoStorage,
    updateTodoById,
}));

describe("useTodos", () => {
    it("loads todos, handles no user, creates and toggles", async () => {
        const todos = [{ id: "t1", title: "Todo", description: "desc", completed: false, createdAt: "x", updatedAt: "x" }];
        let callback: (() => void) | undefined;
        getTodosByUserId.mockResolvedValue(todos);
        subscribeTodoStorage.mockImplementation((cb: () => void) => {
            callback = cb;
            return vi.fn();
        });

        const mod = await import("@/features/todos/hooks/useTodos");
        const noUser = renderHook(() => mod.useTodos());
        expect(noUser.result.current.todos).toEqual([]);
        expect(noUser.result.current.loading).toBe(false);

        const { result } = renderHook(() => mod.useTodos("u1"));
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.todos).toEqual(todos);

        await act(async () => {
            await result.current.createTodo({ title: "Todo", description: "desc", completed: false });
            await result.current.toggleTodo("t1");
            callback?.();
        });

        expect(createTodo).toHaveBeenCalled();
        expect(updateTodoById).toHaveBeenCalledWith("t1", expect.objectContaining({ completed: true }));

        await act(async () => {
            await result.current.toggleTodo("missing");
        });
    });
});
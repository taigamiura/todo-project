import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

const deleteTodoById = vi.fn();
const getTodoById = vi.fn();
const subscribeTodoStorage = vi.fn();
const updateTodoById = vi.fn();

vi.mock("@/lib/storage/todoStorage", () => ({
    deleteTodoById,
    getTodoById,
    subscribeTodoStorage,
    updateTodoById,
}));

describe("useTodoDetail", () => {
    it("loads todo, handles missing ids, updates and deletes", async () => {
        const todo = { id: "t1", title: "Todo", description: "desc", completed: false, createdAt: "x", updatedAt: "x" };
        let callback: (() => void) | undefined;
        getTodoById.mockResolvedValue(todo);
        updateTodoById.mockResolvedValue(todo);
        subscribeTodoStorage.mockImplementation((cb: () => void) => {
            callback = cb;
            return vi.fn();
        });

        const mod = await import("@/features/todos/hooks/useTodoDetail");
        const noIds = renderHook(() => mod.useTodoDetail());
        expect(noIds.result.current.todo).toBeNull();
        expect(noIds.result.current.loading).toBe(false);
        await act(async () => {
            await noIds.result.current.updateTodo({ title: "x", description: "y", completed: false });
            await noIds.result.current.deleteTodo();
        });

        const { result } = renderHook(() => mod.useTodoDetail("u1", "t1"));
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.todo).toEqual(todo);

        await act(async () => {
            await result.current.updateTodo({ title: "Todo", description: "desc", completed: true });
            await result.current.deleteTodo();
            callback?.();
        });

        expect(updateTodoById).toHaveBeenCalled();
        expect(deleteTodoById).toHaveBeenCalledWith("t1");
    });

    it("sets null on fetch error", async () => {
        getTodoById.mockRejectedValue(new Error("missing"));
        subscribeTodoStorage.mockImplementation(() => vi.fn());
        const mod = await import("@/features/todos/hooks/useTodoDetail");
        const { result } = renderHook(() => mod.useTodoDetail("u1", "t1"));
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.todo).toBeNull();
    });
});
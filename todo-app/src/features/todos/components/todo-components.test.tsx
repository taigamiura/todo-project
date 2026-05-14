import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { TodoDetailActions } from "@/features/todos/components/TodoDetailActions";
import { TodoForm } from "@/features/todos/components/TodoForm";
import { TodoList } from "@/features/todos/components/TodoList";
import { TodoListItem } from "@/features/todos/components/TodoListItem";

const todo = { id: "t1", title: "Todo", description: "desc", completed: false, createdAt: "2024-01-01T00:00:00.000Z", updatedAt: "2024-01-01T00:00:00.000Z" };

describe("todo components", () => {
    it("renders list and list item and toggles", async () => {
        const onToggle = vi.fn().mockResolvedValue(undefined);
        const user = userEvent.setup();
        render(
            <div>
                <TodoList todos={[todo]} onToggle={onToggle} />
                <TodoListItem todo={{ ...todo, completed: true, description: "" }} onToggle={onToggle} />
            </div>,
        );

        expect(screen.getAllByText("Todo")).toHaveLength(2);
        expect(screen.getByText("説明は未入力です。")).toBeInTheDocument();
        await user.click(screen.getAllByRole("button", { name: /完了|未完了/ })[0]);
        expect(onToggle).toHaveBeenCalledWith("t1");
    });

    it("submits todo forms and handles errors", async () => {
        const onSubmit = vi.fn().mockResolvedValue(undefined);
        const failingSubmit = vi.fn().mockRejectedValue(new Error("保存失敗"));
        const user = userEvent.setup();

        const { rerender } = render(<TodoForm mode="create" submitLabel="保存" onSubmit={onSubmit} />);
        await user.click(screen.getByRole("button", { name: "保存" }));
        expect(await screen.findByText("タイトルは必須です。")).toBeInTheDocument();

        await user.type(screen.getByPlaceholderText("例: 週次レポートを提出する"), "Todo");
        await user.type(screen.getByPlaceholderText("詳細なメモや手順を入力"), "desc");
        await user.click(screen.getByRole("button", { name: "保存" }));
        await waitFor(() => expect(onSubmit).toHaveBeenCalled());

        rerender(<TodoForm mode="edit" submitLabel="更新" initialValue={{ title: "Edit", description: "desc", completed: true }} onSubmit={failingSubmit} />);
        await user.click(screen.getByRole("button", { name: "更新" }));
        expect(await screen.findByText("保存失敗")).toBeInTheDocument();
    });

    it("keeps edited values across equivalent rerenders in edit mode", async () => {
        const onSubmit = vi.fn().mockResolvedValue(undefined);
        const user = userEvent.setup();

        const { rerender } = render(
            <TodoForm
                mode="edit"
                submitLabel="更新"
                initialValue={{ title: "Original", description: "desc", completed: false }}
                onSubmit={onSubmit}
            />,
        );

        await user.clear(screen.getByLabelText("タイトル"));
        await user.type(screen.getByLabelText("タイトル"), "Updated");
        await user.clear(screen.getByLabelText("説明"));
        await user.type(screen.getByLabelText("説明"), "updated desc");

        rerender(
            <TodoForm
                mode="edit"
                submitLabel="更新"
                initialValue={{ title: "Original", description: "desc", completed: false }}
                onSubmit={onSubmit}
            />,
        );

        await user.click(screen.getByRole("button", { name: "更新" }));

        await waitFor(() =>
            expect(onSubmit).toHaveBeenCalledWith({
                title: "Updated",
                description: "updated desc",
                completed: false,
            }),
        );
    });

    it("deletes through detail actions", async () => {
        const onDelete = vi.fn().mockResolvedValue(undefined);
        const user = userEvent.setup();
        render(<TodoDetailActions onDelete={onDelete} />);
        await user.click(screen.getByRole("button", { name: "Todoを削除" }));
        await waitFor(() => expect(onDelete).toHaveBeenCalled());
        expect(screen.getByRole("button", { name: "削除中..." })).toBeDisabled();
    });
});
import { Todo } from "@/features/todos/types/todo";
import { TodoListItem } from "@/features/todos/components/TodoListItem";

type TodoListProps = {
    todos: Todo[];
    onToggle: (id: string) => Promise<void>;
};

export function TodoList({ todos, onToggle }: TodoListProps) {
    return (
        <div className="space-y-4">
            {todos.map((todo) => (
                <TodoListItem key={todo.id} todo={todo} onToggle={onToggle} />
            ))}
        </div>
    );
}
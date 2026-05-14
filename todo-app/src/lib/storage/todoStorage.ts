import { Todo, TodoDraft } from "@/features/todos/types/todo";
import { parseApiErrorPayload } from "@/lib/apiError";

const TODO_STORAGE_EVENT = "todo-app:todo-storage";

async function requestTodo<T>(path: string, options: RequestInit = {}) {
  const method = options.method ?? "GET";
  const response = await fetch(path, {
    ...options,
    headers: {
      ...(options.body ? { "Content-Type": "application/json" } : {}),
      ...(options.headers ?? {}),
    },
    cache: "no-store",
  });

  const payload = (await response.json().catch(() => null)) as
    | { error?: unknown; todo?: Todo; todos?: Todo[] }
    | null;

  if (!response.ok) {
    throw parseApiErrorPayload(payload, {
      status: response.status,
      code: "TODO_REQUEST_FAILED",
      message: "Todo の操作に失敗しました。",
    });
  }

  if (typeof window !== "undefined" && method !== "GET") {
    window.dispatchEvent(new Event(TODO_STORAGE_EVENT));
  }

  return payload as T;
}

export function subscribeTodoStorage(onStoreChange: () => void) {
  if (typeof window === "undefined") {
    return () => undefined;
  }

  window.addEventListener(TODO_STORAGE_EVENT, onStoreChange);

  return () => {
    window.removeEventListener(TODO_STORAGE_EVENT, onStoreChange);
  };
}

export async function getTodosByUserId() {
  const payload = await requestTodo<{ todos: Todo[] }>("/api/todos", {
    method: "GET",
  });

  return payload.todos;
}

export async function getTodoById(todoId: string) {
  const payload = await requestTodo<{ todo: Todo }>(`/api/todos/${todoId}`, {
    method: "GET",
  });

  return payload.todo;
}

export async function createTodo(input: TodoDraft) {
  const payload = await requestTodo<{ todo: Todo }>("/api/todos", {
    method: "POST",
    body: JSON.stringify(input),
  });

  return payload.todo;
}

export async function updateTodoById(todoId: string, input: TodoDraft) {
  const payload = await requestTodo<{ todo: Todo }>(`/api/todos/${todoId}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });

  return payload.todo;
}

export async function deleteTodoById(todoId: string) {
  await requestTodo<null>(`/api/todos/${todoId}`, {
    method: "DELETE",
  });
}
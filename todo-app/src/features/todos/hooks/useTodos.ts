"use client";

import { useEffect, useState } from "react";
import { Todo, TodoDraft } from "@/features/todos/types/todo";
import { createTodo, getTodosByUserId, subscribeTodoStorage, updateTodoById } from "@/lib/storage/todoStorage";

type UseTodosResult = {
  todos: Todo[];
  loading: boolean;
  createTodo: (input: TodoDraft) => Promise<void>;
  toggleTodo: (id: string) => Promise<void>;
};

export function useTodos(userId?: string): UseTodosResult {
  const [todos, setTodos] = useState<Todo[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!userId) {
      return;
    }

    let isMounted = true;

    const load = async () => {
      try {
        const nextTodos = await getTodosByUserId();
        if (isMounted) {
          setTodos(nextTodos);
        }
      } finally {
        if (isMounted) {
          setLoading(false);
        }
      }
    };

    void load();

    const unsubscribe = subscribeTodoStorage(() => {
      setLoading(true);
      void load();
    });

    return () => {
      isMounted = false;
      unsubscribe();
    };
  }, [userId]);

  return {
    todos: userId ? todos : [],
    loading: userId ? loading : false,
    createTodo: async (input) => {
      if (!userId) {
        return;
      }

      await createTodo(input);
    },
    toggleTodo: async (id) => {
      if (!userId) {
        return;
      }

      const target = todos.find((todo) => todo.id === id);

      if (!target) {
        return;
      }

      await updateTodoById(id, {
        title: target.title,
        description: target.description,
        completed: !target.completed,
      });
    },
  };
}
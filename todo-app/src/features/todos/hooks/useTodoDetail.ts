"use client";

import { useEffect, useState } from "react";
import { Todo, TodoDraft } from "@/features/todos/types/todo";
import { deleteTodoById, getTodoById, subscribeTodoStorage, updateTodoById } from "@/lib/storage/todoStorage";

type UseTodoDetailResult = {
  todo: Todo | null;
  loading: boolean;
  updateTodo: (input: TodoDraft) => Promise<void>;
  deleteTodo: () => Promise<void>;
};

export function useTodoDetail(userId?: string, todoId?: string): UseTodoDetailResult {
  const [todo, setTodo] = useState<Todo | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!userId || !todoId) {
      return;
    }

    let isMounted = true;

    const load = async () => {
      try {
        const nextTodo = await getTodoById(todoId);
        if (isMounted) {
          setTodo(nextTodo);
        }
      } catch {
        if (isMounted) {
          setTodo(null);
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
  }, [todoId, userId]);

  return {
    todo: userId && todoId ? todo : null,
    loading: userId && todoId ? loading : false,
    updateTodo: async (input) => {
      if (!userId || !todoId) {
        return;
      }

      const updated = await updateTodoById(todoId, input);
      setTodo(updated);
    },
    deleteTodo: async () => {
      if (!userId || !todoId) {
        return;
      }

      await deleteTodoById(todoId);
      setTodo(null);
    },
  };
}
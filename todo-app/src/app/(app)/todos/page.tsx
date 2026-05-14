"use client";

import { useMemo, useState } from "react";
import { Card } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { PageHeader } from "@/components/ui/PageHeader";
import { LogoutButton } from "@/features/auth/components/LogoutButton";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { TodoForm } from "@/features/todos/components/TodoForm";
import { TodoList } from "@/features/todos/components/TodoList";
import { useTodos } from "@/features/todos/hooks/useTodos";

export default function TodosPage() {
    const { user, loading: authLoading } = useAuth();
    const [isCreating, setIsCreating] = useState(false);
    const { todos, loading, createTodo, toggleTodo } = useTodos(user?.id);

    const completedCount = useMemo(() => todos.filter((todo) => todo.completed).length, [todos]);

    if (authLoading || !user) {
        return (
            <main className="flex min-h-screen items-center justify-center bg-[var(--color-surface)] px-6 py-10">
                <Card className="max-w-md text-center text-sm text-slate-600">ユーザー情報を確認しています...</Card>
            </main>
        );
    }

    return (
        <main className="min-h-screen bg-[var(--color-surface)] px-6 py-8">
            <div className="mx-auto flex w-full max-w-6xl flex-col gap-8">
                <PageHeader
                    eyebrow="Todo Dashboard"
                    title={`${user.name}さんのタスク一覧`}
                    description="一覧表示、作成、詳細画面への遷移、ログアウトをまとめた実務寄りの最小構成です。"
                    actions={<LogoutButton />}
                />

                <section className="grid gap-4 md:grid-cols-3">
                    <Card>
                        <p className="text-sm text-slate-500">総タスク数</p>
                        <p className="mt-3 text-3xl font-semibold text-slate-950">{todos.length}</p>
                    </Card>
                    <Card>
                        <p className="text-sm text-slate-500">完了済み</p>
                        <p className="mt-3 text-3xl font-semibold text-slate-950">{completedCount}</p>
                    </Card>
                    <Card>
                        <p className="text-sm text-slate-500">未完了</p>
                        <p className="mt-3 text-3xl font-semibold text-slate-950">{todos.length - completedCount}</p>
                    </Card>
                </section>

                <section className="grid gap-6 lg:grid-cols-[0.9fr_1.1fr]">
                    <Card className="h-fit">
                        <div className="flex items-center justify-between gap-4">
                            <div>
                                <p className="text-sm font-medium text-sky-700">作成ボタン</p>
                                <h2 className="mt-1 text-2xl font-semibold text-slate-950">新しい Todo を追加</h2>
                            </div>
                            <button
                                type="button"
                                onClick={() => setIsCreating((current) => !current)}
                                className="rounded-full bg-slate-950 px-5 py-3 text-sm font-semibold text-white transition hover:bg-slate-800"
                            >
                                {isCreating ? "フォームを閉じる" : "作成する"}
                            </button>
                        </div>

                        {isCreating ? (
                            <div className="mt-6">
                                <TodoForm
                                    mode="create"
                                    submitLabel="Todoを作成"
                                    onSubmit={async (input) => {
                                        await createTodo(input);
                                        setIsCreating(false);
                                    }}
                                />
                            </div>
                        ) : (
                            <p className="mt-6 text-sm leading-7 text-slate-600">
                                作成ボタンから追加した Todo は一覧と詳細画面で編集できます。完了状態は一覧から即時に切り替え可能です。
                            </p>
                        )}
                    </Card>

                    <Card>
                        <div className="flex items-center justify-between gap-4">
                            <div>
                                <p className="text-sm font-medium text-slate-500">一覧表示</p>
                                <h2 className="mt-1 text-2xl font-semibold text-slate-950">Todo 一覧</h2>
                            </div>
                            <span className="rounded-full bg-sky-50 px-3 py-1 text-xs font-semibold text-sky-700">
                                {loading ? "読み込み中" : `${todos.length} 件`}
                            </span>
                        </div>

                        <div className="mt-6">
                            {todos.length === 0 ? (
                                <EmptyState
                                    title="Todo がまだありません"
                                    description="まずは 1 件作成して、一覧と詳細編集の流れを確認してください。"
                                />
                            ) : (
                                <TodoList todos={todos} onToggle={toggleTodo} />
                            )}
                        </div>
                    </Card>
                </section>
            </div>
        </main>
    );
}